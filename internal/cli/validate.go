package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"cogni/internal/config"
)

// runValidate builds the handler for the validate command.
func runValidate(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}

		flags := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		flags.SetOutput(stderr)
		specPath := flags.String("spec", "", "Path to config file (default: search for .cogni/config.yml)")
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

		resolvedSpec, err := resolveSpecPath(*specPath)
		if err != nil {
			fmt.Fprintf(stderr, "Validation failed:\n%v\n", err)
			return ExitError
		}

		if _, err := config.Load(resolvedSpec); err != nil {
			fmt.Fprintf(stderr, "Validation failed:\n%s\n", err.Error())
			return ExitError
		}

		fmt.Fprintln(stdout, "Config OK")
		return ExitOK
	}
}
