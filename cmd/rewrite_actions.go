package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

func init() {
	rootCmd.AddCommand(RewriteActionsCmd)
	RewriteActionsCmd.SetOut(os.Stdout)
}

var RewriteActionsCmd = &cobra.Command{
	Use:   "rewrite-actions <chart>",
	Short: "Prints a list of actions to rewrite a chart to applying the rewrite rules",
	//Long:  "",
	PreRunE: LoadRewriteRules,
	Run: func(cmd *cobra.Command, args []string) {
		var actions []*lib.RewriteAction

		for _, imageTemplate := range ImageTemplates {
			image, err := imageTemplate.Render(Chart, []*lib.RewriteAction{})
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			newActions, err := imageTemplate.Apply(image, Rules)
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}

			actions = append(actions, newActions...)
		}

		encoded, err := json.Marshal(actions)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}

		cmd.Println(string(encoded))
	},
}
