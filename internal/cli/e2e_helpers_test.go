//go:build live
// +build live

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/report"
	"cogni/internal/runner"
	"cogni/internal/spec"

	"gopkg.in/yaml.v3"
)

// defaultLLMModel is the fallback model for live LLM tests.
const defaultLLMModel = "gpt-4.1-mini"

// requireLiveLLM ensures a live LLM key is available for integration tests.
func requireLiveLLM(t *testing.T) string {
	t.Helper()
	key := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	if key == "" {
		fallback := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
		if fallback != "" {
			t.Setenv("LLM_API_KEY", fallback)
			key = fallback
		}
	}
	if key == "" {
		t.Skip("LLM_API_KEY is not set")
	}
	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if model == "" {
		model = defaultLLMModel
	}
	return model
}

// modelOverride returns a model override from the environment if set.
func modelOverride(base string) string {
	override := strings.TrimSpace(os.Getenv("LLM_MODEL_OVERRIDE"))
	if override == "" {
		return base
	}
	return override
}

// defaultAgent returns a baseline agent config for tests.
func defaultAgent(id, model string) spec.AgentConfig {
	return spec.AgentConfig{
		ID:          id,
		Type:        "builtin",
		Provider:    "openrouter",
		Model:       model,
		MaxSteps:    6,
		Temperature: 0.0,
	}
}

// runCLI invokes the CLI and returns stdout, stderr, and exit code.
func runCLI(t *testing.T, args []string) (string, string, int) {
	t.Helper()
	var out, err bytes.Buffer
	exitCode := Run(args, &out, &err)
	return out.String(), err.String(), exitCode
}

// writeConfig writes a config file to the repo and returns its path.
func writeConfig(t *testing.T, repoRoot string, cfg spec.Config) string {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	path := filepath.Join(repoRoot, ".cogni", "config.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// resolveResults loads a run by ref and returns its metadata.
func resolveResults(t *testing.T, repoRoot, outputDir, ref string) (runner.Results, string) {
	t.Helper()
	results, runDir, err := report.ResolveRun(outputDir, repoRoot, ref)
	if err != nil {
		t.Fatalf("resolve run: %v", err)
	}
	return results, runDir
}

// outputDir resolves an output path under the repo root when needed.
func outputDir(repoRoot string, dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(repoRoot, dir)
}
