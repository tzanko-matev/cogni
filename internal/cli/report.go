package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cogni/internal/report"
	"cogni/internal/runner"
)

var buildReportHTML = report.BuildReportHTML

func runReport(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		inputDir := fs.String("input", "", "Directory containing runs")
		specPath := fs.String("spec", "", "Path to config file (default: search for .cogni/config.yml)")
		rangeSpec := fs.String("range", "", "Commit range start..end")
		outputPath := fs.String("output", "", "Report output path")
		if err := fs.Parse(args); err != nil {
			return ExitUsage
		}

		if *rangeSpec == "" {
			fmt.Fprintln(stderr, "Missing --range")
			return ExitUsage
		}

		outputDir, repoRoot, err := resolveInputDir(*inputDir, *specPath)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to resolve input: %v\n", err)
			return ExitError
		}
		parsedRange, err := parseRange(*rangeSpec)
		if err != nil {
			fmt.Fprintf(stderr, "Invalid range: %v\n", err)
			return ExitUsage
		}
		rangeResult, err := resolveRange(context.Background(), repoRoot, parsedRange)
		if err != nil {
			fmt.Fprintf(stderr, "Range resolution failed: %v\n", err)
			return ExitError
		}

		commits := append([]string{rangeResult.Start}, rangeResult.Commits...)
		runs := make([]runner.Results, 0, len(commits))
		for _, commit := range commits {
			run, _, err := resolveRun(outputDir, repoRoot, commit)
			if err != nil {
				fmt.Fprintf(stderr, "Warning: missing run for %s\n", commit)
				continue
			}
			runs = append(runs, run)
		}
		if len(runs) == 0 {
			fmt.Fprintln(stderr, "No runs found for range")
			return ExitError
		}

		html := buildReportHTML(runs)
		reportPath := *outputPath
		if reportPath == "" {
			reportPath = filepath.Join(outputDir, "report.html")
		}
		if err := os.WriteFile(reportPath, []byte(html), 0o644); err != nil {
			fmt.Fprintf(stderr, "Failed to write report: %v\n", err)
			return ExitError
		}
		fmt.Fprintf(stdout, "Report written to %s\n", reportPath)
		return ExitOK
	}
}
