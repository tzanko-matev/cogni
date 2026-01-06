package cucumber

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cogni/internal/cli"

	"github.com/cucumber/godog"
)

type featureState struct {
	repoDir       string
	configPath    string
	previousWD    string
	previousEnv   map[string]*string
	stdout        bytes.Buffer
	stderr        bytes.Buffer
	exitCode      int
	initialized   bool
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &featureState{}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		state.reset()
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		state.cleanup()
		return ctx, nil
	})

	ctx.Step(`^a git repository with a valid Cogni configuration$`, state.aGitRepositoryWithValidConfig)
	ctx.Step(`^LLM provider credentials are available in the environment$`, state.llmCredentialsAreAvailable)
	ctx.Step(`^the config is invalid$`, state.theConfigIsInvalid)
	ctx.Step(`^I run "([^"]+)"$`, state.iRunCommand)
	ctx.Step(`^the output lists these commands:$`, state.theOutputListsCommands)
	ctx.Step(`^the exit code is non-zero$`, state.theExitCodeIsNonZero)
	ctx.Step(`^the error message points to the invalid field$`, state.theErrorMessagePointsToInvalidField)
}

func (s *featureState) reset() {
	s.stdout.Reset()
	s.stderr.Reset()
	s.exitCode = 0
	s.previousEnv = map[string]*string{}
	s.initialized = false
}

func (s *featureState) cleanup() {
	if s.previousWD != "" {
		_ = os.Chdir(s.previousWD)
	}
	for key, value := range s.previousEnv {
		if value == nil {
			_ = os.Unsetenv(key)
			continue
		}
		_ = os.Setenv(key, *value)
	}
	if s.repoDir != "" {
		_ = os.RemoveAll(s.repoDir)
	}
}

func (s *featureState) aGitRepositoryWithValidConfig() error {
	if s.initialized {
		return nil
	}
	dir, err := os.MkdirTemp("", "cogni-feature-*")
	if err != nil {
		return fmt.Errorf("create temp repo: %w", err)
	}
	s.repoDir = dir
	s.configPath = filepath.Join(dir, ".cogni", "config.yml")
	if err := os.MkdirAll(filepath.Dir(s.configPath), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := s.writeConfig(validConfigYAML()); err != nil {
		return err
	}
	if err := s.initGitRepo(dir); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working dir: %w", err)
	}
	s.previousWD = wd
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}
	s.initialized = true
	return nil
}

func (s *featureState) llmCredentialsAreAvailable() error {
	return s.setEnv("LLM_API_KEY", "test-key")
}

func (s *featureState) theConfigIsInvalid() error {
	if err := s.aGitRepositoryWithValidConfig(); err != nil {
		return err
	}
	return s.writeConfig(invalidConfigYAML())
}

func (s *featureState) iRunCommand(command string) error {
	args := strings.Fields(command)
	if len(args) == 0 {
		return fmt.Errorf("command is empty")
	}
	if args[0] == "cogni" {
		args = args[1:]
	}
	s.stdout.Reset()
	s.stderr.Reset()
	s.exitCode = cli.Run(args, &s.stdout, &s.stderr)
	return nil
}

func (s *featureState) theOutputListsCommands(table *godog.Table) error {
	output := s.stdout.String()
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			command := strings.TrimSpace(cell.Value)
			if command == "" {
				continue
			}
			if !strings.Contains(output, command) {
				return fmt.Errorf("expected command %q in output", command)
			}
		}
	}
	return nil
}

func (s *featureState) theExitCodeIsNonZero() error {
	if s.exitCode == 0 {
		return fmt.Errorf("expected non-zero exit code")
	}
	return nil
}

func (s *featureState) theErrorMessagePointsToInvalidField() error {
	errOutput := s.stderr.String()
	if !strings.Contains(errOutput, "version") {
		return fmt.Errorf("expected error to mention version, got %q", errOutput)
	}
	return nil
}

func (s *featureState) initGitRepo(dir string) error {
	if err := s.runGit(dir, "-c", "init.defaultBranch=main", "init"); err != nil {
		return err
	}
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("fixture"), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	if err := s.runGit(dir, "add", "README.md"); err != nil {
		return err
	}
	if err := s.runGit(dir, "commit", "-m", "initial"); err != nil {
		return err
	}
	return nil
}

func (s *featureState) runGit(dir string, args ...string) error {
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
		return fmt.Errorf("git %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *featureState) setEnv(key, value string) error {
	if s.previousEnv == nil {
		s.previousEnv = map[string]*string{}
	}
	if _, exists := s.previousEnv[key]; !exists {
		if current, ok := os.LookupEnv(key); ok {
			copy := current
			s.previousEnv[key] = &copy
		} else {
			s.previousEnv[key] = nil
		}
	}
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("set env %s: %w", key, err)
	}
	return nil
}

func (s *featureState) writeConfig(contents string) error {
	if s.configPath == "" {
		return fmt.Errorf("config path is not set")
	}
	if err := os.WriteFile(s.configPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func validConfigYAML() string {
	return `version: 1
repo:
  output_dir: ".cogni/results"

agents:
  - id: default
    type: builtin
    provider: "openrouter"
    model: "gpt-4.1-mini"

default_agent: "default"

tasks:
  - id: smoke_task
    type: qa
    agent: "default"
    prompt: "return valid JSON"
`
}

func invalidConfigYAML() string {
	return `version: 2
repo:
  output_dir: ".cogni/results"

agents:
  - id: default
    type: builtin
    provider: "openrouter"
    model: "gpt-4.1-mini"

default_agent: "default"

tasks:
  - id: smoke_task
    type: qa
    agent: "default"
    prompt: "return valid JSON"
`
}
