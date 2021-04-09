package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
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

		for _, image := range ImageTemplates {
			err := image.Render(Chart)
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			images = append(images, Rules.RewriteImage(image.OriginalImage).Remote())
		}

		encoded, err := json.Marshal(images)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}

		cmd.Println(string(encoded))
	},
}
