package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"helm.sh/helm/v3/pkg/chart"
)

func init() {
	rootCmd.AddCommand(RewriteActionsCmd)
	RewriteActionsCmd.SetOut(os.Stdout)

	RewriteActionsCmd.Flags().StringVar(&RewriteRulesFile, "rules-file", "", "File with rewrite rules")
	_ = RewriteActionsCmd.MarkFlagRequired("rules-file")
}

var RewriteActionsCmd = &cobra.Command{
	Use:   "rewrite-actions <chart>",
	Short: "Generate a list of rewrite actions, based on the rewrite rules, to modify container image references in a Helm chart",
	//Long:  "",
	PreRunE: RunSerially(LoadChart, LoadImageTemplates, LoadRewriteRules),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		actions, err := GetRewriteActions(Chart, Rules)
		if err != nil {
			return err
		}

		encoded, err := json.Marshal(actions)
		if err != nil {
			return err
		}

		cmd.Println(string(encoded))
		return nil
	},
}

func GetRewriteActions(chart *chart.Chart, rules *lib.RewriteRules) ([]*lib.RewriteAction, error) {
	var actions []*lib.RewriteAction

	for _, imageTemplate := range ImageTemplates {
		image, err := imageTemplate.Render(chart, []*lib.RewriteAction{})
		if err != nil {
			return nil, err
		}

		newActions, err := imageTemplate.Apply(image, rules)
		if err != nil {
			return nil, err
		}

		actions = append(actions, newActions...)
	}

	return actions, nil
}
