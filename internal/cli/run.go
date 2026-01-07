package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"cogni/internal/config"
	"cogni/internal/runner"
)

// runAndWrite is a test seam for runner execution.
var runAndWrite = runner.RunAndWrite

// runRun builds the handler for the run command.
func runRun(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		specPath := fs.String("spec", "", "Path to config file (default: search for .cogni/config.yml)")
		agentOverride := fs.String("agent", "", "Agent id override")
		outputDir := fs.String("output-dir", "", "Override output directory")
		repeat := fs.Int("repeat", 1, "Repeat count")
		verbose := fs.Bool("verbose", false, "Verbose logging")
		noColor := fs.Bool("no-color", false, "Disable ANSI colors in verbose logs")
		if err := fs.Parse(args); err != nil {
			return ExitUsage
		}

		resolvedSpec, err := resolveSpecPath(*specPath)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to locate config: %v\n", err)
			return ExitError
		}

		cfg, err := config.Load(resolvedSpec)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to load config: %v\n", err)
			return ExitError
		}

		selectors, err := runner.ParseSelectors(fs.Args())
		if err != nil {
			fmt.Fprintf(stderr, "Invalid selectors: %v\n", err)
			return ExitUsage
		}

		repoRoot := config.RepoRootFromConfigPath(resolvedSpec)

		results, paths, err := runAndWrite(context.Background(), cfg, runner.RunParams{
			RepoRoot:      repoRoot,
			OutputDir:     *outputDir,
			AgentOverride: *agentOverride,
			Selectors:     selectors,
			Repeat:        *repeat,
			Verbose:       *verbose,
			VerboseWriter: stdout,
			NoColor:       *noColor,
		})
		if err != nil {
			fmt.Fprintf(stderr, "Run failed: %v\n", err)
			return ExitError
		}

		fmt.Fprintf(stdout, "Run %s completed\n", results.RunID)
		for _, task := range results.Tasks {
			if task.Cucumber == nil {
				continue
			}
			summary := task.Cucumber.Summary
			fmt.Fprintf(stdout, "Cucumber task %s accuracy: %d/%d (%.1f%%)\n",
				task.TaskID,
				summary.ExamplesCorrect,
				summary.ExamplesTotal,
				summary.Accuracy*100,
			)
		}
		fmt.Fprintf(stdout, "Results: %s\n", paths.ResultsPath())
		fmt.Fprintf(stdout, "Report: %s\n", paths.ReportPath())
		return ExitOK
	}
}
