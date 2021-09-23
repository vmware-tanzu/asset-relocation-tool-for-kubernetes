// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/avast/retry-go"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

const (
	// EmbeddedHintsFilename to be present in the Helm Chart rootpath
	EmbeddedHintsFilename = ".relok8s-images.yaml"
	// DefaultRetries indicates the default number of retries for pull/push operations
	DefaultRetries = 3
)

var (
	// ErrImageHintsMissing indicates that neither the hints file was provided nor found in the Helm chart
	ErrImageHintsMissing = errors.New("no image hints provided")
	// ErrOCIRewritesMissing indicates that no rewrite rules have been provided
	ErrOCIRewritesMissing = errors.New("at least one rewrite rule is required")
)

type ChartLoadingError struct {
	Path  string
	Inner error
}

func (e *ChartLoadingError) Error() string {
	return fmt.Sprintf("failed to load Helm Chart at %q: %s", e.Path, e.Inner.Error())
}

func (e *ChartLoadingError) Unwrap() error {
	return e.Inner
}

// Logger represents an interface used to output moving information
type Logger interface {
	Printf(format string, i ...interface{})
	Println(i ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Printf(format string, i ...interface{}) {
	fmt.Printf(format, i...)
}

func (l *defaultLogger) Println(i ...interface{}) {
	fmt.Println(i...)
}

// ChartMover represents a Helm Chart moving relocation. It's initialization must be done view NewChartMover
type ChartMover struct {
	chart        *chart.Chart
	imageChanges []*internal.ImageChange
	chartChanges []*internal.RewriteAction
	logger       Logger
	retries      uint
}

// ChartMetadata exposes metadata about the Helm Chart to be relocated
type ChartMetadata struct {
	Name    string
	Version string
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
func NewChartMover(chartPath string, imageHintsFile string, rules *RewriteRules, opts ...Option) (*ChartMover, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, &ChartLoadingError{Path: chartPath, Inner: err}
	}

	if rules.Registry == "" && rules.RepositoryPrefix == "" {
		return nil, ErrOCIRewritesMissing
	}

	c := &ChartMover{
		chart:   chart,
		logger:  &defaultLogger{},
		retries: DefaultRetries,
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

	c.logger.Println("Computing relocation...\n")

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

// WithRetries sets how many times to retry push operations
func (cm *ChartMover) WithRetries(retries uint) *ChartMover {
	cm.retries = retries
	return cm
}

// Print shows the changes expected to be performed during relocation,
// including the new location of the Helm Chart Images as well as
// the expected rewrites in the Helm Chart.
func (cm *ChartMover) Print() {
	log := cm.logger
	log.Println("Image copies:")
	for _, change := range cm.imageChanges {
		pushRequiredTxt := "already exists"
		if change.ShouldPush() {
			pushRequiredTxt = "push required"
		}
		log.Printf(" %s => %s (%s) (%s)\n",
			change.ImageReference.Name(), change.RewrittenReference.Name(), change.Digest, pushRequiredTxt)
	}

	var chartToModify *chart.Chart
	for _, change := range cm.chartChanges {
		destination := change.FindChartDestination(cm.chart)
		if chartToModify != destination {
			chartToModify = destination
			log.Printf("\nChanges to be applied to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}

		// Remove chart name from the path since we are already indicating what values.yaml file we are changing
		log.Printf("  %s: %s\n", namespacedPath(change.Path, chartToModify.Name()), change.Value)
	}
}

// namespacedPath removes the chartName from the beginning of the full path
// i.e .mariadb.image.registry => .image.registry if the chartName is mariadb
func namespacedPath(fullpath, chartName string) string {
	re := regexp.MustCompile(fmt.Sprintf("^.%s.", chartName))
	return re.ReplaceAllString(fullpath, ".")
}

/*
Move executes the Chart relocation which includes

1 - Push all the images to their new locations

2 - Rewrite the Helm Chart and its subcharts

3 - Repackage the Helm chart as toChartFilename
*/
func (cm *ChartMover) Move(toChartFilename string) error {
	log := cm.logger

	log.Printf("Relocating %s@%s...\n", cm.chart.Name(), cm.chart.Metadata.Version)

	err := pushRewrittenImages(cm.imageChanges, cm.retries, log)
	if err != nil {
		return err
	}
	err = modifyChart(cm.chart, cm.chartChanges, toChartFilename)
	if err != nil {
		return err
	}

	log.Println("Done")
	log.Println(toChartFilename)
	return nil
}

// ChartMetadata returns information from the Helm Chart ready to be relocated
func (cm *ChartMover) ChartMetadata() (*ChartMetadata, error) {
	if cm.chart == nil {
		return nil, errors.New("Helm Chart not loaded")
	}

	return &ChartMetadata{
		Name:    cm.chart.Name(),
		Version: cm.chart.Metadata.Version,
	}, nil
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

func computeChanges(chart *chart.Chart, imageChanges []*internal.ImageChange, registryRules *RewriteRules) ([]*internal.ImageChange, []*internal.RewriteAction, error) {
	var chartChanges []*internal.RewriteAction
	imageCache := map[string]bool{}

	for _, change := range imageChanges {
		rewriteRules := &internal.OCIImageLocation{
			Registry:         registryRules.Registry,
			RepositoryPrefix: registryRules.RepositoryPrefix,
		}

		newActions, err := change.Pattern.Apply(change.ImageReference.Context(), change.Digest, rewriteRules)
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

// load patterns from either a hints file or an existing EmbeddedHintsFilename
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

// Option adds optional functionality to NewChartMover constructor
type Option func(*ChartMover)

// WithRetries defines how many times to retry the push operation
func WithRetries(retries uint) Option {
	return func(c *ChartMover) {
		c.retries = retries
	}
}

// WithLogger sets a custom Logger interface
func WithLogger(l Logger) Option {
	return func(c *ChartMover) {
		c.logger = l
	}
}
