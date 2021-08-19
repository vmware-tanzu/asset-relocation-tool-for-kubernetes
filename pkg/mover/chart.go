package mover

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/avast/retry-go"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/internal"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

const DefaultRetries = 3

type RewriteRules struct {
	Registry         string
	RepositoryPrefix string
}

type Logger interface {
	Printf(format string, i ...interface{})
}

type ChartMover struct {
	chart        *chart.Chart
	imageChanges []*internal.ImageChange
	chartChanges []*internal.RewriteAction
	logger       Logger
	retries      uint
}

// Returns the chart embedded image patterns from the .relok8s-images.yaml file
func ChartPatterns(chart *chart.Chart) string {
	for _, file := range chart.Files {
		if file.Name == ".relok8s-images.yaml" && file.Data != nil {
			return string(file.Data)
		}
	}
	return ""
}

// LoadImagePatterns from file first, or the embedded from the chart as a fallback
func LoadImagePatterns(patternsFile string, chart *chart.Chart) (string, error) {
	if patternsFile != "" {
		contents, err := os.ReadFile(patternsFile)
		if err != nil {
			return "", fmt.Errorf("failed to read the image patterns file: %w", err)
		}
		return string(contents), nil
	}
	return ChartPatterns(chart), nil
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
func NewChartMover(chart *chart.Chart, patterns string, rules *RewriteRules, log Logger) (*ChartMover, error) {
	imagePatterns, err := internal.ParseImagePatterns(patterns)
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
	return &ChartMover{
		chart:        chart,
		imageChanges: imageChanges,
		chartChanges: chartChanges,
		logger:       log,
		retries:      DefaultRetries,
	}, nil
}

// WithRetries customizes the mover push retries
func (cm *ChartMover) WithRetries(retries uint) *ChartMover {
	cm.retries = retries
	return cm
}

// Print dumps the chart mover changes to the mover logger
func (cm *ChartMover) Print() {
	log := cm.logger
	log.Printf("\nImage moves:\n")
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
			log.Printf("\nChanges written to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}
		log.Printf("  %s: %s\n", change.Path, change.Value)
	}
}

// Move executes the chart move image and chart changes.
// The result chart is saved as toChartFilename.
func (cm *ChartMover) Move(toChartFilename string) error {
	log := cm.logger
	err := pushRewrittenImages(cm.imageChanges, cm.retries, log)
	if err != nil {
		return err
	}
	log.Printf("Writing chart files... ")
	err = modifyChart(cm.chart, cm.chartChanges, toChartFilename)
	if err != nil {
		return err
	}
	log.Printf("Done\n")
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

func computeChanges(chart *chart.Chart, imageChanges []*internal.ImageChange, registryRules *RewriteRules) ([]*internal.ImageChange, []*internal.RewriteAction, error) {
	var chartChanges []*internal.RewriteAction
	imageCache := map[string]bool{}

	for _, change := range imageChanges {
		rewriteRules := &internal.OCIImageLocation{
			Registry:         registryRules.Registry,
			RepositoryPrefix: registryRules.RepositoryPrefix,
			Digest:           change.Digest,
		}

		newActions, err := change.Pattern.Apply(change.ImageReference, rewriteRules)
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

				if needToPush {
				} else {
					change.AlreadyPushed = true
				}
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
					log.Printf("Done\n")
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
