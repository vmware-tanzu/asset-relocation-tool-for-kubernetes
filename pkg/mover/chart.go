package mover

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/avast/retry-go"
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
	chart        *chart.Chart
	imageChanges []*internal.ImageChange
	chartChanges []*internal.RewriteAction
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

// BuildRules creates the rules spec string from registry & repo prefix settings
func BuildRules(registryRule, repositoryPrefixRule string) string {
	sb := &strings.Builder{}
	if registryRule != "" {
		fmt.Fprintf(sb, "registry: \"%s\"", registryRule)
	}
	if repositoryPrefixRule != "" {
		fmt.Fprintf(sb, "repositoryPrefix: \"%s\"", repositoryPrefixRule)
	}
	return sb.String()
}

// LoadRules from file rule settings first, or a rulesFile as a fallback
func LoadRules(registryRule, repositoryPrefixRule, rulesFile string) (string, error) {
	rules := BuildRules(registryRule, repositoryPrefixRule)
	if rules == "" && rulesFile != "" {
		contents, err := os.ReadFile(rulesFile)
		if err != nil {
			return "", fmt.Errorf("failed to read the rewrite rules file: %w", err)
		}
		return string(contents), nil
	}
	return rules, nil
}

// NewChartMover creates a ChartMover to relocate a chart following the given
// imagePatters and rules.
// TODO: Can/should we make this not need a logger as a input?
func NewChartMover(chart *chart.Chart, patterns string, rules string, log Printer) (*ChartMover, error) {
	imagePatterns, err := ParseImagePatterns(patterns, log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image patterns: %w", err)
	}
	imageChanges, err := PullOriginalImages(chart, imagePatterns, log)
	if err != nil {
		return nil, fmt.Errorf("failed to pull original images: %w", err)
	}
	rewriteRules, err := ParseRules(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rules: %w", err)
	}
	imageChanges, chartChanges, err := ComputeChanges(chart, imageChanges, rewriteRules, log)
	if err != nil {
		return nil, fmt.Errorf("failed to compute chart rewrites: %w", err)
	}
	return &ChartMover{chart: chart, imageChanges: imageChanges, chartChanges: chartChanges}, nil
}

// Print dumps the chart mover changes to the given logger
func (rl *ChartMover) Print(log Printer) {
	log.Println("\nImages to be pushed:")
	noImagesToPush := true
	for _, change := range rl.imageChanges {
		if change.ShouldPush() {
			log.Printf("  %s (%s)\n", change.RewrittenReference.Name(), change.Digest)
			noImagesToPush = false
		}
	}
	if noImagesToPush {
		log.Printf("  no images require pushing")
	}

	var chartToModify *chart.Chart
	for _, change := range rl.chartChanges {
		destination := change.FindChartDestination(rl.chart)
		if chartToModify != destination {
			chartToModify = destination
			log.Printf("\nChanges written to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}
		log.Printf("  %s: %s\n", change.Path, change.Value)
	}
}

// Apply executes the chart move image and chart changes
func (rl *ChartMover) Apply(outputFmt string, retries uint, log Printer) error {
	err := PushRewrittenImages(rl.imageChanges, retries, log)
	if err != nil {
		return err
	}
	log.Print("Writing chart files... ")
	err = modifyChart(rl.chart, rl.chartChanges, outputFmt)
	if err != nil {
		log.Println("")
		return err
	}
	log.Println("Done")
	return nil
}

// PullOriginalImages takes the chart and image patters to pull all images
// and compute the image changes for a move
func PullOriginalImages(chart *chart.Chart, pattens []*internal.ImageTemplate, log Printer) ([]*internal.ImageChange, error) {
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

// ComputeChanges calculates the image and chart changes, it also checks which
// images need to be pushed
func ComputeChanges(chart *chart.Chart, imageChanges []*internal.ImageChange, rules *internal.RewriteRules, log Printer) ([]*internal.ImageChange, []*internal.RewriteAction, error) {
	var chartChanges []*internal.RewriteAction
	imageCache := map[string]bool{}

	for _, change := range imageChanges {
		rules.Digest = change.Digest
		newActions, err := change.Pattern.Apply(change.ImageReference, rules)
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
				log.Printf("Checking %s (%s)... ", rewrittenImage.Name(), change.Digest)
				needToPush, err := internal.Image.Check(change.Digest, rewrittenImage)
				if err != nil {
					log.Println("")
					return nil, nil, err
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
	return imageChanges, chartChanges, nil
}

// PushRewrittenImages processes all image changes pushing to the target locations
func PushRewrittenImages(imageChanges []*internal.ImageChange, retries uint, log Printer) error {
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

func modifyChart(originalChart *chart.Chart, actions []*internal.RewriteAction, targetFormat string) error {
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

func ParseImagePatterns(patterns string, log Printer) ([]*internal.ImageTemplate, error) {
	var templateStrings []string
	err := yaml.Unmarshal(([]byte)(patterns), &templateStrings)
	if err != nil {
		return nil, fmt.Errorf("image pattern file is not in the correct format: %w", err)
	}

	imagePatterns := []*internal.ImageTemplate{}
	for _, line := range templateStrings {
		temp, err := internal.NewFromString(line)
		if err != nil {
			return nil, err
		}
		imagePatterns = append(imagePatterns, temp)
	}

	return imagePatterns, nil
}

func ParseRules(rules string) (*internal.RewriteRules, error) {
	rewriteRules := &internal.RewriteRules{}
	err := yaml.UnmarshalStrict(([]byte)(rules), &rewriteRules)
	if err != nil {
		return nil, fmt.Errorf("the given rewrite rules are not in the correct format: %w", err)
	}
	return rewriteRules, nil
}
