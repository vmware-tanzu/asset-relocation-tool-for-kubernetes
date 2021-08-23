// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
)

// Package level documentation
func Example() {
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
		fmt.Println(err)
		return
	}

	// Next we just need to call Move providing the destination path of the rewritten Helm Chart
	// i.e chartMover.Move("./helm-chart-relocated.tgz")
	// Additionally, some extra metadata about the provided Helm Chart can be retrieved.
	// Useful to generate custom destination filepaths
	chartMetadata, err := chartMover.ChartMetadata()
	if err != nil {
		fmt.Println(err)
		return
	}

	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := fmt.Sprintf("./%s-%s.relocated.tgz", chartMetadata.Name, chartMetadata.Version)
	// Perform the push, rewrite and repackage of the Helm Chart
	chartMover.Move(destinationPath)
}
