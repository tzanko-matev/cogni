//go:build cucumber

package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/cucumber/godog"

	"cogni/internal/runner"
	"cogni/internal/ui/live"
)

// TestLiveUIScenarios runs the live UI feature scenarios.
func TestLiveUIScenarios(t *testing.T) {
	featurePath := filepath.Join("..", "..", "spec", "features", "output-live-ui", "testing.feature")
	suite := godog.TestSuite{
		Name:                "output-live-ui",
		ScenarioInitializer: InitializeLiveUIScenario,
		Options: &godog.Options{
			Format:    "pretty",
			Paths:     []string{featurePath},
			Strict:    true,
			TestingT:  t,
			Randomize: 0,
		},
	}
	if suite.Run() != 0 {
		t.Fatalf("non-zero godog status")
	}
}

// InitializeLiveUIScenario wires steps for live UI scenarios.
func InitializeLiveUIScenario(ctx *godog.ScenarioContext) {
	state := &liveUIScenarioState{}
	orig := isTerminal
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		state.reset()
		isTerminal = func(io.Writer) bool { return state.isTTY }
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		isTerminal = orig
		return ctx, nil
	})

	ctx.Step(`^a TTY stdout$`, state.givenTTY)
	ctx.Step(`^stdout is not a TTY$`, state.givenNonTTY)
	ctx.Step(`^a question_eval task with (\d+) questions$`, state.givenQuestionEvalTask)
	ctx.Step(`^a question that invokes a tool$`, state.givenQuestionWithTool)
	ctx.Step(`^I run "([^"]+)"$`, state.whenIRun)
	ctx.Step(`^a live UI is shown$`, state.thenLiveUIShown)
	ctx.Step(`^the UI lists each question with a status$`, state.thenQuestionStatuses)
	ctx.Step(`^the UI shows a tool call status for that question$`, state.thenToolStatusShown)
	ctx.Step(`^the output uses plain summary text$`, state.thenPlainOutput)
}

type liveUIScenarioState struct {
	isTTY    bool
	decision uiModeDecision
	uiState  live.State
}

// reset clears scenario state.
func (s *liveUIScenarioState) reset() {
	s.isTTY = false
	s.decision = uiModeDecision{}
	s.uiState = live.State{}
}

// givenTTY marks stdout as a TTY.
func (s *liveUIScenarioState) givenTTY() error {
	s.isTTY = true
	return nil
}

// givenNonTTY marks stdout as non-TTY.
func (s *liveUIScenarioState) givenNonTTY() error {
	s.isTTY = false
	return nil
}

// givenQuestionEvalTask seeds queued questions for UI state.
func (s *liveUIScenarioState) givenQuestionEvalTask(count int) error {
	if count < 1 {
		return nil
	}
	now := time.Now()
	for i := 0; i < count; i++ {
		s.uiState = live.Reduce(s.uiState, runner.QuestionEvent{
			TaskID:        "task-1",
			QuestionIndex: i,
			QuestionText:  "Question",
			Type:          runner.QuestionQueued,
			EmittedAt:     now,
		})
	}
	return nil
}

// givenQuestionWithTool seeds tool activity for a question.
func (s *liveUIScenarioState) givenQuestionWithTool() error {
	now := time.Now()
	s.uiState = live.Reduce(s.uiState, runner.QuestionEvent{
		TaskID:        "task-1",
		QuestionIndex: 0,
		QuestionText:  "Question",
		Type:          runner.QuestionRunning,
		EmittedAt:     now,
	})
	s.uiState = live.Reduce(s.uiState, runner.QuestionEvent{
		TaskID:        "task-1",
		QuestionIndex: 0,
		QuestionText:  "Question",
		Type:          runner.QuestionToolStart,
		ToolName:      "search",
		EmittedAt:     now,
	})
	return nil
}

// whenIRun evaluates UI mode decision for the scenario.
func (s *liveUIScenarioState) whenIRun(_ string) error {
	decision, err := resolveUIMode("auto", false, nil)
	if err != nil {
		return err
	}
	s.decision = decision
	return nil
}

// thenLiveUIShown asserts the live UI is enabled.
func (s *liveUIScenarioState) thenLiveUIShown() error {
	if !s.decision.useLive {
		return fmt.Errorf("expected live UI to be enabled")
	}
	return nil
}

// thenQuestionStatuses asserts that question rows exist.
func (s *liveUIScenarioState) thenQuestionStatuses() error {
	if len(s.uiState.Rows) == 0 {
		return fmt.Errorf("expected question rows")
	}
	return nil
}

// thenToolStatusShown asserts tool activity is recorded.
func (s *liveUIScenarioState) thenToolStatusShown() error {
	if len(s.uiState.Rows) == 0 || !s.uiState.Rows[0].HasTool {
		return fmt.Errorf("expected tool status to be set")
	}
	return nil
}

// thenPlainOutput asserts the live UI is disabled.
func (s *liveUIScenarioState) thenPlainOutput() error {
	if s.decision.useLive {
		return fmt.Errorf("expected plain output")
	}
	return nil
}
