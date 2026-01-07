//go:build cucumber
// +build cucumber

package cucumber

import (
	"fmt"
	"os"
	"path/filepath"
)

// aGitRepositoryWithValidConfig sets up a temp repo with a valid config.
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

// llmCredentialsAreAvailable stubs the LLM API key for tests.
func (s *featureState) llmCredentialsAreAvailable() error {
	return s.setEnv("LLM_API_KEY", "test-key")
}

// theConfigIsInvalid replaces the config with an invalid configuration.
func (s *featureState) theConfigIsInvalid() error {
	if err := s.aGitRepositoryWithValidConfig(); err != nil {
		return err
	}
	return s.writeConfig(invalidConfigYAML())
}

// writeConfig persists configuration content to the repo config path.
func (s *featureState) writeConfig(contents string) error {
	if s.configPath == "" {
		return fmt.Errorf("config path is not set")
	}
	if err := os.WriteFile(s.configPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// validConfigYAML returns a minimal valid config for cucumber tests.
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

// invalidConfigYAML returns a config with an unsupported version.
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
