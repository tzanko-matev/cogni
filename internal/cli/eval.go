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
	"cogni/internal/spec"
)

// runEvalAndWrite is a test seam for question evaluation execution.
var runEvalAndWrite = runner.RunAndWrite

// runEval builds the handler for the eval command.
func runEval(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		specPath := fs.String("spec", "", "Path to config file (default: search for .cogni/config.yml)")
		agentID := fs.String("agent", "", "Agent id for evaluation (defaults to config default_agent)")
		outputDir := fs.String("output-dir", "", "Override output directory")
		verbose := fs.Bool("verbose", false, "Verbose logging")
		logPath := fs.String("log", "", "Write verbose logs to a file")
		noColor := fs.Bool("no-color", false, "Disable ANSI colors in verbose logs")
		if err := fs.Parse(args); err != nil {
			return ExitUsage
		}

		questionArgs := fs.Args()
		if len(questionArgs) != 1 {
			fmt.Fprintln(stderr, "Usage: cogni eval <questions_file> --agent <id>")
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

		selectedAgent := strings.TrimSpace(*agentID)
		if selectedAgent == "" {
			selectedAgent = strings.TrimSpace(cfg.DefaultAgent)
		}
		if selectedAgent == "" {
			fmt.Fprintln(stderr, "Missing --agent (no default_agent configured)")
			return ExitUsage
		}

		questionsPath, err := filepath.Abs(questionArgs[0])
		if err != nil {
			fmt.Fprintf(stderr, "Failed to resolve questions file: %v\n", err)
			return ExitError
		}

		evalConfig := spec.Config{
			Version:      cfg.Version,
			Repo:         cfg.Repo,
			Agents:       cfg.Agents,
			DefaultAgent: cfg.DefaultAgent,
			Tasks: []spec.TaskConfig{{
				ID:            "question-eval",
				Type:          "question_eval",
				Agent:         selectedAgent,
				QuestionsFile: questionsPath,
			}},
		}
		config.Normalize(&evalConfig)
		repoRoot := config.RepoRootFromConfigPath(resolvedSpec)
		if err := config.Validate(&evalConfig, repoRoot); err != nil {
			fmt.Fprintf(stderr, "Invalid eval config: %v\n", err)
			return ExitError
		}

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

		results, paths, err := runEvalAndWrite(context.Background(), evalConfig, runner.RunParams{
			RepoRoot:         repoRoot,
			OutputDir:        *outputDir,
			Verbose:          *verbose,
			VerboseWriter:    stdout,
			VerboseLogWriter: logFile,
			NoColor:          *noColor,
		})
		if err != nil {
			fmt.Fprintf(stderr, "Eval failed: %v\n", err)
			return ExitError
		}

		fmt.Fprintf(stdout, "Run %s completed\n", results.RunID)
		for _, task := range results.Tasks {
			if task.QuestionEval == nil {
				continue
			}
			summary := task.QuestionEval.Summary
			fmt.Fprintf(stdout, "Question task %s accuracy: %d/%d (%.1f%%)\n",
				task.TaskID,
				summary.QuestionsCorrect,
				summary.QuestionsTotal,
				summary.Accuracy*100,
			)
		}
		fmt.Fprintf(stdout, "Results: %s\n", paths.ResultsPath())
		fmt.Fprintf(stdout, "Report: %s\n", paths.ReportPath())
		return ExitOK
	}
}
