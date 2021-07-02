package pkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/avast/retry-go"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/internal"
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

type ChartRelocation struct {
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

func Compute(input *chart.Chart, imagePatterns []*ImageTemplate, rules *RewriteRules, log Printer) (*ChartRelocation, error) {
	imageChanges, err := PullOriginalImages(input, imagePatterns, log)
	if err != nil {
		return nil, err
	}

	return CheckNewImages(input, imageChanges, rules, log)
}

func Apply(relocation *ChartRelocation, outputFmt string, retries uint, log Printer) error {
	err := PushRewrittenImages(relocation.ImageChanges, retries, log)
	if err != nil {
		return err
	}

	log.Print("Writing chart files... ")
	err = ModifyChart(relocation.Chart, relocation.ChartChanges, outputFmt)
	if err != nil {
		log.Println("")
		return err
	}
	log.Println("Done")
	return nil
}

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

func CheckNewImages(chart *chart.Chart, imageChanges []*ImageChange, rules *RewriteRules, log Printer) (*ChartRelocation, error) {
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
	return &ChartRelocation{Chart: chart, ImageChanges: imageChanges, ChartChanges: chartChanges}, nil
}

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

func PrintChanges(relocation *ChartRelocation, log Printer) {
	log.Println("\nImages to be pushed:")
	noImagesToPush := true
	for _, change := range relocation.ImageChanges {
		if change.ShouldPush() {
			log.Printf("  %s (%s)\n", change.RewrittenReference.Name(), change.Digest)
			noImagesToPush = false
		}
	}
	if noImagesToPush {
		log.Printf("  no images require pushing")
	}

	var chartToModify *chart.Chart
	for _, change := range relocation.ChartChanges {
		destination := change.FindChartDestination(relocation.Chart)
		if chartToModify != destination {
			chartToModify = destination
			log.Printf("\nChanges written to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}
		log.Printf("  %s: %s\n", change.Path, change.Value)
	}
}

func ModifyChart(originalChart *chart.Chart, actions []*RewriteAction, targetFormat string) error {
	var err error
	modifiedChart := originalChart
	for _, action := range actions {
		modifiedChart, err = action.Apply(modifiedChart)
		if err != nil {
			return err
		}
	}

	return SaveChart(modifiedChart, targetFormat)
}

func SaveChart(chart *chart.Chart, targetFormat string) error {
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
