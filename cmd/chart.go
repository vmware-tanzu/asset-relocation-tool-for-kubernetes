package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/client"
	dockerparser "github.com/novln/docker-parser"
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

	RegistryAuthList []string
	RegistryAuth     = map[string]string{}
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

	ChartMoveCmd.Flags().StringArrayVar(&RegistryAuthList, "registry-auth", []string{}, "Supply credentials for connecting to a registry. In the format <registry.url>=<username>:<password>. Can be called multiple times.")
}

var ChartCmd = &cobra.Command{
	Use: "chart",
	//Short:             "Relocates a Helm chart",
	//Long:              "Relocates a Helm chart by applying rewrite rules to the list of images and modifying the chart to refer to the new image references",
}

type ImageChange struct {
	Digest   string
	Template string
	Source   *dockerparser.Reference
	Dest     *dockerparser.Reference
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
	PreRunE: RunSerially(LoadChart, LoadImagePatterns, ParseRules, ParseRegistryAuth),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		var (
			actions      []*lib.RewriteAction
			imageDigests = map[string]string{}
			imagesToPush []*ImageChange
		)

		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return errors.Wrap(err, "failed to initialize docker client")
		}

		img := &ImageManager{
			Context:      context.Background(),
			DockerClient: cli,
			Auth:         RegistryAuth,
		}

		for _, imageTemplate := range ImagePatterns {
			originalImage, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				return err
			}

			if imageDigests[originalImage.Remote()] == "" {
				cmd.Printf("Pulling %s... ", originalImage.Remote())
				digest, err := img.PullImage(originalImage)
				if err != nil {
					cmd.Println("")
					return err
				}
				cmd.Println("Done")
				imageDigests[originalImage.Remote()] = digest
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

			if originalImage.Remote() != rewrittenImage.Remote() {
				digest := imageDigests[originalImage.Remote()]
				needToPush, err := img.CheckImage(digest, rewrittenImage)
				if err != nil {
					return err
				}

				if needToPush {
					imagesToPush = append(imagesToPush, &ImageChange{
						Digest:   digest,
						Template: imageTemplate.Raw,
						Source:   originalImage,
						Dest:     rewrittenImage,
					})
				} else {
					cmd.Printf("%s already exists with digest %s\n", rewrittenImage.Remote(), digest)
				}
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
			cmd.Printf("Pushing %s... ", imageToPush.Dest.Remote())
			err = img.PushImage(imageToPush.Source, imageToPush.Dest)
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
		_, _ = fmt.Fprintf(output, "  %s (%s)\n", image.Dest.Remote(), image.Digest)
	}

	_, _ = fmt.Fprintf(output, "\n Changes written to %s/values.yaml:\n", Chart.ChartPath())
	for _, action := range actions {
		_, _ = fmt.Fprintf(output, "  %s: %s\n", action.Path, action.Value)
	}
}
