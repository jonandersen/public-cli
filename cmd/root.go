package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

// jsonOutput controls whether output is formatted as JSON
var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:     "pub",
	Short:   "Public.com Trading CLI",
	Long:    `A CLI for trading stocks, ETFs, options, and crypto via Public.com's API.`,
	Version: Version,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
}

// GetJSONMode returns whether JSON output mode is enabled.
func GetJSONMode() bool {
	return jsonOutput
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
