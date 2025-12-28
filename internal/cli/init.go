package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"cogni/internal/config"
)

func runInit(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}

		flags := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		flags.SetOutput(stderr)
		specPath := flags.String("spec", ".cogni.yml", "Path to write .cogni.yml")
		if err := flags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCommandUsage(cmd, stdout)
				return ExitOK
			}
			fmt.Fprintf(stderr, "invalid arguments: %v\n", err)
			printCommandUsage(cmd, stderr)
			return ExitUsage
		}
		if flags.NArg() > 0 {
			fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(flags.Args(), " "))
			printCommandUsage(cmd, stderr)
			return ExitUsage
		}

		if err := config.Scaffold(*specPath); err != nil {
			fmt.Fprintf(stderr, "Init failed: %v\n", err)
			return ExitError
		}

		schemasPath := filepath.Join(filepath.Dir(*specPath), "schemas", "auth_flow_summary.schema.json")
		fmt.Fprintf(stdout, "Wrote %s\n", *specPath)
		fmt.Fprintf(stdout, "Wrote %s\n", schemasPath)
		return ExitOK
	}
}
