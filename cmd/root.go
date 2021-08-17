// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const AppName = "relok8s"

var rootCmd = &cobra.Command{
	Use: AppName,
	// Do not show the Usage page on every raised error
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// rootCmd may return an error, but Cobra is already displaying it
		// so here we just swallow but still exit with an error code
		os.Exit(1)
	}
}
