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
)

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&ImageListFile,
		"image-templates",
		"i",
		"",
		"File with image reference templates")
	_ = rootCmd.MarkPersistentFlagRequired("images")

	rootCmd.SetOut(os.Stdout)
}

var rootCmd = &cobra.Command{
	Use: fmt.Sprintf("%s <chart>", AppName),
	//Short:             "Rewrites ",
	//Long:              fmt.Sprintf(`%s gets all possible images out of a helm chart`, AppName),
	//PersistentPreRunE: RunSerially(LoadChart, LoadImageTemplates),
	//Run: func(cmd *cobra.Command, args []string) {
	//	for _, imageTemplate := range ImageTemplates {
	//		_, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
	//		if err != nil {
	//			cmd.PrintErrln(err.Error())
	//			return
	//		}
	//
	//		//newActions, err := imageTemplate.Apply(rules)
	//	}
	//
	//	//PullImages(ImageTemplates)
	//},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
