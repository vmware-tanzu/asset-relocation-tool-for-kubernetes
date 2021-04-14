package cmd

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
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
	Chart, err = chartutil.Load(args[0])
	if err != nil {
		return errors.Wrap(err, "failed to load helm chart")
	}
	return nil
}

func LoadImageTemplates(cmd *cobra.Command, args []string) error {
	if ImageListFile == "" {
		return errors.New("image list file is required")
	}

	fileContents, err := ioutil.ReadFile(ImageListFile)
	if err != nil {
		return errors.Wrap(err, "failed to read image list file")
	}

	var templateStrings []string
	err = yaml.Unmarshal(fileContents, &templateStrings)
	if err != nil {
		return errors.Wrap(err, "the image list file contents are not in the correct format")
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
		return errors.New("rewrite rules file is required")
	}

	fileContents, err := ioutil.ReadFile(RewriteRulesFile)
	if err != nil {
		return errors.Wrap(err, "failed to read rewrite rules file")
	}

	err = yaml.Unmarshal(fileContents, &Rules)
	if err != nil {
		return errors.Wrap(err, "the rewrite rules file contents are not in the correct format")
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
