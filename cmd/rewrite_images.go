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

}

var RewriteImagesCmd = &cobra.Command{
	Use:   "rewrite-images <chart>",
	Short: "Renders and lists the images, found using the image list file",
	//Long:  "",
	PreRunE: LoadRewriteRules,
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
