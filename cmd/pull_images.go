package cmd

import (
	"context"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

func init() {
	rootCmd.AddCommand(PullImagesCmd)
	PullImagesCmd.SetOut(os.Stdout)
}

var PullImagesCmd = &cobra.Command{
	Use:     "pull-images <chart>",
	Short:   "Pulls the original container images from a chart",
	Long:    "",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return errors.Wrap(err, "failed to initialize docker client")
		}

		for _, imageTemplate := range ImageTemplates {
			image, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				return err
			}

			cmd.Printf("Pulling %s... ", image.Remote())
			_, err = cli.ImagePull(ctx, image.Remote(), types.ImagePullOptions{})
			if err != nil {
				cmd.Println("")
				return errors.Wrapf(err, "failed to pull image %s", image.Remote())
			}
			cmd.Println("Done")
		}

		return nil
	},
}
