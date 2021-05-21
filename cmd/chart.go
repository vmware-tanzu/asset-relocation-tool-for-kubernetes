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
		var (
			actions      []*lib.RewriteAction
			imageDigests = map[string]*ImageChange{}
			changes      []*ImageChange
		)

		for _, imagePattern := range ImagePatterns {
			originalImage, err := imagePattern.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				return err
			}

			change := &ImageChange{
				Pattern:        imagePattern,
				ImageReference: originalImage,
			}

			if imageDigests[originalImage.Name()] == nil {
				cmd.Printf("Pulling %s... ", originalImage.Name())
				image, digest, err := PullImage(originalImage)
				if err != nil {
					cmd.Println("")
					return err
				}
				cmd.Println("Done")
				change.Image = image
				change.Digest = digest
				imageDigests[originalImage.Name()] = change
			} else {
				change.Image = imageDigests[originalImage.Name()].Image
				change.Digest = imageDigests[originalImage.Name()].Digest
			}
			changes = append(changes, change)
		}

		for _, change := range changes {
			Rules.Digest = change.Digest
			newActions, err := change.Pattern.Apply(change.ImageReference, Rules)
			if err != nil {
				return err
			}

			actions = append(actions, newActions...)

			rewrittenImage, err := change.Pattern.Render(Chart, newActions)
			if err != nil {
				return err
			}

			change.RewrittenReference = rewrittenImage

			if change.ShouldPush() {
				cmd.Printf("Checking %s (%s)... ", rewrittenImage.Name(), change.Digest)
				needToPush, err := CheckImage(change.Digest, rewrittenImage)
				if err != nil {
					cmd.Println("")
					return err
				}

				if needToPush {
					cmd.Println("Push required")
				} else {
					cmd.Println("Already exists")
					change.AlreadyPushed = true
				}
			}
		}

		PrintChanges(cmd.OutOrStdout(), changes, actions)

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

		for _, change := range changes {
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

		return ModifyChart(Chart, actions)
	},
}

func PrintChanges(output io.Writer, changes []*ImageChange, actions []*lib.RewriteAction) {
	_, _ = fmt.Fprintln(output, "\nImages to be pushed:")
	for _, change := range changes {
		if change.ShouldPush() {
			_, _ = fmt.Fprintf(output, "  %s (%s)\n", change.RewrittenReference.Name(), change.Digest)
		}
	}

	_, _ = fmt.Fprintf(output, "\n Changes written to %s/values.yaml:\n", Chart.ChartPath())
	for _, action := range actions {
		_, _ = fmt.Fprintf(output, "  %s: %s\n", action.Path, action.Value)
	}
}
