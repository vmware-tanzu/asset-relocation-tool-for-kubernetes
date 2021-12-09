// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
)

// Move a chart and its dependencies to another registry and or repository
func Example() {
	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := "%s-%s.relocated.tgz"

	// Initialize the Mover action
	chartMover, err := NewChartMover(
		&ChartMoveRequest{
			Source: Source{
				// The Helm Chart can be provided in either tarball or directory form
				Chart: ChartSpec{Local: &LocalChart{Path: "./helm_chart.tgz"}},
				// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
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
	err = chartMover.Move()
	if err != nil {
		fmt.Println(err)
		return
	}
}
