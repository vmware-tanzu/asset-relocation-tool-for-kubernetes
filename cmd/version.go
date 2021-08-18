package cmd

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(VersionCmd)
	VersionCmd.SetOut(os.Stdout)
}

var Version = "dev"

func versionHelp() string {
	return fmt.Sprintf("Print the version number of %s", AppName)
}

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: versionHelp(),
	Long:  versionHelp(),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("%s version: %s\n", AppName, Version)
	},
}
