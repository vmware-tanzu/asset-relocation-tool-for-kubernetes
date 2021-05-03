package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

func init() {
	rootCmd.AddCommand(RewriteImagesCmd)
	RewriteImagesCmd.SetOut(os.Stdout)

	RewriteImagesCmd.Flags().StringVarP(
		&RewriteRulesFile,
		"rules-file",
		"r",
		"",
		"File with rewrite rules")
	RewriteImagesCmd.MarkFlagRequired("rules-file")
}

var RewriteImagesCmd = &cobra.Command{
	Use:     "rewrite-images <chart>",
	Short:   "Lists the container images in a chart, modified by the rewrite rules",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference, and uses rewrite rules to modify the values.",
	Example: "rewrite-images <chart> -i <image templates> -r <rules file>",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates, LoadRewriteRules),
	RunE: func(cmd *cobra.Command, args []string) error {
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
