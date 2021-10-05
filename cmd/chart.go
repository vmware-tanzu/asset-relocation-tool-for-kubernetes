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
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
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

func validateChartArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("requires a chart argument")
	} else if len(args) > 1 {
		return fmt.Errorf("expected 1 chart argument, received %d args", len(args))
	}
	return nil
}

func newChartMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move <chart>",
		Short:   "Relocates a Helm Chart along with its associated container images",
		Long:    "It takes the provided Helm Chart, resolves and re-pushes all the dependent images, providing as output a modified Helm Chart (and subcharts) pointing to the new location of the images.",
		Example: "move my-chart --image-patterns my-image-hints.yaml --registry my-registry.company.com ",
		RunE:    moveChart,
		Args:    validateChartArgs,
	}

	f := cmd.Flags()
	// TODO(miguel): Change to image-hints
	f.StringVarP(&imagePatternsFile, "image-patterns", "i", "", "file with image patterns")
	f.BoolVarP(&skipConfirmation, "yes", "y", false, "Proceed without prompting for confirmation")

	f.StringVar(&registryRule, "registry", "", "hostname of the registry used to push the new images")
	f.StringVar(&repositoryPrefixRule, "repo-prefix", "", "path prefix to be used when relocating the container images")

	f.UintVar(&retries, "retries", defaultRetries, "number of times to retry push operations")
	f.StringVar(&output, "out", "*.relocated.tgz", "output chart name produced, from current dir")

	return cmd
}

func moveChart(cmd *cobra.Command, args []string) error {
	targetRewriteRules := &mover.RewriteRules{
		Registry:         registryRule,
		RepositoryPrefix: repositoryPrefixRule,
	}
	err := targetRewriteRules.Validate()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Could not get current path: %w", err)
	}

	outFmt, err := parseOutputFlag(output)
	if err != nil {
		return fmt.Errorf("failed to move chart: %w", err)
	}
	outputPathFmt := filepath.Join(cwd, outFmt)

	chartMover, err := mover.NewChartMover(
		&mover.ChartMoveRequest{
			Source: mover.Source{
				LocalChart:     mover.LocalChart{Path: args[0]},
				ImageHintsFile: imagePatternsFile,
			},
			Target: mover.Target{
				LocalChart: mover.LocalChart{Path: outputPathFmt},
				Rules:      *targetRewriteRules,
			},
		},
		mover.WithRetries(retries), mover.WithLogger(cmd),
	)
	if err != nil {
		var loadingError *mover.ChartLoadingError
		if errors.As(err, &loadingError) {
			return loadingError
		} else if err == mover.ErrImageHintsMissing {
			return fmt.Errorf("image patterns file is required. Please try again with '--image-patterns <image patterns file>' or as part of the Helm chart at [chart]/%s file", mover.EmbeddedHintsFilename)
		} else if err == mover.ErrOCIRewritesMissing {
			return fmt.Errorf("at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
		}

		cmd.SilenceUsage = true
		return err
	}

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

	return chartMover.Move()
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
