// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/avast/retry-go"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
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

	// defaultTarPermissions
	defaultTarPermissions = 0750

	// HintsFilename to be present in the offline archive root path
	HintsFilename = "hints.yaml"
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

func (l defaultLogger) Printf(format string, i ...interface{}) {
	fmt.Printf(format, i...)
}

func (l defaultLogger) Println(i ...interface{}) {
	fmt.Println(i...)
}

type noLogger struct{}

func (nl noLogger) Printf(format string, i ...interface{}) {}

func (nl noLogger) Println(i ...interface{}) {}

// DefaultLogger to stdout
var DefaultLogger Logger = defaultLogger{}

// DefaultNoLogger swallows all logs
var NoLogger Logger = noLogger{}

// ChartMetadata exposes metadata about the Helm Chart to be relocated
type ChartMetadata struct {
	Name    string
	Version string
}

// LocalChart is a reference to a local chart
type LocalChart struct {
	Path string
}

// OfflineArchive is a self contained version of a chart including
// container images within and the hints file
type OfflineArchive LocalChart

// ContainerRepository defines a private repo name and credentials
type ContainerRepository struct {
	Server             string
	Username, Password string
}

// Containers is the section for private repository definition
type Containers struct {
	ContainerRepository
}

// ChartSpec of possible chart inputs or outputs
type ChartSpec struct {
	Local   *LocalChart
	Archive *OfflineArchive
}

// Source of the chart move
type Source struct {
	Chart          ChartSpec
	ImageHintsFile string
	Containers     Containers
}

// Target of the chart move
type Target struct {
	Chart      ChartSpec
	Rules      RewriteRules
	Containers Containers
}

// ChartMoveRequest defines a chart move
type ChartMoveRequest struct {
	Source Source
	Target Target
}

