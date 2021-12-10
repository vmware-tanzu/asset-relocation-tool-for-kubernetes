// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/avast/retry-go"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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

	// IntermediateBundleHintsFilename to be present in the intermediate archive root path
	IntermediateBundleHintsFilename = "hints.yaml"
)

var (
	// ErrImageHintsMissing indicates that neither the hints file was provided nor found in the Helm chart
	ErrImageHintsMissing = errors.New("no image hints provided")

	// ErrOCIRewritesMissing indicates that no rewrite rules have been provided
	ErrOCIRewritesMissing = errors.New("at least one rewrite rule is required")

	// ErrDuplicateChartFiles found in the input chart
	ErrDuplicateChartFiles = errors.New("duplicated chart files found")
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

// IntermediateBundle is a self contained version of the original chart with
// the hints file and container images
type IntermediateBundle LocalChart

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
	Local              *LocalChart
	IntermediateBundle *IntermediateBundle
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
	chartDestination          string
	imageChanges              []*internal.ImageChange
	chartChanges              []*internal.RewriteAction
	sourceContainerRegistry   internal.ContainerRegistryInterface
	targetContainerRegistry   internal.ContainerRegistryInterface
	targetIntermediateTarPath string
	chart                     *chart.Chart
	logger                    Logger
	retries                   uint
	intermediateBundle        *intermediateBundle
	// raw contents of the hints file. Sample:
	// test/fixtures/testchart.images.yaml
	rawHints []byte
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
func NewChartMover(req *ChartMoveRequest, opts ...Option) (*ChartMover, error) {
	sourceAuth := req.Source.Containers.ContainerRepository
	targetAuth := req.Target.Containers.ContainerRepository
	cm := &ChartMover{
		logger:                  defaultLogger{},
		retries:                 DefaultRetries,
		sourceContainerRegistry: internal.NewContainerRegistryClient(sourceAuth),
		targetContainerRegistry: internal.NewContainerRegistryClient(targetAuth),
	}

	if err := validateTarget(&req.Target); err != nil {
		return nil, err
	}

	if err := cm.loadChart(&req.Source); err != nil {
		if !errors.Is(err, ErrDuplicateChartFiles) {
			return nil, err
		}
		cm.logger.Printf("Warning: %v", err)
	}

	if req.Target.Chart.IntermediateBundle != nil {
		cm.targetIntermediateTarPath = req.Target.Chart.IntermediateBundle.Path
	} else if req.Target.Chart.Local != nil {
		cm.chartDestination =
			targetOutput(req.Target.Chart.Local.Path, cm.chart.Name(), cm.chart.Metadata.Version)
	}

	// Option overrides
	for _, opt := range opts {
		if opt != nil {
			opt(cm)
		}
	}

	if err := cm.loadImageHints(&req.Source); err != nil {
		return nil, fmt.Errorf("failed to load hints file: %w", err)
	}

	imagePatterns, err := internal.ParseImagePatterns(cm.rawHints)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image patterns: %w", err)
	}

	cm.logger.Println("Computing relocation...\n")
	imageChanges, err := cm.loadOriginalImages(imagePatterns)
	if err != nil {
		return nil, err
	}

	imageChanges, chartChanges, err := cm.computeChanges(imageChanges, &req.Target.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to compute chart rewrites: %w", err)
	}

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
	if cm.targetIntermediateTarPath != "" {
		cm.printSaveIntermediateBundle()
		return
	}
	cm.printMove()
}

// loadChart loads the chart in memory from the intermediate bundle or a given path
func (cm *ChartMover) loadChart(src *Source) error {
	if src.Chart.Local != nil {
		return cm.loadChartFromPath(src.Chart.Local.Path)
	} else if src.Chart.IntermediateBundle != nil {
		return cm.loadChartFromIntermediateBundle(src.Chart.IntermediateBundle.Path)
	}
	return fmt.Errorf("must provide either a local chart or an intermediate bundle as input")
}

// loadChartFromIntermediateBundle loads the chart in memory after extracting
// its files from the bundle into a temporary directory
func (cm *ChartMover) loadChartFromIntermediateBundle(bundlePath string) error {
	cm.intermediateBundle = newBundle(bundlePath)

	chartPath, err := os.MkdirTemp("", "bundle-chart-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory to extract bundled chart %s: %w",
			cm.intermediateBundle.bundlePath, err)
	}
	defer os.RemoveAll(chartPath)

	if err := cm.intermediateBundle.extractChartTo(chartPath); err != nil {
		return err
	}

	return cm.loadChartFromPath(chartPath)
}

