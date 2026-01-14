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
	"cogni/internal/ui/live"
)

// runEvalAndWrite is a test seam for question evaluation execution.
var runEvalAndWrite = runner.RunAndWrite

var evalFlagsRequiringValue = map[string]bool{
	"spec":       true,
	"agent":      true,
	"output-dir": true,
	"log":        true,
	"ui":         true,
}

var evalFlagsWithoutValue = map[string]bool{
	"verbose":  true,
	"no-color": true,
}

// runEval builds the handler for the eval command.
func runEval(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		normalizedArgs, err := normalizeEvalArgs(args)
		if err != nil {
			fmt.Fprintf(stderr, "Invalid eval arguments: %v\n", err)
			fmt.Fprintln(stderr, "Usage: cogni eval <questions_file> --agent <id>")
			return ExitUsage
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		specPath := fs.String("spec", "", "Path to config file (default: search for .cogni/config.yml)")
		agentID := fs.String("agent", "", "Agent id for evaluation (defaults to config default_agent)")
		outputDir := fs.String("output-dir", "", "Override output directory")
		verbose := fs.Bool("verbose", false, "Verbose logging")
		logPath := fs.String("log", "", "Write verbose logs to a file")
		noColor := fs.Bool("no-color", false, "Disable ANSI colors in verbose logs")
		uiMode := fs.String("ui", "auto", "UI mode: auto, live, plain")
		if err := fs.Parse(normalizedArgs); err != nil {
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

		results, paths, err := runEvalAndWrite(context.Background(), evalConfig, runner.RunParams{
			RepoRoot:         repoRoot,
			OutputDir:        *outputDir,
			Verbose:          *verbose,
			VerboseWriter:    stdout,
			VerboseLogWriter: logFile,
			NoColor:          *noColor,
			Observer:         observer,
		})
		stopUI()
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

func normalizeEvalArgs(args []string) ([]string, error) {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				positionals = append(positionals, args[i+1:]...)
			}
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			name, hasInlineValue := splitEvalFlag(arg)
			if name == "" {
				continue
			}
			if evalFlagsRequiringValue[name] && !hasInlineValue {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag %s requires a value", arg)
				}
				flags = append(flags, args[i+1])
				i++
				continue
			}
			if !evalFlagsRequiringValue[name] && !evalFlagsWithoutValue[name] && !hasInlineValue {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...), nil
}

func splitEvalFlag(flagToken string) (string, bool) {
	trimmed := strings.TrimLeft(flagToken, "-")
	if trimmed == "" {
		return "", false
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) == 2 {
		return parts[0], true
	}
	return parts[0], false
}
