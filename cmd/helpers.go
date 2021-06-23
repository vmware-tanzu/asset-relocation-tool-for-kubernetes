package cmd

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

var (
	Chart         *chart.Chart
	ImagePatterns []*lib.ImageTemplate
)

type Printer interface {
	Print(i ...interface{})
	Println(i ...interface{})
	Printf(format string, i ...interface{})
	PrintErr(i ...interface{})
	PrintErrln(i ...interface{})
	PrintErrf(format string, i ...interface{})
}

func LoadChart(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] == "" {
		return errors.New("missing helm chart")
	}

	var err error
	Chart, err = loader.Load(args[0])
	if err != nil {
		return errors.Wrapf(err, "failed to load helm chart at \"%s\"", args[0])
	}
	return nil
}

func LoadImagePatterns(cmd *cobra.Command, args []string) error {
	if ImagePatternsFile == "" {
		return errors.New("image patterns file is required. Please try again with '--image-patterns <image patterns file>'")
	}

	fileContents, err := ioutil.ReadFile(ImagePatternsFile)
	if err != nil {
		return errors.Wrap(err, "failed to read image pattern file")
	}

	var templateStrings []string
	err = yaml.Unmarshal(fileContents, &templateStrings)
	if err != nil {
		return errors.Wrap(err, "image pattern file is not in the correct format")
	}

	for _, line := range templateStrings {
		temp, err := lib.NewFromString(line)
		if err != nil {
			return err
		}
		ImagePatterns = append(ImagePatterns, temp)
	}

	return nil
}

func ParseRules(cmd *cobra.Command, args []string) error {
	Rules = &lib.RewriteRules{}
	if RulesFile != "" {
		fileContents, err := ioutil.ReadFile(RulesFile)
		if err != nil {
			return errors.Wrap(err, "failed to read rewrite the rules file")
		}

		err = yaml.UnmarshalStrict(fileContents, &Rules)
		if err != nil {
			return errors.Wrap(err, "the rewrite rules file is not in the correct format")
		}
	}

	if RegistryRule != "" {
		Rules.Registry = RegistryRule
	}
	if RepositoryPrefixRule != "" {
		Rules.RepositoryPrefix = RepositoryPrefixRule
	}

	if *Rules == (lib.RewriteRules{}) {
		return errors.New("Error: at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix")
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

func ModifyChart(originalChart *chart.Chart, actions []*lib.RewriteAction, targetFormat string) error {
	var err error
	modifiedChart := originalChart
	for _, action := range actions {
		modifiedChart, err = action.Apply(modifiedChart)
		if err != nil {
			return err
		}
	}

	return SaveChart(modifiedChart, targetFormat)
}

func SaveChart(chart *chart.Chart, targetFormat string) error {
	cwd, _ := os.Getwd()
	tempDir, err := ioutil.TempDir(cwd, "relok8s-*")
	if err != nil {
		return err
	}

	filename, err := chartutil.Save(chart, tempDir)
	if err != nil {
		return err
	}

	destinationFile := filepath.Join(cwd, fmt.Sprintf(targetFormat, chart.Name(), chart.Metadata.Version))
	err = os.Rename(filename, destinationFile)
	if err != nil {
		return err
	}

	return os.Remove(tempDir)
}
