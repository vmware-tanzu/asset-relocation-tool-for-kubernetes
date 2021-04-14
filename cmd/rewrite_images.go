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

	RewriteImagesCmd.Flags().StringVar(&RewriteRulesFile, "rules-file", "", "File with rewrite rules")
	_ = RewriteImagesCmd.MarkFlagRequired("rules-file")
}

var RewriteImagesCmd = &cobra.Command{
	Use:     "rewrite-images <chart>",
	Short:   "Lists the container images in a chart, modified by the rewrite rules",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference, and uses rewrite rules to modify the values.",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates, LoadRewriteRules),
	Run: func(cmd *cobra.Command, args []string) {
		var images []string

		for _, imageTemplate := range ImageTemplates {
			originalImage, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			actions, err := imageTemplate.Apply(originalImage, Rules)
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			rewrittenImage, err := imageTemplate.Render(Chart, actions)
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			images = append(images, rewrittenImage.Remote())
		}

		encoded, err := json.Marshal(images)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}

		cmd.Println(string(encoded))
	},
}
