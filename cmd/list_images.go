package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"helm.sh/helm/v3/pkg/chart"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		images, err := GetImages(Chart)
		if err != nil {
			return err
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
