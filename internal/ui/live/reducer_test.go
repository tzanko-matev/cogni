package live

import (
	"testing"
	"time"

	"cogni/internal/runner"
	"cogni/internal/testutil"
)

// TestReduceQuestionLifecycle verifies core status transitions are recorded.
func TestReduceQuestionLifecycle(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		start := time.Now()
		state := State{}
		state = Reduce(state, event(0, runner.QuestionQueued, "", start))
		state = Reduce(state, event(0, runner.QuestionScheduled, "", start))
		state = Reduce(state, event(0, runner.QuestionReserving, "", start))
		state = Reduce(state, event(0, runner.QuestionRunning, "", start))
		state = Reduce(state, event(0, runner.QuestionParsing, "", start))
		done := event(0, runner.QuestionCorrect, "", start.Add(150*time.Millisecond))
		done.Tokens = 120
		state = Reduce(state, done)

		row := state.Rows[0]
		if row.Status != runner.QuestionCorrect {
			t.Fatalf("expected correct status, got %s", row.Status)
		}
		if row.Tokens != 120 {
			t.Fatalf("expected tokens to be set, got %d", row.Tokens)
		}
		if state.Counts.Correct != 1 {
			t.Fatalf("expected correct count, got %d", state.Counts.Correct)
		}
	})
}

// TestReduceWaitingIncrementsRetry verifies retry counts are tracked.
func TestReduceWaitingIncrementsRetry(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		state := State{}
		state = Reduce(state, event(0, runner.QuestionWaitingRateLimit, "", time.Now()))
		state = Reduce(state, event(0, runner.QuestionWaitingLimitDecreasing, "", time.Now()))
		row := state.Rows[0]
		if row.RetryCount != 2 {
			t.Fatalf("expected retries=2, got %d", row.RetryCount)
		}
		if state.Counts.Waiting != 1 {
			t.Fatalf("expected waiting count, got %d", state.Counts.Waiting)
		}
	})
}

// TestReduceTerminalErrors verifies parse and runtime error handling.
func TestReduceTerminalErrors(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		state := State{}
		parseEvt := event(0, runner.QuestionParseError, "parse failed", time.Now())
		state = Reduce(state, parseEvt)
		if state.Rows[0].Error == "" {
			t.Fatalf("expected parse error to be recorded")
		}
		runtimeEvt := event(1, runner.QuestionRuntimeError, "boom", time.Now())
		state = Reduce(state, runtimeEvt)
		if state.Rows[1].Status != runner.QuestionRuntimeError {
			t.Fatalf("expected runtime error status, got %s", state.Rows[1].Status)
		}
	})
}

// TestReduceToolStatus verifies tool activity updates are stored.
func TestReduceToolStatus(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		state := State{}
		state = Reduce(state, event(0, runner.QuestionRunning, "", time.Now()))
		toolStart := event(0, runner.QuestionToolStart, "", time.Now())
		toolStart.ToolName = "search"
		state = Reduce(state, toolStart)
		if !state.Rows[0].HasTool || state.Rows[0].Tool.State != "running" {
			t.Fatalf("expected running tool status")
		}
		toolFinish := event(0, runner.QuestionToolFinish, "", time.Now())
		toolFinish.ToolName = "search"
		toolFinish.ToolDuration = 250 * time.Millisecond
		state = Reduce(state, toolFinish)
		if state.Rows[0].Tool.State != "done" {
			t.Fatalf("expected done tool status")
		}
	})
}

// event builds a QuestionEvent for testing.
func event(index int, kind runner.QuestionEventType, errMsg string, when time.Time) runner.QuestionEvent {
	return runner.QuestionEvent{
		TaskID:        "task-1",
		QuestionIndex: index,
		QuestionID:    "",
		QuestionText:  "Question",
		Type:          kind,
		Error:         errMsg,
		EmittedAt:     when,
	}
}

// runWithTimeout executes a test body with a timeout.
func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	ctx := testutil.Context(t, timeout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("test timed out")
	}
}