// ChartMover represents a Helm Chart moving relocation. It's initialization must be done view NewChartMover
type ChartMover struct {
	chartDestination        string
	imageChanges            []*internal.ImageChange
	chartChanges            []*internal.RewriteAction
	sourceContainerRegistry internal.ContainerRegistryInterface
	targetContainerRegistry internal.ContainerRegistryInterface
	targetOfflineTar        string
	chart                   *chart.Chart
	logger                  Logger
	retries                 uint
	rawHints                []byte
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
func NewChartMover(req *ChartMoveRequest, opts ...Option) (*ChartMover, error) {
	chart, err := loader.Load(req.Source.Chart.Local.Path)
	if err != nil {
		return nil, &ChartLoadingError{Path: req.Source.Chart.Local.Path, Inner: err}
	}

	if err := validateTarget(&req.Target); err != nil {
		return nil, err
	}

	sourceAuth := req.Source.Containers.ContainerRepository
	targetAuth := req.Target.Containers.ContainerRepository
	cm := &ChartMover{
		chart:                   chart,
		logger:                  defaultLogger{},
		retries:                 DefaultRetries,
		sourceContainerRegistry: internal.NewContainerRegistryClient(sourceAuth),
		targetContainerRegistry: internal.NewContainerRegistryClient(targetAuth),
	}
	if req.Target.Chart.Archive != nil {
		cm.targetOfflineTar = req.Target.Chart.Archive.Path
	}
	if req.Target.Chart.Local != nil {
		cm.chartDestination = targetOutput(req.Target.Chart.Local.Path, chart.Name(), chart.Metadata.Version)
	}

	// Option overrides
	for _, opt := range opts {
		if opt != nil {
			opt(cm)
		}
	}

	patternsRaw, err := loadPatterns(req.Source.ImageHintsFile, chart, cm.logger)
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

	cm.logger.Println("Computing relocation...\n")

	imageChanges, err := cm.pullOriginalImages(imagePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to pull original images: %w", err)
	}

	imageChanges, chartChanges, err := cm.computeChanges(imageChanges, &req.Target.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to compute chart rewrites: %w", err)
	}

	cm.rawHints = patternsRaw
	cm.imageChanges = imageChanges
	cm.chartChanges = chartChanges

	return cm, nil
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
	if cm.targetOfflineTar != "" {
		cm.printSaveOfflineBundle()
		return
	}
	cm.printMove()
}

func (cm *ChartMover) printSaveOfflineBundle() {
	log := cm.logger
	log.Printf("Will archive Helm Chart %s@%s, dependent images and hint file to offline tarball %s\n",
		cm.chart.Metadata.Name, cm.chart.Metadata.Version, cm.targetOfflineTar)
	names := map[string]bool{}
	for _, change := range cm.imageChanges {
		app := change.ImageReference.Context().Name()
		version := change.ImageReference.Identifier()
		fullImageName := fmt.Sprintf("%s:%s", app, version)
		names[fullImageName] = true
	}

	log.Printf("%d images detected to be archived:\n", len(names))
	for name := range names {
		log.Printf("%s\n", name)
	}
}

func (cm *ChartMover) printMove() {
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
  Move performs the relocation.

A regular move executes the Chart relocation which includes

1 - Push all the images to their new locations

2 - Rewrite the Helm Chart and its subcharts

3 - Repackage the Helm chart as toChartFilename

A save to an offline tarball bundle will:

1 - Drop all images to disk, with the original chart (unpacked) and hints file

2 - Package all in a single compressed tarball
*/
func (cm *ChartMover) Move() error {
	if cm.targetOfflineTar != "" {
		return cm.saveOfflineBundle()
	}
	return cm.moveChart()
}

func (cm *ChartMover) saveOfflineBundle() error {
	log := cm.logger

	tarPath, err := os.MkdirTemp("", "offline-tarball-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory to build tar: %w", err)
	}
	log.Printf("Writing chart at %s/...\n", cm.chart.Metadata.Name)
	if err := writeChart(cm.chart, filepath.Join(tarPath, cm.chart.Metadata.Name)); err != nil {
		return fmt.Errorf("failed archiving chart %s: %w", cm.chart.Name(), err)
	}
	if err := packImages(tarPath, cm.imageChanges, cm.logger); err != nil {
		return fmt.Errorf("failed archiving images: %w", err)
	}
	log.Printf("Writing hints file %s...\n", HintsFilename)
	if err := os.WriteFile(filepath.Join(tarPath, HintsFilename), cm.rawHints, defaultTarPermissions); err != nil {
		return fmt.Errorf("failed to write hints file: %w", err)
	}
	log.Printf("Packing all as tarball %s...\n", cm.targetOfflineTar)
	if err := tarDirectory(tarPath, cm.targetOfflineTar); err != nil {
		return fmt.Errorf("failed to tar bundle as %s: %w", cm.targetOfflineTar, err)
	}
	return os.RemoveAll(tarPath)
}

func writeChart(chart *chart.Chart, targetDir string) error {
	if err := os.MkdirAll(targetDir, defaultTarPermissions); err != nil {
		return fmt.Errorf("failed to create target chart %s: %w", targetDir, err)
	}
	for _, file := range chart.Raw {
		dir := filepath.Dir(file.Name)
		dirPath := filepath.Join(targetDir, dir)
		if err := os.MkdirAll(dirPath, defaultTarPermissions); err != nil {
			return fmt.Errorf("failed to create path %s: %w", dirPath, err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, file.Name), file.Data, defaultTarPermissions); err != nil {
			return fmt.Errorf("failed to write chart's inner file %s: %v", file.Name, err)
		}
	}
	return nil
}

func packImages(archivePath string, imageChanges []*internal.ImageChange, logger Logger) error {
	imagesTarball := filepath.Join(archivePath, "images.tar")
	tagToImage := map[name.Tag]v1.Image{}
	for _, change := range imageChanges {
		app := change.ImageReference.Context().Name()
		version := change.ImageReference.Identifier()
		fullImageName := fmt.Sprintf("%s:%s", app, version)
		tag, err := name.NewTag(fullImageName)
		if err != nil {
			return fmt.Errorf("failed to create tag %q: %v", fullImageName, err)
		}
		if previousImage, ok := tagToImage[tag]; ok {
			if err := deduplicateByDigest(fullImageName, change.Image, previousImage); err != nil {
				return err
			}
			continue
		}
		tagToImage[tag] = change.Image
		logger.Printf("Processing image %s\n", fullImageName)
	}
	logger.Printf("Packing all %d images within images.tar...\n", len(tagToImage))
	if err := tarball.MultiWriteToFile(imagesTarball, tagToImage); err != nil {
		return err
	}
	return nil
}

func deduplicateByDigest(name string, current, previous v1.Image) error {
	previousDigest, err := previous.Digest()
	if err != nil {
		return fmt.Errorf("failed to check previous image digest: %v", err)
	}
	imageDigest, err := current.Digest()
	if err != nil {
		return fmt.Errorf("failed to check current image digest: %v", err)
	}
	if previousDigest != imageDigest {
		return fmt.Errorf("found image %q with different digests %s vs %s",
			name, previousDigest, imageDigest)
	}
	return nil
}

func (cm *ChartMover) moveChart() error {
	log := cm.logger
	log.Printf("Relocating %s@%s...\n", cm.chart.Name(), cm.chart.Metadata.Version)

	err := cm.pushRewrittenImages(cm.imageChanges)
	if err != nil {
		return err
	}
	err = modifyChart(cm.chart, cm.chartChanges, cm.chartDestination)
	if err != nil {
		return err
	}

	log.Println("Done")
	log.Println(cm.chartDestination)
	return nil
}

// validateTarget ensures the requested Target has expected inputs.
// If the archival target is not set, at least one transformation rule must be set
func validateTarget(target *Target) error {
	if target.Chart.Archive != nil {
		return nil
	}
	rules := target.Rules
	if rules.Registry == "" && rules.RepositoryPrefix == "" {
		return ErrOCIRewritesMissing
	}
	return nil
}

func (cm *ChartMover) pullOriginalImages(pattens []*internal.ImageTemplate) ([]*internal.ImageChange, error) {
	var changes []*internal.ImageChange
	imageCache := map[string]*internal.ImageChange{}

	for _, pattern := range pattens {
		originalImage, err := pattern.Render(cm.chart)
		if err != nil {
			return nil, err
		}

		change := &internal.ImageChange{
			Pattern:        pattern,
			ImageReference: originalImage,
		}

		if imageCache[originalImage.Name()] == nil {
			image, digest, err := cm.sourceContainerRegistry.Pull(originalImage)
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

func (cm *ChartMover) computeChanges(imageChanges []*internal.ImageChange, registryRules *RewriteRules) ([]*internal.ImageChange, []*internal.RewriteAction, error) {
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

		rewrittenImage, err := change.Pattern.Render(cm.chart, newActions...)
		if err != nil {
			return nil, nil, err
		}

		change.RewrittenReference = rewrittenImage

		if change.ShouldPush() {
			if imageCache[rewrittenImage.Name()] {
				// This image has already been checked previously, so just force this one to be skipped
				change.AlreadyPushed = true
			} else {
				needToPush, err := cm.targetContainerRegistry.Check(change.Digest, rewrittenImage)
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

func (cm *ChartMover) pushRewrittenImages(imageChanges []*internal.ImageChange) error {
	for _, change := range imageChanges {
		if change.ShouldPush() {
			err := retry.Do(
				func() error {
					cm.logger.Printf("Pushing %s...\n", change.RewrittenReference.Name())
					err := cm.targetContainerRegistry.Push(change.Image, change.RewrittenReference)
					if err != nil {
						return err
					}
					cm.logger.Println("Done")
					return nil
				},
				retry.Attempts(cm.retries),
				retry.OnRetry(func(n uint, err error) {
					cm.logger.Printf("Attempt #%d failed: %s\n", n+1, err.Error())
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

func targetOutput(targetFormat, name, version string) string {
	return fmt.Sprintf(targetFormat, name, version)
}
