package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/config"
	"cogni/internal/vcs"
)

// runInit builds the handler for the init command.
func runInit(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}

		flags := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		flags.SetOutput(stderr)
		specPath := flags.String("spec", "", "Path to config file (default: auto-detect)")
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

		in := initInput
		if in == nil {
			in = os.Stdin
		}
		reader := bufio.NewReader(in)

		var targetSpecPath string
		var configDir string
		var repoRoot string

		specPathValue := strings.TrimSpace(*specPath)
		if specPathValue == "" {
			repoRoot = discoverGitRoot("")
			baseDir := repoRoot
			if baseDir == "" {
				wd, err := os.Getwd()
				if err != nil {
					fmt.Fprintf(stderr, "Init failed: %v\n", err)
					return ExitError
				}
				baseDir = wd
			}
			configDir = filepath.Join(baseDir, config.ConfigDirName)
			targetSpecPath = filepath.Join(configDir, config.ConfigFileName)
		} else {
			absSpec, err := filepath.Abs(specPathValue)
			if err != nil {
				fmt.Fprintf(stderr, "Init failed: %v\n", err)
				return ExitError
			}
			targetSpecPath = absSpec
			configDir = filepath.Dir(targetSpecPath)
			repoRoot = discoverGitRoot(config.RepoRootFromConfigPath(targetSpecPath))
		}

		if info, err := os.Stat(configDir); err == nil && !info.IsDir() {
			fmt.Fprintf(stderr, "Init failed: config directory %q is not a directory\n", configDir)
			return ExitError
		}
		if info, err := os.Stat(targetSpecPath); err == nil {
			if info.IsDir() {
				fmt.Fprintf(stderr, "Init failed: spec path %q is a directory\n", targetSpecPath)
				return ExitError
			}
			fmt.Fprintf(stderr, "Init failed: spec file already exists at %q\n", targetSpecPath)
			return ExitError
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "Init failed: stat spec file: %v\n", err)
			return ExitError
		}

		confirm, err := promptYesNo(reader, stdout, fmt.Sprintf("Initialize Cogni config in %s?", configDir), true)
		if err != nil {
			fmt.Fprintf(stderr, "Init failed: %v\n", err)
			return ExitError
		}
		if !confirm {
			fmt.Fprintln(stderr, "Init cancelled.")
			return ExitError
		}

		outputDir, err := promptString(reader, stdout, "Results folder", config.DefaultOutputDir)
		if err != nil {
			fmt.Fprintf(stderr, "Init failed: %v\n", err)
			return ExitError
		}

		addGitignore := false
		if repoRoot != "" {
			answer, err := promptYesNo(reader, stdout, "Add results folder to .gitignore?", true)
			if err != nil {
				fmt.Fprintf(stderr, "Init failed: %v\n", err)
				return ExitError
			}
			addGitignore = answer
		}

		if err := config.Scaffold(targetSpecPath, outputDir); err != nil {
			fmt.Fprintf(stderr, "Init failed: %v\n", err)
			return ExitError
		}

		schemasPath := filepath.Join(filepath.Dir(targetSpecPath), "schemas", "auth_flow_summary.schema.json")
		fmt.Fprintf(stdout, "Wrote %s\n", targetSpecPath)
		fmt.Fprintf(stdout, "Wrote %s\n", schemasPath)
		if addGitignore {
			updated, err := addGitignoreEntry(repoRoot, outputDir)
			if err != nil {
				fmt.Fprintf(stderr, "Init failed: update .gitignore: %v\n", err)
				return ExitError
			}
			if updated {
				fmt.Fprintf(stdout, "Updated %s\n", filepath.Join(repoRoot, ".gitignore"))
			}
		}
		return ExitOK
	}
}

// initInput allows tests to override stdin for init prompts.
var initInput io.Reader = os.Stdin

// discoverGitRoot returns the git root or empty when not found.
func discoverGitRoot(startDir string) string {
	root, err := vcs.DiscoverRepoRoot(context.Background(), startDir)
	if err != nil {
		return ""
	}
	return root
}
