package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/report"
	"cogni/internal/runner"
	"cogni/internal/spec"

	"gopkg.in/yaml.v3"
)

const defaultLLMModel = "gpt-4.1-mini"

const jsonRules = `Rules:
- Use repository tools to read the cited files.
- Return ONLY a JSON object with keys "answer" and "citations".
- "citations" must be an array of objects: {"path":"...","lines":[start,end]} with 1-based inclusive lines.`

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

func modelOverride(base string) string {
	override := strings.TrimSpace(os.Getenv("LLM_MODEL_OVERRIDE"))
	if override == "" {
		return base
	}
	return override
}

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

func runCLI(t *testing.T, args []string) (string, string, int) {
	t.Helper()
	var out, err bytes.Buffer
	exitCode := Run(args, &out, &err)
	return out.String(), err.String(), exitCode
}

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

func resolveResults(t *testing.T, repoRoot, outputDir, ref string) (runner.Results, string) {
	t.Helper()
	results, runDir, err := report.ResolveRun(outputDir, repoRoot, ref)
	if err != nil {
		t.Fatalf("resolve run: %v", err)
	}
	return results, runDir
}

func outputDir(repoRoot string, dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(repoRoot, dir)
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func writeFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func simpleRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "-c", "init.defaultBranch=main", "init")

	writeFile(t, root, "README.md", "# Sample Service\nThis repo exists only for Cogni integration tests.\n")
	writeFile(t, root, "app.md", "# App Notes\nService owner: Platform Team\n")
	writeFile(t, root, "config.yml", "service_name: Sample Service\nowner: Platform Team\n")
	writeFile(t, root, filepath.Join("config", "app-config.yml"), "mode: sample\n")

	runGit(t, root, "add", "README.md", "app.md", "config.yml", "config/app-config.yml")
	runGit(t, root, "commit", "-m", "init")
	return root
}

func historyRepo(t *testing.T) (string, string, string) {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "-c", "init.defaultBranch=main", "init")

	writeFile(t, root, "README.md", "# Sample Service\nRelease stage: alpha\n")
	writeFile(t, root, "change-log.md", "- 0.1.0: initial\n")
	runGit(t, root, "add", "README.md", "change-log.md")
	runGit(t, root, "commit", "-m", "init")
	first := runGit(t, root, "rev-parse", "HEAD")

	writeFile(t, root, "README.md", "# Sample Service\nRelease stage: beta\n")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "update release stage")
	second := runGit(t, root, "rev-parse", "HEAD")

	return root, first, second
}

func cucumberRepo(t *testing.T) (string, string, string) {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "-c", "init.defaultBranch=main", "init")

	writeFile(t, root, "README.md", "Cucumber repo\n")
	writeFile(t, root, "spec/features/sample.feature", `Feature: Sample

  @id:smoke
  Scenario: Smoke
    Given something
`)
	writeFile(t, root, "spec/expectations/expectations.yml", `examples:
  smoke:1: true
`)
	runGit(t, root, "add", "README.md", "spec/features/sample.feature", "spec/expectations/expectations.yml")
	runGit(t, root, "commit", "-m", "init cucumber fixtures")
	first := runGit(t, root, "rev-parse", "HEAD")

	writeFile(t, root, "README.md", "Cucumber repo updated\n")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "update readme")
	second := runGit(t, root, "rev-parse", "HEAD")

	return root, first, second
}

func baseConfig(output string, agents []spec.AgentConfig, defaultAgent string, tasks []spec.TaskConfig) spec.Config {
	return spec.Config{
		Version:      1,
		Repo:         spec.RepoConfig{OutputDir: output},
		Agents:       agents,
		DefaultAgent: defaultAgent,
		Tasks:        tasks,
	}
}

func TestE2EProviderConnectivity(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t1",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"Sample Service",
				"README.md",
			},
		},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}

	results, runDir := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if len(results.Tasks) != 1 || results.Tasks[0].Status != "pass" {
		t.Fatalf("expected pass result, got %+v", results.Tasks)
	}
	if _, err := os.Stat(filepath.Join(runDir, "results.json")); err != nil {
		t.Fatalf("missing results.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "report.html")); err != nil {
		t.Fatalf("missing report.html: %v", err)
	}
}

func TestE2EBasicQACitations(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read app.md and report the service owner. The answer must include the exact phrase \"Platform Team\". Cite app.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t2",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"Platform Team",
				"app.md",
			},
		},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "pass" || !results.Tasks[0].Attempts[0].Eval.CitationValid {
		t.Fatalf("expected citation pass, got %+v", results.Tasks[0])
	}
}

