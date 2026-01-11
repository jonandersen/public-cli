package main

import (
	"fmt"
	"os"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "--help", "-h", "help":
		printHelp()
	case "--version", "-v", "version":
		fmt.Printf("pub version %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Run 'pub --help' for usage.")
		os.Exit(1)
	}
}

func printHelp() {
	help := `pub - Public.com Trading CLI

Usage:
  pub <command> [flags]

Commands:
  account     View account information and portfolio
  quote       Get quotes for stocks, ETFs, or crypto
  order       Place and manage orders
  options     View options chains and expirations

Flags:
  -h, --help      Show help
  -v, --version   Show version

Examples:
  pub account           View your accounts
  pub quote AAPL        Get quote for Apple
  pub order buy AAPL 10 Buy 10 shares of Apple

Use "pub <command> --help" for more information about a command.`

	fmt.Println(help)
}
