package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	skipConfirmation bool

	ImagePatternsFile string

	RulesFile            string
	RegistryRule         string
	RepositoryPrefixRule string
	Rules                *lib.RewriteRules
)

func init() {
	rootCmd.AddCommand(ChartCmd)
	ChartCmd.AddCommand(ChartMoveCmd)
	ChartMoveCmd.SetOut(os.Stdout)

	ChartMoveCmd.Flags().StringVarP(&ImagePatternsFile, "image-patterns", "i", "", "File with image patterns")
	_ = ChartMoveCmd.MarkFlagRequired("images")
	ChartMoveCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Do not prompt for confirmation")

	ChartMoveCmd.Flags().StringVar(&RulesFile, "rules", "", "File containing rewrite rules")
	ChartMoveCmd.Flags().StringVar(&RegistryRule, "registry", "", "Image registry rewrite rule")
	ChartMoveCmd.Flags().StringVar(&RepositoryPrefixRule, "repo-prefix", "", "Image repository prefix rule")
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
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		imageChanges, err := PullOriginalImages(cmd.OutOrStdout())
		if err != nil {
			return err
		}

		imageChanges, chartChanges, err := CheckNewImages(imageChanges, cmd.OutOrStdout())
		if err != nil {
			return err
		}

		PrintChanges(cmd.OutOrStdout(), imageChanges, chartChanges)

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

		for _, change := range imageChanges {
			if change.ShouldPush() {
				cmd.Printf("Pushing %s... ", change.RewrittenReference.Name())
				err := PushImage(change.Image, change.RewrittenReference)
				if err != nil {
					cmd.Println("")
					return err
				}
				cmd.Println("Done")
			}
		}

		return ModifyChart(Chart, chartChanges)
	},
}

func PullOriginalImages(output io.Writer) ([]*ImageChange, error) {
	var changes []*ImageChange
	imageCache := map[string]*ImageChange{}

	for _, imagePattern := range ImagePatterns {
		originalImage, err := imagePattern.Render(Chart, []*lib.RewriteAction{})
		if err != nil {
			return nil, err
		}

		change := &ImageChange{
			Pattern:        imagePattern,
			ImageReference: originalImage,
		}

		if imageCache[originalImage.Name()] == nil {
			_, _ = fmt.Fprintf(output, "Pulling %s... ", originalImage.Name())
			image, digest, err := PullImage(originalImage)
			if err != nil {
				_, _ = fmt.Fprintln(output, "")
				return nil, err
			}
			_, _ = fmt.Fprintln(output, "Done")
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

func CheckNewImages(imageChanges []*ImageChange, output io.Writer) ([]*ImageChange, []*lib.RewriteAction, error) {
	var chartChanges []*lib.RewriteAction

	for _, change := range imageChanges {
		Rules.Digest = change.Digest
		newActions, err := change.Pattern.Apply(change.ImageReference, Rules)
		if err != nil {
			return nil, nil, err
		}

		chartChanges = append(chartChanges, newActions...)

		rewrittenImage, err := change.Pattern.Render(Chart, newActions)
		if err != nil {
			return nil, nil, err
		}

		change.RewrittenReference = rewrittenImage

		if change.ShouldPush() {
			_, _ = fmt.Fprintf(output, "Checking %s (%s)... ", rewrittenImage.Name(), change.Digest)
			needToPush, err := CheckImage(change.Digest, rewrittenImage)
			if err != nil {
				_, _ = fmt.Fprintln(output, "")
				return nil, nil, err
			}

			if needToPush {
				_, _ = fmt.Fprintln(output, "Push required")
			} else {
				_, _ = fmt.Fprintln(output, "Already exists")
				change.AlreadyPushed = true
			}
		}
	}
	return imageChanges, chartChanges, nil
}

func PrintChanges(output io.Writer, imageChanges []*ImageChange, chartChanges []*lib.RewriteAction) {
	_, _ = fmt.Fprintln(output, "\nImages to be pushed:")
	for _, change := range imageChanges {
		if change.ShouldPush() {
			_, _ = fmt.Fprintf(output, "  %s (%s)\n", change.RewrittenReference.Name(), change.Digest)
		}
	}

	var chartToModify *chart.Chart
	for _, change := range chartChanges {
		destination := change.FindChartDestination(Chart)
		if chartToModify != destination {
			chartToModify = destination
			_, _ = fmt.Fprintf(output, "\n Changes written to %s/values.yaml:\n", chartToModify.ChartPath())
		}
		_, _ = fmt.Fprintf(output, "  %s: %s\n", change.Path, change.Value)
	}
}
