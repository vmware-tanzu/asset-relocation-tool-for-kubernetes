package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"helm.sh/helm/v3/pkg/chart"
)

var PullImages bool

func init() {
	rootCmd.AddCommand(ListImagesCmd)
	ListImagesCmd.SetOut(os.Stdout)

	ListImagesCmd.Flags().BoolVar(&PullImages, "pull", false, "pull unedited images")
}

var ListImagesCmd = &cobra.Command{
	Use:     "list-images <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	Example: "list-images <chart> -i <image templates>",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		images, err := GetImages(Chart)
		if err != nil {
			return err
		}

		if PullImages {
			cli, err := client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				return errors.Wrap(err, "failed to initialize docker client")
			}

			img := &ImageManager{
				Output:       cmd.ErrOrStderr(),
				Context:      context.Background(),
				DockerClient: cli,
			}

			for _, image := range images {
				err = img.PullImage(image)
				if err != nil {
					return err
				}
			}
		}

		encoded, err := json.Marshal(images)
		if err != nil {
			return err
		}

		cmd.Println(string(encoded))
		return nil
	},
}

func GetImages(chart *chart.Chart) ([]string, error) {
	var images []string

	for _, imageTemplate := range ImageTemplates {
		image, err := imageTemplate.Render(chart, []*lib.RewriteAction{})
		if err != nil {
			return nil, err
		}

		images = append(images, image.Remote())
	}

	return images, nil
}
