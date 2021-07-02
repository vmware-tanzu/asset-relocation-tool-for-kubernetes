package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var (
	Chart         *chart.Chart
	ImagePatterns []*pkg.ImageTemplate
)

func LoadChart(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] == "" {
		return fmt.Errorf("missing helm chart")
	}

	var err error
	Chart, err = loader.Load(args[0])
	if err != nil {
		return fmt.Errorf("failed to load helm chart at \"%s\": %w", args[0], err)
	}
	return nil
}

func LoadImagePatterns(cmd *cobra.Command, args []string) error {
	fileContents, err := ReadImagePatterns(ImagePatternsFile, Chart)
	if err != nil {
		return fmt.Errorf("failed to read image pattern file: %w", err)
	}

	if fileContents == nil {
		return fmt.Errorf("image patterns file is required. Please try again with '--image-patterns <image patterns file>'")
	}

	if ImagePatternsFile == "" {
		cmd.Println("Using embedded image patterns file.")
	}

	var templateStrings []string
	err = yaml.Unmarshal(fileContents, &templateStrings)
	if err != nil {
		return fmt.Errorf("image pattern file is not in the correct format: %w", err)
	}

	for _, line := range templateStrings {
		temp, err := pkg.NewFromString(line)
		if err != nil {
			return err
		}
		ImagePatterns = append(ImagePatterns, temp)
	}

	return nil
}

func ReadImagePatterns(patternsFile string, chart *chart.Chart) ([]byte, error) {
	if patternsFile != "" {
		return ioutil.ReadFile(patternsFile)
	}
	for _, file := range chart.Files {
		if file.Name == ".relok8s-images.yaml" && file.Data != nil {
			return file.Data, nil
		}
	}
	return nil, nil
}

func ParseRules(cmd *cobra.Command, args []string) error {
	Rules = &pkg.RewriteRules{}
	if RulesFile != "" {
		fileContents, err := ioutil.ReadFile(RulesFile)
		if err != nil {
			return fmt.Errorf("failed to read the rewrite rules file: %w", err)
		}

		err = yaml.UnmarshalStrict(fileContents, &Rules)
		if err != nil {
			return fmt.Errorf("the rewrite rules file is not in the correct format: %w", err)
		}
	}

	if RegistryRule != "" {
		Rules.Registry = RegistryRule
	}
	if RepositoryPrefixRule != "" {
		Rules.RepositoryPrefix = RepositoryPrefixRule
	}

	if *Rules == (pkg.RewriteRules{}) {
		return fmt.Errorf("Error: at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
	}

	return nil
}

func RunSerially(funcs ...func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, fn := range funcs {
			err := fn(cmd, args)
			if err != nil {
				return err
			}
		}
		return nil
	}
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
