package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const AppName = "chart-mover"

var (
	ImageListFile    string
	RewriteRulesFile string
	//RewriteRuleArgs       []string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&ImageListFile, "images", "i", "", "File with image reference templates")
	_ = rootCmd.MarkPersistentFlagRequired("images")

	rootCmd.PersistentFlags().StringVar(&RewriteRulesFile, "rules-file", "", "File with rewrite rules")

	rootCmd.SetOut(os.Stdout)
}

var rootCmd = &cobra.Command{
	Use:               fmt.Sprintf("%s <chart>", AppName),
	Short:             "Rewrites ",
	Long:              fmt.Sprintf(`%s gets all possible images out of a helm chart`, AppName),
	PersistentPreRunE: RunSerially(LoadChart, LoadImageTemplates),
	Args:              cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, image := range ImageTemplates {
			err := image.Render(Chart)
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			//newActions, err := image.Apply(rules)
		}

		//PullImages(ImageTemplates)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
