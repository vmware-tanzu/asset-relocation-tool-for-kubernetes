package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

const AppName = "relok8s"

var rootCmd = &cobra.Command{
	Use: AppName,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
