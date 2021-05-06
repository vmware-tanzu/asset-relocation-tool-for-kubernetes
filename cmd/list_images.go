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

var pullImages bool

func init() {
	rootCmd.AddCommand(ListImagesCmd)
	ListImagesCmd.SetOut(os.Stdout)

	ListImagesCmd.Flags().BoolVar(&pullImages, "pull", false, "pull unedited images")

	ListImagesCmd.Flags().StringArrayVar(&RegistryAuthList, "registry-auth", []string{}, "Supply credentials for connecting to a registry. In the format <registry.url>=<username>:<password>. Can be called multiple times.")
}

var ListImagesCmd = &cobra.Command{
	Use:     "list-images <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	Example: "list-images <chart> -i <image templates>",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates, ParseRegistryAuth),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		var images []string

		for _, imageTemplate := range ImageTemplates {
			image, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				return err
			}

			if pullImages {
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

				err = img.PullImage(image)
				if err != nil {
					return err
				}
			}

			images = append(images, image.Remote())
		}

		encoded, err := json.Marshal(images)
		if err != nil {
			return err
		}

		cmd.Println(string(encoded))
		return nil
	},
}
