package cmd

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var (
	Chart          *chart.Chart
	ImageTemplates []*lib.ImageTemplate
)

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

func LoadImageTemplates(cmd *cobra.Command, args []string) error {
	if ImageListFile == "" {
		return errors.New("image-templates is required. Please try again with '-i <image templates>'")
	}

	fileContents, err := ioutil.ReadFile(ImageListFile)
	if err != nil {
		return errors.Wrap(err, "failed to read image-templates")
	}

	var templateStrings []string
	err = yaml.Unmarshal(fileContents, &templateStrings)
	if err != nil {
		return errors.Wrap(err, "image-templates are not in the correct format")
	}

	for _, line := range templateStrings {
		temp, err := lib.NewFromString(line)
		if err != nil {
			return err
		}
		ImageTemplates = append(ImageTemplates, temp)
	}

	return nil
}

var Rules *lib.RewriteRules

func LoadRewriteRules(cmd *cobra.Command, args []string) error {
	if RewriteRulesFile == "" {
		return errors.New("rules-file is required. Please try again with '-r <rules file>'")
	}

	fileContents, err := ioutil.ReadFile(RewriteRulesFile)
	if err != nil {
		return errors.Wrap(err, "failed to read rewrite rules-file")
	}

	err = yaml.UnmarshalStrict(fileContents, &Rules)
	if err != nil {
		return errors.Wrap(err, "the rewrite rules-file contents are not in the correct format")
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
