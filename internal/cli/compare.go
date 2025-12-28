package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"cogni/internal/config"
	"cogni/internal/report"
	"cogni/internal/vcs"
)

var resolveRun = report.ResolveRun
var parseRange = vcs.ParseRange
var resolveRange = vcs.ResolveRange

func runCompare(cmd *Command) func(args []string, stdout, stderr io.Writer) int {
	return func(args []string, stdout, stderr io.Writer) int {
		if wantsHelp(args) {
			printCommandUsage(cmd, stdout)
			return ExitOK
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.SetOutput(stderr)
		inputDir := fs.String("input", "", "Directory containing runs")
		specPath := fs.String("spec", ".cogni.yml", "Path to .cogni.yml")
		baseRef := fs.String("base", "", "Base commit/run/ref")
		headRef := fs.String("head", "", "Head commit/run/ref")
		rangeSpec := fs.String("range", "", "Commit range start..end")
		if err := fs.Parse(args); err != nil {
			return ExitUsage
		}

		outputDir, repoRoot, err := resolveInputDir(*inputDir, *specPath)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to resolve input: %v\n", err)
			return ExitError
		}

		if *rangeSpec != "" {
			rangeSpecValue, err := parseRange(*rangeSpec)
			if err != nil {
				fmt.Fprintf(stderr, "Invalid range: %v\n", err)
				return ExitUsage
			}
			rangeResult, err := resolveRange(context.Background(), repoRoot, rangeSpecValue)
			if err != nil {
				fmt.Fprintf(stderr, "Range resolution failed: %v\n", err)
				return ExitError
			}
			*baseRef = rangeResult.Start
			*headRef = rangeResult.End
		}

		if *baseRef == "" {
			fmt.Fprintln(stderr, "Missing --base or --range")
			return ExitUsage
		}
		if *headRef == "" {
			*headRef = "HEAD"
		}

		baseResults, _, err := resolveRun(outputDir, repoRoot, *baseRef)
		if err != nil {
			fmt.Fprintf(stderr, "Base run not found: %v\n", err)
			return ExitError
		}
		headResults, _, err := resolveRun(outputDir, repoRoot, *headRef)
		if err != nil {
			fmt.Fprintf(stderr, "Head run not found: %v\n", err)
			return ExitError
		}

		passDelta := headResults.Summary.PassRate - baseResults.Summary.PassRate
		tokenDelta := headResults.Summary.TokensTotal - baseResults.Summary.TokensTotal

		fmt.Fprintf(stdout, "Base %s pass rate %.2f%% tokens %d\n", baseResults.Repo.Commit, baseResults.Summary.PassRate*100, baseResults.Summary.TokensTotal)
		fmt.Fprintf(stdout, "Head %s pass rate %.2f%% tokens %d\n", headResults.Repo.Commit, headResults.Summary.PassRate*100, headResults.Summary.TokensTotal)
		fmt.Fprintf(stdout, "Delta pass rate %+0.2f%% tokens %+d\n", passDelta*100, tokenDelta)
		return ExitOK
	}
}

func resolveInputDir(inputDir, specPath string) (string, string, error) {
	if inputDir != "" {
		abs, err := filepath.Abs(inputDir)
		if err != nil {
			return "", "", err
		}
		return abs, "", nil
	}
	cfg, err := config.Load(specPath)
	if err != nil {
		return "", "", err
	}
	absSpec, err := filepath.Abs(specPath)
	if err != nil {
		return "", "", err
	}
	repoRoot := filepath.Dir(absSpec)
	outputDir := cfg.Repo.OutputDir
	if outputDir == "" {
		return "", "", fmt.Errorf("repo.output_dir is required")
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(repoRoot, outputDir)
	}
	return outputDir, repoRoot, nil
}
