// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
)

// Load a chart and its dependencies from a local intermediate bundle tarball
// into a registry and or repository
func Example_load() {
	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := "%s-%s.relocated.tgz"

	// Initialize the Mover action
	chartMover, err := NewChartMover(
		&ChartMoveRequest{
			Source: Source{
				Chart: ChartSpec{
					// The source intermediate bundle path to place the charts and all its dependencies
					IntermediateBundle: &IntermediateBundle{Path: "helm_chart.intermediate-bundle.tar"},
				},
				// no path to hints file as it is already coming inside the intermediate bundle
				ImageHintsFile: "./image-hints.yaml",
			},
			Target: Target{
				Chart: ChartSpec{Local: &LocalChart{Path: destinationPath}},
				// Where to push and how to rewrite the found images
				// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
				Rules: RewriteRules{
					Registry:         "myregistry.com",
					RepositoryPrefix: "/myteam",
				},
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Perform the push, rewrite and repackage of the Helm Chart
	// All origin data is taken from within the source intermediate bundle
	err = chartMover.Move()
	if err != nil {
		fmt.Println(err)
		return
	}
}
