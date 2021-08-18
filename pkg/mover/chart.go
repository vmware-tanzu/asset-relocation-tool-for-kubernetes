// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/avast/retry-go"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/internal"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

const (
	defaultRetries        = 3
	EmbeddedHintsFilename = ".relok8s-images.yaml"
)

var ErrImageHintsMissing = errors.New("no image hints provided`")

type Logger interface {
	Printf(format string, i ...interface{})
	Println(i ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Printf(format string, i ...interface{}) {
	fmt.Printf(format, i...)
	return
}

func (l *defaultLogger) Println(i ...interface{}) {
	fmt.Println(i...)
	return
}

type ChartMover struct {
	chart *chart.Chart
	// Extracted metadata from the provided Helm Chart
	ChartName    string
	ChartVersion string
	imageChanges []*internal.ImageChange
	chartChanges []*internal.RewriteAction
	logger       Logger
	retries      uint
}

// Rewrite rules to be applied to the existing OCI images
type OCIImageRewriteRules struct {
	Registry         string
	RepositoryPrefix string
}

func loadPatterns(imageHintsFile string, chart *chart.Chart, log Logger) ([]byte, error) {
	var patternsRaw []byte
	var err error

	if imageHintsFile != "" {
		patternsRaw, err = loadPatternsFromFile(imageHintsFile, log)
		if err != nil {
			return nil, err
		}
	} else {
		// If patterns file is not provided we try to find the patterns from inside the Chart
		patternsRaw = loadPatternsFromChart(chart, log)
	}

	return patternsRaw, err
}

// Returns the chart embedded image patterns from the .relok8s-images.yaml file
func loadPatternsFromChart(chart *chart.Chart, log Logger) []byte {
	// TODO: This is an overkill, we know the location of the file
	// we should just check for it
	for _, file := range chart.Files {
		if file.Name == EmbeddedHintsFilename && file.Data != nil {
			log.Printf("%s hints file found\n", EmbeddedHintsFilename)
			return file.Data
		}
	}

	return nil
}

// loadPatternsFromFile from file first, or the embedded from the chart as a fallback
func loadPatternsFromFile(patternsFile string, log Logger) ([]byte, error) {
	contents, err := os.ReadFile(patternsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read the image patterns file: %w", err)
	}

	return contents, nil
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
func NewChartMover(chartPath string, imageHintsFile string, rules *OCIImageRewriteRules, opts ...Option) (*ChartMover, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load Helm Chart at %q: %w", chartPath, err)
	}

	c := &ChartMover{
		chart:        chart,
		ChartName:    chart.Name(),
		ChartVersion: chart.Metadata.Version,
		logger:       &defaultLogger{},
		retries:      defaultRetries,
	}

	// Option overrides
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	patternsRaw, err := loadPatterns(imageHintsFile, chart, c.logger)
	if err != nil {
		return nil, err
	}

	if patternsRaw == nil {
		return nil, ErrImageHintsMissing
	}

	imagePatterns, err := internal.ParseImagePatterns(patternsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image patterns: %w", err)
	}
	imageChanges, err := pullOriginalImages(chart, imagePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to pull original images: %w", err)
	}
	imageChanges, chartChanges, err := computeChanges(chart, imageChanges, rules)
	if err != nil {
		return nil, fmt.Errorf("failed to compute chart rewrites: %w", err)
	}

	c.imageChanges = imageChanges
	c.chartChanges = chartChanges

	return c, nil
}

// Print dumps the chart mover changes to the mover logger
func (cm *ChartMover) Print() {
	log := cm.logger
	log.Println("Image moves:")
	for _, change := range cm.imageChanges {
		pushRequiredTxt := ""
		if change.ShouldPush() {
			pushRequiredTxt = " (push required)"
		}
		log.Printf(" %s => %s (%s)%s\n",
			change.ImageReference.Name(), change.RewrittenReference.Name(), change.Digest, pushRequiredTxt)
	}

	var chartToModify *chart.Chart
	for _, change := range cm.chartChanges {
		destination := change.FindChartDestination(cm.chart)
		if chartToModify != destination {
			chartToModify = destination
			log.Printf("\nChanges to be applied to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}
		log.Printf("  %s: %s\n", change.Path, change.Value)
	}
	log.Println("")
}

// Move executes the chart move image and chart changes.
// The result chart is saved as toChartFilename.
func (cm *ChartMover) Move(toChartFilename string) error {
	log := cm.logger
	err := pushRewrittenImages(cm.imageChanges, cm.retries, log)
	if err != nil {
		return err
	}

	log.Println("Writing chart files...")
	err = modifyChart(cm.chart, cm.chartChanges, toChartFilename)
	if err != nil {
		return err
	}

	log.Println("Done\n")
	return nil
}

func pullOriginalImages(chart *chart.Chart, pattens []*internal.ImageTemplate) ([]*internal.ImageChange, error) {
	var changes []*internal.ImageChange
	imageCache := map[string]*internal.ImageChange{}

	for _, pattern := range pattens {
		originalImage, err := pattern.Render(chart)
		if err != nil {
			return nil, err
		}

		change := &internal.ImageChange{
			Pattern:        pattern,
			ImageReference: originalImage,
		}

		if imageCache[originalImage.Name()] == nil {
			image, digest, err := internal.Image.Pull(originalImage)
			if err != nil {
				return nil, err
			}
			change.Image = image
			change.Digest = digest
			imageCache[originalImage.Name()] = change
		} else {
			change.Image = imageCache[originalImage.Name()].Image
			change.Digest = imageCache[originalImage.Name()].Digest
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func computeChanges(chart *chart.Chart, imageChanges []*internal.ImageChange, registryRules *OCIImageRewriteRules) ([]*internal.ImageChange, []*internal.RewriteAction, error) {
	var chartChanges []*internal.RewriteAction
	imageCache := map[string]bool{}

	for _, change := range imageChanges {
		rewriteRules := internal.OCIImageLocation{
			Registry:         registryRules.Registry,
			RepositoryPrefix: registryRules.RepositoryPrefix,
			Digest:           change.Digest,
		}

		newActions, err := change.Pattern.Apply(change.ImageReference, &rewriteRules)
		if err != nil {
			return nil, nil, err
		}

		chartChanges = append(chartChanges, newActions...)

		rewrittenImage, err := change.Pattern.Render(chart, newActions...)
		if err != nil {
			return nil, nil, err
		}

		change.RewrittenReference = rewrittenImage

		if change.ShouldPush() {
			if imageCache[rewrittenImage.Name()] {
				// This image has already been checked previously, so just force this one to be skipped
				change.AlreadyPushed = true
			} else {
				needToPush, err := internal.Image.Check(change.Digest, rewrittenImage)
				if err != nil {
					return nil, nil, err
				}

				change.AlreadyPushed = !needToPush
				imageCache[rewrittenImage.Name()] = true
			}
		}
	}
	return imageChanges, chartChanges, nil
}

func pushRewrittenImages(imageChanges []*internal.ImageChange, retries uint, log Logger) error {
	for _, change := range imageChanges {
		if change.ShouldPush() {
			err := retry.Do(
				func() error {
					log.Printf("Pushing %s...\n", change.RewrittenReference.Name())
					err := internal.Image.Push(change.Image, change.RewrittenReference)
					if err != nil {
						return err
					}
					log.Println("Done")
					return nil
				},
				retry.Attempts(retries),
				retry.OnRetry(func(n uint, err error) {
					log.Printf("Attempt #%d failed: %s\n", n+1, err.Error())
				}),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func modifyChart(originalChart *chart.Chart, actions []*internal.RewriteAction, toChartFilename string) error {
	var err error
	modifiedChart := originalChart
	for _, action := range actions {
		modifiedChart, err = action.Apply(modifiedChart)
		if err != nil {
			return err
		}
	}

	return saveChart(modifiedChart, toChartFilename)
}

func saveChart(chart *chart.Chart, toChartFilename string) error {
	cwd, _ := os.Getwd()
	tempDir, err := ioutil.TempDir(cwd, "relok8s-*")
	if err != nil {
		return err
	}

	filename, err := chartutil.Save(chart, tempDir)
	if err != nil {
		return err
	}

	err = os.Rename(filename, toChartFilename)
	if err != nil {
		return err
	}

	return os.Remove(tempDir)
}

type Option func(*ChartMover)

// WithRetries customizes the mover push retries
func WithRetries(retries uint) Option {
	return func(c *ChartMover) {
		c.retries = retries
	}
}

func WithLogger(l Logger) Option {
	return func(c *ChartMover) {
		c.logger = l
	}
}
