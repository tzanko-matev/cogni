package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/config"
	"cogni/internal/runner"
	"cogni/internal/ui/live"
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
		verbose := fs.Bool("verbose", false, "Verbose logging")
		logPath := fs.String("log", "", "Write verbose logs to a file")
		noColor := fs.Bool("no-color", false, "Disable ANSI colors in verbose logs")
		uiMode := fs.String("ui", "auto", "UI mode: auto, live, plain")
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

		var logFile io.WriteCloser
		if strings.TrimSpace(*logPath) != "" {
			dir := filepath.Dir(*logPath)
			if dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					fmt.Fprintf(stderr, "Failed to create log directory: %v\n", err)
					return ExitError
				}
			}
			file, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				fmt.Fprintf(stderr, "Failed to open log file: %v\n", err)
				return ExitError
			}
			logFile = file
			defer func() { _ = logFile.Close() }()
		}

		decision, err := resolveUIMode(*uiMode, *verbose, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "Invalid ui mode: %v\n", err)
			return ExitUsage
		}
		if decision.warning != "" {
			fmt.Fprintln(stderr, decision.warning)
		}
		var uiController *live.Controller
		if decision.useLive {
			uiController = live.Start(stdout, live.Options{NoColor: *noColor})
		}
		stopUI := func() {
			if uiController != nil {
				uiController.Close()
				uiController.Wait()
			}
		}
		defer stopUI()

		var observer runner.RunObserver
		if uiController != nil {
			observer = uiController
		}

		results, paths, err := runAndWrite(context.Background(), cfg, runner.RunParams{
			RepoRoot:         repoRoot,
			OutputDir:        *outputDir,
			AgentOverride:    *agentOverride,
			Selectors:        selectors,
			Verbose:          *verbose,
			VerboseWriter:    stdout,
			VerboseLogWriter: logFile,
			NoColor:          *noColor,
			Observer:         observer,
		})
		stopUI()
		if err != nil {
			fmt.Fprintf(stderr, "Run failed: %v\n", err)
			return ExitError
		}

		fmt.Fprintf(stdout, "Run %s completed\n", results.RunID)
		for _, task := range results.Tasks {
			if task.QuestionEval != nil {
				summary := task.QuestionEval.Summary
				fmt.Fprintf(stdout, "Question task %s accuracy: %d/%d (%.1f%%)\n",
					task.TaskID,
					summary.QuestionsCorrect,
					summary.QuestionsTotal,
					summary.Accuracy*100,
				)
			}
		}
		fmt.Fprintf(stdout, "Results: %s\n", paths.ResultsPath())
		fmt.Fprintf(stdout, "Report: %s\n", paths.ReportPath())
		return ExitOK
	}
}
