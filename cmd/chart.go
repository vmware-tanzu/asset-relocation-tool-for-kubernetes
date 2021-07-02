package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg"
)

const (
	defaultRetries = 3
)

var (
	skipConfirmation bool
	Retries          uint

	ImagePatternsFile string

	RulesFile            string
	RegistryRule         string
	RepositoryPrefixRule string
	Rules                *pkg.RewriteRules
	Output               string
)

var (
	// ErrorMissingOutPlaceHolder if out flag is missing the wildcard * placeholder
	ErrorMissingOutPlaceHolder = fmt.Errorf("missing '*' placeholder in --out flag")

	// ErrorBadExtension when the out flag does not use a expected file extension
	ErrorBadExtension = fmt.Errorf("bad extension (expected .tgz)")
)

func init() {
	rootCmd.AddCommand(ChartCmd)
	ChartCmd.AddCommand(ChartMoveCmd)
	ChartMoveCmd.SetOut(os.Stdout)

	ChartMoveCmd.Flags().StringVarP(&ImagePatternsFile, "image-patterns", "i", "", "File with image patterns")
	_ = ChartMoveCmd.MarkFlagRequired("images")
	ChartMoveCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Proceed without prompting for confirmation")

	ChartMoveCmd.Flags().StringVar(&RulesFile, "rules", "", "File containing rewrite rules")
	_ = ChartMoveCmd.Flags().MarkHidden("rules")
	ChartMoveCmd.Flags().StringVar(&RegistryRule, "registry", "", "Image registry rewrite rule")
	ChartMoveCmd.Flags().StringVar(&RepositoryPrefixRule, "repo-prefix", "", "Image repository prefix rule")

	ChartMoveCmd.Flags().UintVar(&Retries, "retries", defaultRetries, "Number of times to retry push operations")
	ChartMoveCmd.Flags().StringVar(&Output, "out", "./*.relocated.tgz", "Output chart name produced")
}

var ChartCmd = &cobra.Command{
	Use: "chart",
}

var ChartMoveCmd = &cobra.Command{
	Use:     "move <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	Example: "images <chart> -i <image templates>",
	PreRunE: RunSerially(LoadChart, LoadImagePatterns, ParseRules),
	RunE:    MoveChart,
}

func MoveChart(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	outputFmt, err := ParseOutputFlag(Output)
	if err != nil {
		return fmt.Errorf("failed to move chart: %w", err)
	}

	relocation, err := pkg.Compute(Chart, ImagePatterns, Rules, cmd)
	if err != nil {
		return err
	}

	pkg.PrintChanges(relocation, cmd)

	if !skipConfirmation {
		cmd.Println("Would you like to proceed? (y/N)")
		proceed, err := GetConfirmation(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to prompt for confirmation: %w", err)
		}

		if !proceed {
			cmd.Println("Aborting")
			return nil
		}
	}

	return pkg.Apply(relocation, outputFmt, Retries, cmd)
}

func ParseOutputFlag(out string) (string, error) {
	if !strings.Contains(out, "*") {
		return "", fmt.Errorf("%w: %s", ErrorMissingOutPlaceHolder, out)
	}
	if !strings.HasSuffix(out, ".tgz") {
		return "", fmt.Errorf("%w: %s", ErrorBadExtension, out)
	}
	return strings.Replace(out, "*", "%s-%s", 1), nil
}
