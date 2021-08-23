package mover

import "fmt"

// Package level documentation
func Example() error {
	// Initialize the Mover action
	chartMover, err := NewChartMover(
		// The Helm Chart can be provided in either tarball or directory form
		"./helm_chart.tgz",
		// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
		"./image-hints.yaml",
		// Where to push and how to rewrite the found images
		// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
		&RewriteRules{
			Registry:         "myregistry.com",
			RepositoryPrefix: "/myteam",
		},
	)
	if err != nil {
		if err == ErrImageHintsMissing {
			return fmt.Errorf("image patterns file is required", EmbeddedHintsFilename)
		} else if err == ErrOCIRewritesMissing {
			return fmt.Errorf("at least one rewrite rule must be given")
		}

		return err
	}

	// Next we just need to call Move providing the destinatin path of the rewritten Helm Chart
	// i.e chartMover.Move("./helm-chart-relocated.tgz")
	// Additionally, some extra metadata about the provided Helm Chart can be retrieved.
	// Useful to generate custom destination filepaths
	chartMetadata, err := chartMover.ChartMetadata()
	if err != nil {
		return err
	}

	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := fmt.Sprintf("./%s-%s.relocated.tgz", chartMetadata.Name, chartMetadata.Version)
	// Perform the push, rewrite and repackage of the Helm Chart
	chartMover.Move(destinationPath)

	return nil
}
