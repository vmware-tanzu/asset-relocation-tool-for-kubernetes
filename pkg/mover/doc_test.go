// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
)

// Package level documentation
func Example() {
	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := "%s-%s.relocated.tgz"

	// Initialize the Mover action
	chartMover, err := NewChartMover(
		&ChartMoveRequest{
			Source: ChartSource{
				// The Helm Chart can be provided in either tarball or directory form
				Chart: "./helm_chart.tgz",
				// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
				ImageHintsFile: "./image-hints.yaml",
			},
			Target: ChartTarget{
				Chart: destinationPath,
			},
			// Where to push and how to rewrite the found images
			// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
			Rules: RewriteRules{
				Registry:         "myregistry.com",
				RepositoryPrefix: "/myteam",
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
