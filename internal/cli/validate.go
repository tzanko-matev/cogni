package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"cogni/internal/config"
)

func runValidate(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}

		flags := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		flags.SetOutput(stderr)
		specPath := flags.String("spec", ".cogni.yml", "Path to .cogni.yml")
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

		if _, err := config.Load(*specPath); err != nil {
			fmt.Fprintf(stderr, "Validation failed:\n%s\n", err.Error())
			return ExitError
		}

		fmt.Fprintln(stdout, "Config OK")
		return ExitOK
	}
}
