// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/pkg/mover"
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
)

var (
	// errMissingOutPlaceHolder if out flag is missing the wildcard * placeholder
	errMissingOutPlaceHolder = fmt.Errorf("missing '*' placeholder in --out flag")

	// errBadExtension when the out flag does not use a expected file extension
	errBadExtension = fmt.Errorf("bad extension (expected .tgz)")
)

func init() {
	chartCmd := &cobra.Command{Use: "chart"}
	chartCmd.AddCommand(newChartMoveCmd())

	rootCmd.AddCommand(chartCmd)
}

func newChartMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move <chart>",
		Short:   "Relocates a Helm Chart along with their associated container images",
		Long:    "It takes the provided Helm Chart, resolves and repushes all the dependent images, providing as output a modified Helm Chart (and subcharts) pointing to the new location of the images.",
		Example: "move my-chart --image-patterns my-image-hints.yaml --registry my-registry.company.com ",
		RunE:    MoveChart,
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

func MoveChart(cmd *cobra.Command, args []string) error {
	targetRewriteRules := &mover.OCIImageRewriteRules{
		Registry:         registryRule,
		RepositoryPrefix: repositoryPrefixRule,
	}

	chartMover, err := mover.NewChartMover(
		args[0],
		imagePatternsFile,
		targetRewriteRules,
		mover.WithRetries(retries), mover.WithLogger(cmd),
	)
	if err != nil {
		if err == mover.ErrImageHintsMissing {
			return fmt.Errorf("image patterns file is required. Please try again with '--image-patterns <image patterns file>' or as part of the Helm chart at [chart]/%s file", mover.EmbeddedHintsFilename)
		} else if err == mover.ErrOCIRewritesMissing {
			return fmt.Errorf("at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
		}
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working dir: %w", err)
	}

	outputFmt, err := parseOutputFlag(output)
	if err != nil {
		return fmt.Errorf("the --out flag is incorrect: %w", err)
	}

	// Show confirmation
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

	destinationFile := targetOutput(cwd, outputFmt, chartMover.ChartName, chartMover.ChartVersion)
	cmd.Printf("Relocating %s@%s\n", chartMover.ChartName, chartMover.ChartVersion)

	if err := chartMover.Move(destinationFile); err != nil {
		return fmt.Errorf("error relocating the chart: %w", err)
	}

	cmd.Println(destinationFile)
	return nil
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
