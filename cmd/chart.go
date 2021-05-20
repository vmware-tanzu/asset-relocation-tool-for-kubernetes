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
	ChartCmd.SetOut(os.Stdout)

	ChartCmd.AddCommand(ChartSave)
	ChartSave.SetOut(os.Stdout)

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
	//Short:             "Relocates a Helm chart",
	//Long:              "Relocates a Helm chart by applying rewrite rules to the list of images and modifying the chart to refer to the new image references",
}

type ImageChange struct {
	Image       v1.Image
	OriginalTag name.Reference
	NewTag      name.Reference
	Digest      string
}

func (change *ImageChange) ShouldPush() bool {
	return change.OriginalTag.Name() != change.NewTag.Name()
}

var ChartSave = &cobra.Command{ // TODO: Please remove eventually
	Use:     "save <chart>",
	Short:   "For testing",
	PreRunE: RunSerially(LoadChart),
	RunE: func(cmd *cobra.Command, args []string) error {
		actions := []*lib.RewriteAction{{
			Path:  ".image.tag",
			Value: "oldest",
		}}

		return ModifyChart(Chart, actions)
	},
}

var ChartMoveCmd = &cobra.Command{
	Use:     "move <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	Example: "images <chart> -i <image templates>",
	PreRunE: RunSerially(LoadChart, LoadImagePatterns, ParseRules),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		var (
			actions      []*lib.RewriteAction
			imageDigests = map[string]string{}
			imagesToPush []*ImageChange
		)

		for _, imageTemplate := range ImagePatterns {
			originalImage, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				return err
			}

			change := &ImageChange{
				OriginalTag: originalImage,
			}
			if imageDigests[originalImage.Name()] == "" {
				cmd.Printf("Pulling %s... ", originalImage.Name())
				image, digest, err := PullImage(originalImage)
				if err != nil {
					cmd.Println("")
					return err
				}
				cmd.Println("Done")
				imageDigests[originalImage.Name()] = digest
				change.Image = image
				change.Digest = digest
			}

			newActions, err := imageTemplate.Apply(originalImage, Rules)
			if err != nil {
				return err
			}

			actions = append(actions, newActions...)

			rewrittenImage, err := imageTemplate.Render(Chart, newActions)
			if err != nil {
				return err
			}

			change.NewTag = rewrittenImage

			if change.ShouldPush() {
				cmd.Printf("Checking %s... ", rewrittenImage.Name())
				needToPush, err := CheckImage(change.Digest, rewrittenImage)
				if err != nil {
					cmd.Println("")
					return err
				}

				if needToPush {
					imagesToPush = append(imagesToPush, change)
				} else {
					cmd.Printf("%s already exists with digest %s\n", rewrittenImage.Name(), change.Digest)
				}
				cmd.Println("Done")
			}
		}

		PrintChanges(cmd.OutOrStdout(), imagesToPush, actions)

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

		for _, imageToPush := range imagesToPush {
			cmd.Printf("Pushing %s... ", imageToPush.NewTag.Name())
			err := PushImage(imageToPush.Image, imageToPush.NewTag)
			if err != nil {
				cmd.Println("")
				return err
			}
			cmd.Println("Done")
		}

		return ModifyChart(Chart, actions)
	},
}

func PrintChanges(output io.Writer, imagesToPush []*ImageChange, actions []*lib.RewriteAction) {
	_, _ = fmt.Fprintln(output, "\nImages to be pushed:")
	for _, image := range imagesToPush {
		_, _ = fmt.Fprintf(output, "  %s (%s)\n", image.NewTag.Name(), image.Digest)
	}

	_, _ = fmt.Fprintf(output, "\n Changes written to %s/values.yaml:\n", Chart.ChartPath())
	for _, action := range actions {
		_, _ = fmt.Fprintf(output, "  %s: %s\n", action.Path, action.Value)
	}
}
