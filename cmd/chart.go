package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/avast/retry-go"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
	"helm.sh/helm/v3/pkg/chart"
)

const (
	defaultRetries = 3
)

var (
	skipConfirmation bool
	Retries          uint

	ImagePatternsFile string

	RulesFile            string
	RegistryRule         string
	RepositoryPrefixRule string
	Rules                *lib.RewriteRules
	Output               string
)

var (
	// ErrorMissingOutPlaceHolder if out flag is missing the wildcard *  placeholder
	ErrorMissingOutPlaceHolder = fmt.Errorf("missing '*' placeholder in --out flag")

	// ErrorBadExtension when the out flag does not use a expected file extension
	ErrorBadExtension = fmt.Errorf("bad extension (expected .tgz)")
)

func init() {
	rootCmd.AddCommand(ChartCmd)
	ChartCmd.AddCommand(ChartMoveCmd)
	ChartMoveCmd.SetOut(os.Stdout)

	ChartMoveCmd.Flags().StringVarP(&ImagePatternsFile, "image-patterns", "i", "", "File with image patterns")
	_ = ChartMoveCmd.MarkFlagRequired("images")
	ChartMoveCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Proceed without prompting for confirmation")

	ChartMoveCmd.Flags().StringVar(&RulesFile, "rules", "", "File containing rewrite rules")
	_ = ChartMoveCmd.Flags().MarkHidden("rules")
	ChartMoveCmd.Flags().StringVar(&RegistryRule, "registry", "", "Image registry rewrite rule")
	ChartMoveCmd.Flags().StringVar(&RepositoryPrefixRule, "repo-prefix", "", "Image repository prefix rule")

	ChartMoveCmd.Flags().UintVar(&Retries, "retries", defaultRetries, "Number of times to retry push operations")
	ChartMoveCmd.Flags().StringVar(&Output, "out", "./*.relocated.tgz", "Output chart name produced")
}

var ChartCmd = &cobra.Command{
	Use: "chart",
}

type ImageChange struct {
	Pattern            *lib.ImageTemplate
	ImageReference     name.Reference
	RewrittenReference name.Reference
	Image              v1.Image
	Digest             string
	AlreadyPushed      bool
}

func (change *ImageChange) ShouldPush() bool {
	return !change.AlreadyPushed && change.ImageReference.Name() != change.RewrittenReference.Name()
}

var ChartMoveCmd = &cobra.Command{
	Use:     "move <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	Example: "images <chart> -i <image templates>",
	PreRunE: RunSerially(LoadChart, LoadImagePatterns, ParseRules),
	RunE:    MoveChart,
}

func MoveChart(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	targetFormat, err := ParseOutputFlag(Output)
	if err != nil {
		return fmt.Errorf("failed to move chart: %w", err)
	}

	imageChanges, err := PullOriginalImages(Chart, ImagePatterns, cmd)
	if err != nil {
		return err
	}

	imageChanges, chartChanges, err := CheckNewImages(Chart, imageChanges, Rules, cmd)
	if err != nil {
		return err
	}

	PrintChanges(imageChanges, chartChanges, cmd)

	if !skipConfirmation {
		cmd.Println("Would you like to proceed? (y/N)")
		proceed, err := GetConfirmation(cmd.InOrStdin())
		if err != nil {
			return errors.Wrap(err, "failed to prompt for confirmation")
		}

		if !proceed {
			cmd.Println("Aborting")
			return nil
		}
	}

	err = PushRewrittenImages(imageChanges, cmd)
	if err != nil {
		return err
	}

	cmd.Print("Writing chart files... ")
	err = ModifyChart(Chart, chartChanges, targetFormat)
	if err != nil {
		cmd.Println("")
		return err
	}
	cmd.Println("Done")
	return nil
}

func ParseOutputFlag(out string) (string, error) {
	if !strings.Contains(out, "*") {
		return "", fmt.Errorf("%w: %s", ErrorMissingOutPlaceHolder, out)
	}
	if !strings.HasSuffix(out, ".tgz") {
		return "", fmt.Errorf("%w: %s", ErrorBadExtension, out)
	}
	return strings.Replace(out, "*", "%s-%s", 1), nil
}

func PullOriginalImages(chart *chart.Chart, pattens []*lib.ImageTemplate, log Printer) ([]*ImageChange, error) {
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
			image, digest, err := lib.Image.Pull(originalImage)
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

func CheckNewImages(chart *chart.Chart, imageChanges []*ImageChange, rules *lib.RewriteRules, log Printer) ([]*ImageChange, []*lib.RewriteAction, error) {
	var chartChanges []*lib.RewriteAction
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
				needToPush, err := lib.Image.Check(change.Digest, rewrittenImage)
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

func PushRewrittenImages(imageChanges []*ImageChange, log Printer) error {
	for _, change := range imageChanges {
		if change.ShouldPush() {
			err := retry.Do(
				func() error {
					log.Printf("Pushing %s... ", change.RewrittenReference.Name())
					err := lib.Image.Push(change.Image, change.RewrittenReference)
					if err != nil {
						log.Println("")
						return err
					}
					log.Println("Done")
					return nil
				},
				retry.Attempts(Retries),
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

func PrintChanges(imageChanges []*ImageChange, chartChanges []*lib.RewriteAction, log Printer) {
	log.Println("\nImages to be pushed:")
	noImagesToPush := true
	for _, change := range imageChanges {
		if change.ShouldPush() {
			log.Printf("  %s (%s)\n", change.RewrittenReference.Name(), change.Digest)
			noImagesToPush = false
		}
	}
	if noImagesToPush {
		log.Printf("  no images require pushing")
	}

	var chartToModify *chart.Chart
	for _, change := range chartChanges {
		destination := change.FindChartDestination(Chart)
		if chartToModify != destination {
			chartToModify = destination
			log.Printf("\nChanges written to %s/values.yaml:\n", chartToModify.ChartFullPath())
		}
		log.Printf("  %s: %s\n", change.Path, change.Value)
	}
}