func TestE2EMultiFileEvidence(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Using README.md and app.md, report the project name and service owner in one sentence. The answer must include \"Sample Service\" and \"Platform Team\". Include citations entries for README.md and app.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t3",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"Sample Service",
				"Platform Team",
				"README.md",
				"app.md",
			},
		},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "pass" {
		t.Fatalf("expected pass, got %+v", results.Tasks[0])
	}
}

func TestE2ERepositoryNavigation(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read config/app-config.yml and report its path. The answer must include the exact string \"config/app-config.yml\". Cite config/app-config.yml.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t4",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval: spec.TaskEval{
			ValidateCitations: true,
			MustContainStrings: []string{
				"config/app-config.yml",
			},
		},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "pass" {
		t.Fatalf("expected pass, got %+v", results.Tasks[0])
	}
}

func TestE2EMultipleTasksSummary(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	promptA := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	promptB := "Read app.md and report the service owner. The answer must include the exact phrase \"Platform Team\". Cite app.md.\n\n" + jsonRules
	promptC := "Read config/app-config.yml and report the mode value. The answer must include the exact word \"sample\". Cite config/app-config.yml.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t5a",
		Type:   "qa",
		Agent:  "default",
		Prompt: promptA,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}, {
		ID:     "t5b",
		Type:   "qa",
		Agent:  "default",
		Prompt: promptB,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Platform Team", "app.md"}},
	}, {
		ID:     "t5c",
		Type:   "qa",
		Agent:  "default",
		Prompt: promptC,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"sample", "config/app-config.yml"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Summary.TasksTotal != 3 || results.Summary.TasksPassed != 3 || results.Summary.TasksFailed != 0 {
		t.Fatalf("unexpected summary: %+v", results.Summary)
	}
	if len(results.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(results.Tasks))
	}
}

func TestE2EMultipleAgentsModelOverride(t *testing.T) {
	model := requireLiveLLM(t)
	override := modelOverride(model)
	repoRoot := simpleRepo(t)
	agents := []spec.AgentConfig{
		defaultAgent("default", model),
		defaultAgent("secondary", model),
	}
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", agents, "default", []spec.TaskConfig{{
		ID:     "t6a",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}, {
		ID:     "t6b",
		Type:   "qa",
		Agent:  "secondary",
		Model:  override,
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if len(results.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(results.Tasks))
	}
	if results.Tasks[0].Attempts[0].AgentID != "default" {
		t.Fatalf("unexpected task 1 agent: %+v", results.Tasks[0].Attempts[0])
	}
	if results.Tasks[1].Attempts[0].AgentID != "secondary" || results.Tasks[1].Attempts[0].Model != override {
		t.Fatalf("unexpected task 2 agent/model: %+v", results.Tasks[1].Attempts[0])
	}
}

func TestE2EBudgetLimitFailure(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Before answering, call the list_files tool with an empty glob. Do not answer until after the tool result. After the tool response, return ONLY JSON with keys \"answer\" and \"citations\".\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t7",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Budget: spec.TaskBudget{MaxSteps: 1},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "fail" || results.Tasks[0].FailureReason == nil || *results.Tasks[0].FailureReason != "budget_exceeded" {
		t.Fatalf("expected budget exceeded failure, got %+v", results.Tasks[0])
	}
}

func TestE2EOutputArtifacts(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t8",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	_, runDir := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	resultsPath := filepath.Join(runDir, "results.json")
	reportPath := filepath.Join(runDir, "report.html")

	resultsPayload, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("read results: %v", err)
	}
	if !bytes.Contains(resultsPayload, []byte(`"run_id"`)) || !bytes.Contains(resultsPayload, []byte(`"tasks"`)) {
		t.Fatalf("results.json missing expected fields")
	}

	reportPayload, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportPayload), "Cogni Report") {
		t.Fatalf("report.html missing heading")
	}
}

func TestE2ECompareAcrossCommits(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot, baseCommit, headCommit := historyRepo(t)
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t9",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"Sample Service", "README.md"}},
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	runGit(t, repoRoot, "checkout", baseCommit)
	if _, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath}); exitCode != ExitOK {
		t.Fatalf("base run failed: %s", stderr)
	}

	runGit(t, repoRoot, "checkout", headCommit)
	if _, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath}); exitCode != ExitOK {
		t.Fatalf("head run failed: %s", stderr)
	}

	stdout, stderr, exitCode := runCLI(t, []string{"compare", "--spec", specPath, "--base", baseCommit, "--head", headCommit})
	if exitCode != ExitOK {
		t.Fatalf("compare failed: %s", stderr)
	}
	if !strings.Contains(stdout, "Delta") {
		t.Fatalf("expected compare output, got %q", stdout)
	}

	reportPath := filepath.Join(outputDir(repoRoot, cfg.Repo.OutputDir), "report.html")
	stdout, stderr, exitCode = runCLI(t, []string{"report", "--spec", specPath, "--range", baseCommit + ".." + headCommit, "--output", reportPath})
	if exitCode != ExitOK {
		t.Fatalf("report failed: %s", stderr)
	}
	if !strings.Contains(stdout, "Report written") {
		t.Fatalf("expected report output, got %q", stdout)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("missing report output: %v", err)
	}
}