// loadChartFromPath load the chart in memory from a given path
func (cm *ChartMover) loadChartFromPath(path string) error {
	var err error
	var chart *chart.Chart
	if chart, err = loader.Load(path); err != nil {
		return &ChartLoadingError{Path: path, Inner: err}
	}
	cm.chart, err = deduplicateChartFiles(chart)
	return err
}

// deduplicateChartFiles finds and fixes duplicate files at inchart.
// If duplicates are found an error is returned, the caller might want to
// report and proceed anyway as the output chat is clean of duplicates.
func deduplicateChartFiles(inchart *chart.Chart) (*chart.Chart, error) {
	nameStats := map[string]int{}
	deduplicated := []*chart.File{}
	for _, file := range inchart.Raw {
		times := nameStats[file.Name]
		times++
		nameStats[file.Name] = times
		if times == 1 {
			deduplicated = append(deduplicated, file)
		}
	}
	duplicatesFound := len(deduplicated) < len(inchart.Raw)
	if duplicatesFound {
		summary := duplicatesSummary(nameStats)
		outchart, err := loader.LoadFiles(bufferedFiles(deduplicated))
		if err != nil {
			return outchart, err
		}
		return outchart, fmt.Errorf("%w:\n%s", ErrDuplicateChartFiles, summary)
	}
	return inchart, nil
}

// duplicatesSummary dumps a line per duplicate file with more than one occurrence
func duplicatesSummary(nameStats map[string]int) string {
	sb := &strings.Builder{}
	for name, times := range nameStats {
		if times > 1 {
			fmt.Fprintf(sb, "%s appears %d times", name, times)
		}
	}
	return sb.String()
}

// bufferedFiles converts a list of chart.File to chartBuffered.File
func bufferedFiles(files []*chart.File) []*loader.BufferedFile {
	bufFiles := []*loader.BufferedFile{}
	for _, file := range files {
		bufFiles = append(bufFiles, &loader.BufferedFile{Name: file.Name, Data: file.Data})
	}
	return bufFiles
}

// loadImageHints loads the image hints in memory.
// Uses loadImageHintsFromBundle or loadImageHintsFromFileOrChart.
func (cm *ChartMover) loadImageHints(src *Source) error {
	if src.Chart.IntermediateBundle != nil {
		if src.ImageHintsFile != "" {
			return fmt.Errorf("do not set a hints filename, the bundle already provides it")
		}
		if err := cm.loadImageHintsFromBundle(); err != nil {
			return err
		}
	} else if err := cm.loadImageHintsFromFileOrChart(src.ImageHintsFile); err != nil {
		return err
	}
	if cm.rawHints == nil {
		return ErrImageHintsMissing
	}
	return nil
}

// loadImageHintsFromBundle loads the image hints from the intermediate bundle
func (cm *ChartMover) loadImageHintsFromBundle() error {
	rawHints, err := cm.intermediateBundle.loadImageHints(cm.logger)
	if err != nil {
		return err
	}
	cm.rawHints = rawHints
	return nil
}

// loadImageHintsFromFileOrChart loads the image hints from a given file or the
// chart, if the hints file is present inside.
func (cm *ChartMover) loadImageHintsFromFileOrChart(imageHintsFile string) error {
	rawHints, err := loadImageHints(imageHintsFile, cm.chart, cm.logger)
	if err != nil {
		return err
	}
	cm.rawHints = rawHints
	return nil
}

