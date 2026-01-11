package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "pub",
	Short:   "Public.com Trading CLI",
	Long:    `A CLI for trading stocks, ETFs, options, and crypto via Public.com's API.`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
