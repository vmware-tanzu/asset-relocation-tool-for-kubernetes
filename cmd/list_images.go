package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

func init() {
	rootCmd.AddCommand(ListImagesCmd)
	ListImagesCmd.SetOut(os.Stdout)
}

var ListImagesCmd = &cobra.Command{
	Use:     "list-images <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates),
	Run: func(cmd *cobra.Command, args []string) {
		var images []string

		for _, imageTemplate := range ImageTemplates {
			image, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			images = append(images, image.Remote())
		}

		encoded, err := json.Marshal(images)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}

		cmd.Println(string(encoded))
	},
}
