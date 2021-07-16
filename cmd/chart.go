package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/pkg/mover"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/pkg/rewrite"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
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

	Output string
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

var ChartCmd = &cobra.Command{Use: "chart"}

var ChartMoveCmd = &cobra.Command{
	Use:     "move <chart>",
	Short:   "Lists the container images in a chart",
	Long:    "Finds, renders and lists the container images found in a Helm chart, using an image template file to detect the templates that build the image reference.",
	Example: "images <chart> -i <image templates>",
	RunE:    MoveChart,
}

func loadChartFromArgs(args []string) (*chart.Chart, error) {
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("missing helm chart")
	}

	sourceChart, err := loader.Load(args[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load helm chart at \"%s\": %w", args[0], err)
	}

	return sourceChart, nil
}

func loadImagePatterns(chart *chart.Chart) (string, error) {
	patterns, err := mover.LoadImagePatterns(ImagePatternsFile, chart)
	if err != nil {
		return "", fmt.Errorf("failed to read image pattern file: %w", err)
	}
	if patterns == "" {
		return patterns, fmt.Errorf("image patterns file is required. Please try again with '--image-patterns <image patterns file>'")
	}
	if ImagePatternsFile == "" {
		log.Println("Using embedded image patterns file.")
	}
	return patterns, nil
}

func loadRules() (*rewrite.Rules, error) {
	rules := &rewrite.Rules{}
	if RulesFile != "" {
		var err error
		rules, err = rewrite.ParseRules(RulesFile)
		if err != nil {
			return nil, err
		}
	}

	if RegistryRule != "" {
		rules.Registry = RegistryRule
	}

	if RepositoryPrefixRule != "" {
		rules.RepositoryPrefix = RepositoryPrefixRule
	}

	return rules, nil
}

func MoveChart(cmd *cobra.Command, args []string) error {
	sourceChart, err := loadChartFromArgs(args)
	if err != nil {
		return err
	}

	imagePatterns, err := loadImagePatterns(sourceChart)
	if err != nil {
		return err
	}

	rules, err := loadRules()
	if err != nil {
		return err
	}
	if rules.IsEmpty() {
		return fmt.Errorf("at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
	}

	cmd.SilenceUsage = true

	outputFmt, err := ParseOutputFlag(Output)
	if err != nil {
		return fmt.Errorf("failed to move chart: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working dir: %w", err)
	}
	destinationFile := TargetOutput(cwd, outputFmt, sourceChart.Name(), sourceChart.Metadata.Version)

	chartMover, err := mover.NewChartMover(sourceChart, imagePatterns, rules, cmd)
	if err != nil {
		return err
	}
	chartMover = chartMover.WithRetries(Retries)
	chartMover.Print()

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

	return chartMover.Move(destinationFile)
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

func GetConfirmation(input io.Reader) (bool, error) {
	reader := bufio.NewReader(input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "y" || response == "yes" {
		return true, nil

	}
	return false, nil
}

func TargetOutput(cwd, targetFormat, name, version string) string {
	return filepath.Join(cwd, fmt.Sprintf(targetFormat, name, version))
}
