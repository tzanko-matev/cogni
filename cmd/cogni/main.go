package main

import (
	"os"

	"cogni/internal/cli"
)

// main launches the Cogni CLI.
func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
