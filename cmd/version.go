package cmd

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

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: fmt.Sprintf("Print the version number of %s", AppName),
	Long:  fmt.Sprintf("Print the version number of %s", AppName),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("%s version: %s\n", AppName, Version)
	},
}
