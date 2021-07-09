package mover

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/avast/retry-go"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/internal"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

type Printer interface {
	Print(i ...interface{})
	Println(i ...interface{})
	Printf(format string, i ...interface{})
	PrintErr(i ...interface{})
	PrintErrln(i ...interface{})
	PrintErrf(format string, i ...interface{})
}

type ChartMover struct {
	Chart        *chart.Chart
	ImageChanges []*ImageChange
	ChartChanges []*RewriteAction
}

type ImageChange struct {
	Pattern            *ImageTemplate
	ImageReference     name.Reference
	RewrittenReference name.Reference
	Image              v1.Image
	Digest             string
	AlreadyPushed      bool
}

func (change *ImageChange) ShouldPush() bool {
	return !change.AlreadyPushed && change.ImageReference.Name() != change.RewrittenReference.Name()
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
// TODO: Can/should we make this not need a logger as a input?
func NewChartMover(chart *chart.Chart, imagePatterns []*ImageTemplate, rules *RewriteRules, log Printer) (*ChartMover, error) {
	imageChanges, err := PullOriginalImages(chart, imagePatterns, log)
	if err != nil {
		return nil, err
	}
	return CheckNewImages(chart, imageChanges, rules, log)
}

// Print dumps the chart mover changes to the given logger
func (rl *ChartMover) Print(log Printer) {
	log.Println("\nImages to be pushed:")
	noImagesToPush := true
	for _, change := range rl.ImageChanges {
		if change.ShouldPush() {
			log.Printf("  %s (%s)\n", change.RewrittenReference.Name(), change.Digest)
			noImagesToPush = false
		}
	}
	if noImagesToPush {
		log.Printf("  no images require pushing")
	}

	var chartToModify *chart.Chart
	for _, change := range rl.ChartChanges {
		destination := change.FindChartDestination(rl.Chart)
		if chartToModify != destination {
			chartToModify = destination
			log.Printf("\nChanges written to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}
		log.Printf("  %s: %s\n", change.Path, change.Value)
	}
}

// Apply executes the chart move image and chart changes
func (rl *ChartMover) Apply(outputFmt string, retries uint, log Printer) error {
	err := PushRewrittenImages(rl.ImageChanges, retries, log)
	if err != nil {
		return err
	}
	log.Print("Writing chart files... ")
	err = modifyChart(rl.Chart, rl.ChartChanges, outputFmt)
	if err != nil {
		log.Println("")
		return err
	}
	log.Println("Done")
	return nil
}

// PullOriginalImages takes the chart and image patters to pull all images
// and compute the image changes for a move
func PullOriginalImages(chart *chart.Chart, pattens []*ImageTemplate, log Printer) ([]*ImageChange, error) {
	var changes []*ImageChange
	imageCache := map[string]*ImageChange{}

	for _, pattern := range pattens {
		originalImage, err := pattern.Render(chart)
		if err != nil {
			return nil, err
		}

		change := &ImageChange{
			Pattern:        pattern,
			ImageReference: originalImage,
		}

		if imageCache[originalImage.Name()] == nil {
			log.Printf("Pulling %s... ", originalImage.Name())
			image, digest, err := internal.Image.Pull(originalImage)
			if err != nil {
				log.Println("")
				return nil, err
			}
			log.Println("Done")
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

// CheckNewImages creates a ChartMover from a chart, image changes and rules
func CheckNewImages(chart *chart.Chart, imageChanges []*ImageChange, rules *RewriteRules, log Printer) (*ChartMover, error) {
	var chartChanges []*RewriteAction
	imageCache := map[string]bool{}

	for _, change := range imageChanges {
		rules.Digest = change.Digest
		newActions, err := change.Pattern.Apply(change.ImageReference, rules)
		if err != nil {
			return nil, err
		}

		chartChanges = append(chartChanges, newActions...)

		rewrittenImage, err := change.Pattern.Render(chart, newActions...)
		if err != nil {
			return nil, err
		}

		change.RewrittenReference = rewrittenImage

		if change.ShouldPush() {
			if imageCache[rewrittenImage.Name()] {
				// This image has already been checked previously, so just force this one to be skipped
				change.AlreadyPushed = true
			} else {
				log.Printf("Checking %s (%s)... ", rewrittenImage.Name(), change.Digest)
				needToPush, err := internal.Image.Check(change.Digest, rewrittenImage)
				if err != nil {
					log.Println("")
					return nil, err
				}

				if needToPush {
					log.Println("Push required")
				} else {
					log.Println("Already exists")
					change.AlreadyPushed = true
				}
				imageCache[rewrittenImage.Name()] = true
			}
		}
	}
	return &ChartMover{Chart: chart, ImageChanges: imageChanges, ChartChanges: chartChanges}, nil
}

// PushRewrittenImages processes all image changes pushing to the target locations
func PushRewrittenImages(imageChanges []*ImageChange, retries uint, log Printer) error {
	for _, change := range imageChanges {
		if change.ShouldPush() {
			err := retry.Do(
				func() error {
					log.Printf("Pushing %s... ", change.RewrittenReference.Name())
					err := internal.Image.Push(change.Image, change.RewrittenReference)
					if err != nil {
						log.Println("")
						return err
					}
					log.Println("Done")
					return nil
				},
				retry.Attempts(retries),
				retry.OnRetry(func(n uint, err error) {
					log.PrintErrf("Attempt #%d failed: %s\n", n+1, err.Error())
				}),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func modifyChart(originalChart *chart.Chart, actions []*RewriteAction, targetFormat string) error {
	var err error
	modifiedChart := originalChart
	for _, action := range actions {
		modifiedChart, err = action.Apply(modifiedChart)
		if err != nil {
			return err
		}
	}

	return saveChart(modifiedChart, targetFormat)
}

func saveChart(chart *chart.Chart, targetFormat string) error {
	cwd, _ := os.Getwd()
	tempDir, err := ioutil.TempDir(cwd, "relok8s-*")
	if err != nil {
		return err
	}

	filename, err := chartutil.Save(chart, tempDir)
	if err != nil {
		return err
	}

	destinationFile := TargetOutput(cwd, targetFormat, chart.Name(), chart.Metadata.Version)
	err = os.Rename(filename, destinationFile)
	if err != nil {
		return err
	}

	return os.Remove(tempDir)
}

func TargetOutput(cwd, targetFormat, name, version string) string {
	return filepath.Join(cwd, fmt.Sprintf(targetFormat, name, version))
}

func LoadImagePatterns(chart *chart.Chart, imagePatternsFile string, log Printer) ([]*ImageTemplate, error) {
	fileContents, err := ReadImagePatterns(imagePatternsFile, chart)
	if err != nil {
		return nil, fmt.Errorf("failed to read image pattern file: %w", err)
	}

	if fileContents == nil {
		return nil, fmt.Errorf("image patterns file is required. Please try again with '--image-patterns <image patterns file>'")
	}

	if imagePatternsFile == "" {
		log.Println("Using embedded image patterns file.")
	}

	var templateStrings []string
	err = yaml.Unmarshal(fileContents, &templateStrings)
	if err != nil {
		return nil, fmt.Errorf("image pattern file is not in the correct format: %w", err)
	}

	imagePatterns := []*ImageTemplate{}
	for _, line := range templateStrings {
		temp, err := NewFromString(line)
		if err != nil {
			return nil, err
		}
		imagePatterns = append(imagePatterns, temp)
	}

	return imagePatterns, nil
}

func ReadImagePatterns(patternsFile string, chart *chart.Chart) ([]byte, error) {
	if patternsFile != "" {
		return ioutil.ReadFile(patternsFile)
	}
	for _, file := range chart.Files {
		if file.Name == ".relok8s-images.yaml" && file.Data != nil {
			return file.Data, nil
		}
	}
	return nil, nil
}

func ParseRules(registryRule, repositoryPrefixRule, rulesFile string) (*RewriteRules, error) {
	rules := &RewriteRules{}
	if rulesFile != "" {
		fileContents, err := ioutil.ReadFile(rulesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read the rewrite rules file: %w", err)
		}

		err = yaml.UnmarshalStrict(fileContents, &rules)
		if err != nil {
			return nil, fmt.Errorf("the rewrite rules file is not in the correct format: %w", err)
		}
	}

	if registryRule != "" {
		rules.Registry = registryRule
	}
	if repositoryPrefixRule != "" {
		rules.RepositoryPrefix = repositoryPrefixRule
	}

	if (*rules == RewriteRules{}) {
		return nil, fmt.Errorf("Error: at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
	}

	return rules, nil
}
