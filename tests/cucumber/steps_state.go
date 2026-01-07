//go:build cucumber
// +build cucumber

package cucumber

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/cucumber/godog"
)

// featureState holds scenario state for cucumber CLI tests.
type featureState struct {
	repoDir     string
	configPath  string
	previousWD  string
	previousEnv map[string]*string
	stdout      bytes.Buffer
	stderr      bytes.Buffer
	exitCode    int
	initialized bool
}

// InitializeScenario wires cucumber steps to the feature state.
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

// reset clears buffers and resets state before each scenario.
func (s *featureState) reset() {
	s.stdout.Reset()
	s.stderr.Reset()
	s.exitCode = 0
	s.previousEnv = map[string]*string{}
	s.initialized = false
}

// cleanup restores environment and removes temporary files.
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

// setEnv records and sets an environment variable for the scenario.
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
