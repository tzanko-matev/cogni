package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"cogni/internal/reportserver"
)

// serveReport is a test seam for running the report server.
var serveReport = reportserver.Serve

// runServe builds the handler for the serve command.
func runServe(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}

		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		addr := fs.String("addr", "127.0.0.1:5000", "Address to listen on")
		assetsBaseURL := fs.String("assets-base-url", "", "Base URL for report assets")
		if err := fs.Parse(args); err != nil {
			return ExitUsage
		}

		dbPath := fs.Arg(0)
		if dbPath == "" {
			fmt.Fprintln(stderr, "Missing <db.duckdb>")
			return ExitUsage
		}
		if fs.NArg() > 1 {
			fmt.Fprintln(stderr, "Too many arguments")
			return ExitUsage
		}
		if *addr == "" {
			fmt.Fprintln(stderr, "Missing --addr")
			return ExitUsage
		}
		if _, err := os.Stat(dbPath); err != nil {
			fmt.Fprintf(stderr, "Database not found: %v\n", err)
			return ExitError
		}

		cfg := reportserver.Config{
			Addr:          *addr,
			DBPath:        dbPath,
			AssetsBaseURL: *assetsBaseURL,
		}
		fmt.Fprintf(stdout, "Serving report at http://%s\n", cfg.Addr)
		if err := serveReport(context.Background(), cfg); err != nil {
			fmt.Fprintf(stderr, "Server error: %v\n", err)
			return ExitError
		}
		return ExitOK
	}
}