func (cm *ChartMover) printSaveIntermediateBundle() {
	log := cm.logger
	log.Printf("Will archive Helm Chart %s@%s, dependent images and hint file to intermediate tarball %q\n",
		cm.chart.Metadata.Name, cm.chart.Metadata.Version, cm.targetIntermediateTarPath)
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
		src := change.ImageReference.Name()
		if cm.intermediateBundle != nil {
			src = fmt.Sprintf("(bundle %s:%s)", cm.intermediateBundle.bundlePath, src)
		}
		log.Printf(" %s => %s (%s) (%s)\n",
			src, change.RewrittenReference.Name(), change.Digest, pushRequiredTxt)
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
- Push all the images to their new locations
- Rewrite the Helm Chart and its subcharts
- Repackage the Helm chart as toChartFilename

A save to an offline tarball bundle will:
- Drop all images to disk, with the original chart (unpacked) and hints file
- Package all in a single compressed tarball
*/
func (cm *ChartMover) Move() error {
	if cm.targetIntermediateTarPath != "" {
		bcd := &bundledChartData{
			chart:        cm.chart,
			imageChanges: cm.imageChanges,
			rawHints:     cm.rawHints,
		}
		return saveIntermediateBundle(bcd, cm.targetIntermediateTarPath, cm.logger)
	}
	return cm.moveChart()
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

	log.Println("Done moving", cm.chartDestination)
	return nil
}

// validateTarget ensures the requested Target has expected inputs.
// If the archival target is not set, at least one transformation rule must be set
func validateTarget(target *Target) error {
	if target.Chart.IntermediateBundle != nil {
		return nil
	}
	rules := target.Rules
	if rules.Registry == "" && rules.RepositoryPrefix == "" {
		return ErrOCIRewritesMissing
	}
	return nil
}

// imageLoadFn defines how an image is loaded
type imageLoadFn func(name.Reference) (v1.Image, string, error)

// loadOriginalImages will load container images from a remote registry or a local intermediate bundle.
// The heavy lifting is done by loadImageChanges, but here the actual image load
// function is selected.
func (cm *ChartMover) loadOriginalImages(imagePatterns []*internal.ImageTemplate) ([]*internal.ImageChange, error) {
	loadFn := func(originalImage name.Reference) (v1.Image, string, error) {
		return cm.sourceContainerRegistry.Pull(originalImage)
	}
	action := "pull"
	if cm.intermediateBundle != nil {
		loadFn = func(originalImage name.Reference) (v1.Image, string, error) {
			return cm.intermediateBundle.loadImage(originalImage)
		}
		action = "load"
	}
	imageChanges, err := loadImageChanges(cm.chart, imagePatterns, loadFn)
	if err != nil {
		return nil, fmt.Errorf("failed to %s original images: %w", action, err)
	}
	return imageChanges, nil
}

// loadImageChanges loads images from a loader function load and wraps them as
// ImageChange appropriately. As the load function is abstracted away this
// can be loading remote or local images the same way.
func loadImageChanges(chart *chart.Chart, patterns []*internal.ImageTemplate, load imageLoadFn) ([]*internal.ImageChange, error) {
	var changes []*internal.ImageChange
	imageCache := map[string]*internal.ImageChange{}

	for _, pattern := range patterns {
		originalImage, err := pattern.Render(chart)
		if err != nil {
			return nil, err
		}

		change := &internal.ImageChange{
			Pattern:        pattern,
			ImageReference: originalImage,
		}

		if imageCache[originalImage.Name()] == nil {
			image, digest, err := load(originalImage)
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
	rewriteRules := &internal.OCIImageLocation{
		Registry:         registryRules.Registry,
		RepositoryPrefix: registryRules.RepositoryPrefix,
	}

	for _, change := range imageChanges {
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
				// If ForcePush is set we add it to the list of changes to be performed regardless
				if !registryRules.ForcePush {
					needToPush, err := cm.targetContainerRegistry.Check(change.Digest, rewrittenImage)
					if err != nil {
						return nil, nil, fmt.Errorf("failed check, use forcePush option to override :%w", err)
					}
					change.AlreadyPushed = !needToPush
				}

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

// load hints from either a given hints file or a chart-embedded hints file
func loadImageHints(imageHintsFile string, chart *chart.Chart, log Logger) ([]byte, error) {
	if imageHintsFile != "" {
		rawHints, err := notNilData(loadImageHintsFromFile(imageHintsFile, log))
		if err != nil {
			return nil, err
		}
		return rawHints, nil
	}
	// If the hints file is not provided, try to find the hints inside the Chart
	return loadImageHintsFromChart(chart, log)
}

func loadImageHintsFromChart(chart *chart.Chart, log Logger) ([]byte, error) {
	// We get the file form the parsed chart object, otherwise the chart might
	// have come from a tar or tgz, so its files might not be directly available
	// on disk at this point.
	// In the general case, retrieving the hints file from disk is more work.
	for _, file := range chart.Files {
		if file.Name == EmbeddedHintsFilename {
			if file.Data == nil {
				return nil, errors.New("empty hints file in chart")
			}
			log.Printf("%s hints file found\n", EmbeddedHintsFilename)
			return file.Data, nil
		}
	}
	return nil, nil
}

// loadImageHintsFromFile from a file
func loadImageHintsFromFile(hintsFile string, log Logger) ([]byte, error) {
	contents, err := notNilData(os.ReadFile(hintsFile))
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

func notNilData(data []byte, err error) ([]byte, error) {
	if err == nil && data == nil {
		return nil, errors.New("no data loaded")
	}
	return data, err
}
