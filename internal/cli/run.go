package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"cogni/internal/config"
	"cogni/internal/runner"
)

var runAndWrite = runner.RunAndWrite

func runRun(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		specPath := fs.String("spec", ".cogni.yml", "Path to .cogni.yml")
		agentOverride := fs.String("agent", "", "Agent id override")
		outputDir := fs.String("output-dir", "", "Override output directory")
		if err := fs.Parse(args); err != nil {
			return ExitUsage
		}

		cfg, err := config.Load(*specPath)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to load config: %v\n", err)
			return ExitError
		}

		selectors, err := runner.ParseSelectors(fs.Args())
		if err != nil {
			fmt.Fprintf(stderr, "Invalid selectors: %v\n", err)
			return ExitUsage
		}

		absSpec, err := filepath.Abs(*specPath)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to resolve spec path: %v\n", err)
			return ExitError
		}
		repoRoot := filepath.Dir(absSpec)

		results, paths, err := runAndWrite(context.Background(), cfg, runner.RunParams{
			RepoRoot:      repoRoot,
			OutputDir:     *outputDir,
			AgentOverride: *agentOverride,
			Selectors:     selectors,
		})
		if err != nil {
			fmt.Fprintf(stderr, "Run failed: %v\n", err)
			return ExitError
		}

		fmt.Fprintf(stdout, "Run %s completed\n", results.RunID)
		fmt.Fprintf(stdout, "Results: %s\n", paths.ResultsPath())
		fmt.Fprintf(stdout, "Report: %s\n", paths.ReportPath())
		return ExitOK
	}
}
