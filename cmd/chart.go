// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/pkg/mover"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const (
	defaultRetries = 3
)

var (
	skipConfirmation bool
	retries          uint

	imagePatternsFile string

	registryRule         string
	repositoryPrefixRule string

	output string

	// errMissingOutPlaceHolder if out flag is missing the wildcard * placeholder
	errMissingOutPlaceHolder = errors.New("missing '*' placeholder in --out flag")

	// errBadExtension when the out flag does not use a expected file extension
	errBadExtension = errors.New("bad extension (expected .tgz)")
)

func init() {
	chartCmd := &cobra.Command{Use: "chart"}
	chartCmd.AddCommand(newChartMoveCmd())
	// TODO(miguel): Revisit this override since it seems required only for testing
	chartCmd.SetOut(os.Stdout)

	rootCmd.AddCommand(chartCmd)
}

func newChartMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move <chart>",
		Short:   "Relocates a Helm Chart along with their associated container images",
		Long:    "It takes the provided Helm Chart, resolves and repushes all the dependent images, providing as output a modified Helm Chart (and subcharts) pointing to the new location of the images.",
		Example: "move my-chart --image-patterns my-image-hints.yaml --registry my-registry.company.com ",
		RunE:    moveChart,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires a chart argument")
			}
			return nil
		},
	}

	f := cmd.Flags()
	// TODO(miguel): Change to image-hints
	f.StringVarP(&imagePatternsFile, "image-patterns", "i", "", "dile with image patterns")
	f.BoolVarP(&skipConfirmation, "yes", "y", false, "Proceed without prompting for confirmation")

	f.StringVar(&registryRule, "registry", "", "hostname of the registry used to push the new images")
	f.StringVar(&repositoryPrefixRule, "repo-prefix", "", "path prefix to be used when relocating the container images")

	f.UintVar(&retries, "retries", defaultRetries, "number of times to retry push operations")
	f.StringVar(&output, "out", "./*.relocated.tgz", "output chart name produced")

	return cmd
}

func loadChartFromArgs(args []string) (*chart.Chart, error) {
	sourceChart, err := loader.Load(args[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load Helm Chart at \"%s\": %w", args[0], err)
	}

	return sourceChart, nil
}

func loadImagePatterns(chart *chart.Chart) (string, error) {
	patterns, err := mover.LoadImagePatterns(imagePatternsFile, chart)
	if err != nil {
		return "", fmt.Errorf("failed to read image pattern file: %w", err)
	}
	if patterns == "" {
		return patterns, errors.New("image patterns file is required. Please try again with '--image-patterns <image patterns file>'")
	}
	if imagePatternsFile == "" {
		log.Println("Using embedded image patterns file.")
	}
	return patterns, nil
}

func moveChart(cmd *cobra.Command, args []string) error {
	sourceChart, err := loadChartFromArgs(args)
	if err != nil {
		return err
	}

	imagePatterns, err := loadImagePatterns(sourceChart)
	if err != nil {
		return err
	}

	if registryRule == "" && repositoryPrefixRule == "" {
		return errors.New("at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
	}

	targetRewriteRules := &mover.RewriteRules{
		Registry:         registryRule,
		RepositoryPrefix: repositoryPrefixRule,
	}

	outputFmt, err := parseOutputFlag(output)
	if err != nil {
		return fmt.Errorf("failed to move chart: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working dir: %w", err)
	}
	destinationFile := targetOutput(cwd, outputFmt, sourceChart.Name(), sourceChart.Metadata.Version)

	cmd.Println("Computing relocation...")
	chartMover, err := mover.NewChartMover(sourceChart, imagePatterns, targetRewriteRules, cmd)
	if err != nil {
		return err
	}
	chartMover = chartMover.WithRetries(retries)
	chartMover.Print()

	if !skipConfirmation {
		cmd.Println("Would you like to proceed? (y/N)")
		proceed, err := getConfirmation(cmd.InOrStdin())
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

func parseOutputFlag(out string) (string, error) {
	if !strings.Contains(out, "*") {
		return "", fmt.Errorf("%w: %s", errMissingOutPlaceHolder, out)
	}
	if !strings.HasSuffix(out, ".tgz") {
		return "", fmt.Errorf("%w: %s", errBadExtension, out)
	}
	return strings.Replace(out, "*", "%s-%s", 1), nil
}

func getConfirmation(input io.Reader) (bool, error) {
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

func targetOutput(cwd, targetFormat, name, version string) string {
	return filepath.Join(cwd, fmt.Sprintf(targetFormat, name, version))
}
