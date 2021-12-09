// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
)

// Package level documentation
func Example_save() {
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
				Chart: ChartSpec{
					// The target intermediate bundle path to place the charts and all its dependencies
					IntermediateBundle: &IntermediateBundle{Path: "helm_chart.intermediate-bundle.tar"},
				},
				// No rewrite rules, as this only saves the chart and its dependencies as is
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Save the chart, hints file and all container images
	// into `helm_chart.intermediate-bundle.tar`
	//
	// So we get something like:
	// $ tar tvf helm_chart.intermediate-bundle.tar
	// -rw-r--r-- 0/0             201 1970-01-01 01:00 hints.yaml
	// -rw-r--r-- 0/0             349 1970-01-01 01:00 original-chart/...
	// -rw-r--r-- 0/0          773120 1970-01-01 01:00 images.tar
	err = chartMover.Move()
	if err != nil {
		fmt.Println(err)
		return
	}
}
