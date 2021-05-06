package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

var pushImages bool

func init() {
	rootCmd.AddCommand(RewriteImagesCmd)
	RewriteImagesCmd.SetOut(os.Stdout)

	RewriteImagesCmd.Flags().StringVarP(
		&RewriteRulesFile,
		"rules-file",
		"r",
		"",
		"File with rewrite rules")
	var _ = RewriteImagesCmd.MarkFlagRequired("rules-file")

	RewriteImagesCmd.Flags().BoolVar(&pushImages, "push", false, "Push rewritten images")

	RewriteImagesCmd.Flags().StringArrayVar(&RegistryAuthList, "registry-auth", []string{}, "Supply credentials for connecting to a registry. In the format <registry.url>=<username>:<password>. Can be called multiple times.")
}

var RewriteImagesCmd = &cobra.Command{
	Use:     "rewrite-images <chart>",
	Short:   "Lists the container images in a chart, modified by the rewrite rules",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference, and uses rewrite rules to modify the values.",
	Example: "rewrite-images <chart> -i <image templates> -r <rules file>",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates, LoadRewriteRules, ParseRegistryAuth),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		var images []string

		for _, imageTemplate := range ImageTemplates {
			originalImage, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				return err
			}

			actions, err := imageTemplate.Apply(originalImage, Rules)
			if err != nil {
				return err
			}

			rewrittenImage, err := imageTemplate.Render(Chart, actions)
			if err != nil {
				return err
			}

			if pushImages {
				cli, err := client.NewClientWithOpts(client.FromEnv)
				if err != nil {
					return errors.Wrap(err, "failed to initialize docker client")
				}

				img := &ImageManager{
					Output:       cmd.ErrOrStderr(),
					Context:      context.Background(),
					DockerClient: cli,
					Auth:         RegistryAuth,
				}

				err = img.PushImage(originalImage, rewrittenImage)
				if err != nil {
					return err
				}
			}

			images = append(images, rewrittenImage.Remote())
		}

		encoded, err := json.Marshal(images)
		if err != nil {
			return err
		}

		cmd.Println(string(encoded))
		return nil
	},
}