func TestE2ECucumberEvalCompareAcrossCommits(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot, baseCommit, headCommit := cucumberRepo(t)
	cfg := spec.Config{
		Version:      1,
		Repo:         spec.RepoConfig{OutputDir: "./cogni-results"},
		Agents:       []spec.AgentConfig{defaultAgent("default", model)},
		DefaultAgent: "default",
		Adapters: []spec.AdapterConfig{{
			ID:              "manual",
			Type:            "cucumber_manual",
			FeatureRoots:    []string{"spec/features"},
			ExpectationsDir: "spec/expectations",
		}},
		Tasks: []spec.TaskConfig{{
			ID:             "cucumber-eval",
			Type:           "cucumber_eval",
			Agent:          "default",
			Adapter:        "manual",
			Features:       []string{"spec/features/sample.feature"},
			PromptTemplate: "Return ONLY JSON: {\"example_id\":\"{example_id}\",\"implemented\":true}",
		}},
	}
	specPath := writeConfig(t, repoRoot, cfg)

	runGit(t, repoRoot, "checkout", baseCommit)
	if _, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath}); exitCode != ExitOK {
		t.Fatalf("base run failed: %s", stderr)
	}

	runGit(t, repoRoot, "checkout", headCommit)
	if _, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath}); exitCode != ExitOK {
		t.Fatalf("head run failed: %s", stderr)
	}

	stdout, stderr, exitCode := runCLI(t, []string{"compare", "--spec", specPath, "--base", baseCommit, "--head", headCommit})
	if exitCode != ExitOK {
		t.Fatalf("compare failed: %s", stderr)
	}
	if !strings.Contains(stdout, "Delta") {
		t.Fatalf("expected compare output, got %q", stdout)
	}

	reportPath := filepath.Join(outputDir(repoRoot, cfg.Repo.OutputDir), "cucumber-report.html")
	stdout, stderr, exitCode = runCLI(t, []string{"report", "--spec", specPath, "--range", baseCommit + ".." + headCommit, "--output", reportPath})
	if exitCode != ExitOK {
		t.Fatalf("report failed: %s", stderr)
	}
	if !strings.Contains(stdout, "Report written") {
		t.Fatalf("expected report output, got %q", stdout)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("missing report output: %v", err)
	}
}

func TestE2EInitToRunFlow(t *testing.T) {
	model := requireLiveLLM(t)
	requireGit(t)
	repoRoot := t.TempDir()
	runGit(t, repoRoot, "-c", "init.defaultBranch=main", "init")
	writeFile(t, repoRoot, "README.md", "init\n")
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "init")

	specPath := filepath.Join(repoRoot, ".cogni", "config.yml")
	origInput := initInput
	initInput = strings.NewReader("y\n\nn\n")
	t.Cleanup(func() { initInput = origInput })
	_, stderr, exitCode := runCLI(t, []string{"init", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("init failed: %s", stderr)
	}

	prompt := "Read README.md and return the project name. The answer must include the exact word \"init\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t11",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
		Eval:   spec.TaskEval{ValidateCitations: true, MustContainStrings: []string{"init", "README.md"}},
	}})
	writeConfig(t, repoRoot, cfg)

	_, stderr, exitCode = runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("run failed: %s", stderr)
	}
}

func TestE2EProviderFailureHandling(t *testing.T) {
	model := requireLiveLLM(t)
	repoRoot := simpleRepo(t)
	prompt := "Read README.md and report the project name. The answer must include the exact phrase \"Sample Service\". Cite README.md.\n\n" + jsonRules
	cfg := baseConfig("./cogni-results", []spec.AgentConfig{defaultAgent("default", model)}, "default", []spec.TaskConfig{{
		ID:     "t12",
		Type:   "qa",
		Agent:  "default",
		Prompt: prompt,
	}})
	specPath := writeConfig(t, repoRoot, cfg)

	t.Setenv("LLM_API_KEY", "invalid-key")
	_, stderr, exitCode := runCLI(t, []string{"run", "--spec", specPath})
	if exitCode != ExitOK {
		t.Fatalf("expected exit %d, got %d (%s)", ExitOK, exitCode, stderr)
	}

	results, _ := resolveResults(t, repoRoot, outputDir(repoRoot, cfg.Repo.OutputDir), "HEAD")
	if results.Tasks[0].Status != "error" || results.Tasks[0].FailureReason == nil || *results.Tasks[0].FailureReason != "runtime_error" {
		t.Fatalf("expected runtime error failure, got %+v", results.Tasks[0])
	}
}
