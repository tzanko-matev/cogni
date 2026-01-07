package runner

import (
	"context"
	"fmt"
	"io"
	"time"

	"cogni/internal/agent"
	"cogni/internal/eval"
)

// runTask executes a single non-cucumber task and records attempts.
func runTask(
	ctx context.Context,
	repoRoot string,
	task taskRun,
	toolsDefs []agent.ToolDefinition,
	executor agent.ToolExecutor,
	providerFactory ProviderFactory,
	tokenCounter agent.TokenCounter,
	repeat int,
	verbose bool,
	verboseWriter io.Writer,
	verboseLogWriter io.Writer,
	noColor bool,
) TaskResult {
	result := TaskResult{TaskID: task.Task.ID, Type: task.Task.Type}
	attempts := make([]AttemptResult, 0, repeat)
	var failureReason *string

	for attemptIndex := 1; attemptIndex <= repeat; attemptIndex++ {
		logVerbose(verbose, verboseWriter, verboseLogWriter, noColor, styleTask, fmt.Sprintf("Task %s attempt %d/%d agent=%s model=%s", task.Task.ID, attemptIndex, repeat, task.AgentID, task.Model))
		provider, err := providerFactory(task.Agent, task.Model)
		if err != nil {
			reason := "runtime_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		session := newSession(task, repoRoot, toolsDefs, verbose)
		runMetrics, runErr := agent.RunTurn(ctx, session, provider, executor, task.Task.Prompt, agent.RunOptions{
			TokenCounter:    tokenCounter,
			CompactionLimit: task.Task.Budget.MaxTokens,
			Limits: agent.RunLimits{
				MaxSteps:   limitOrDefault(task.Task.Budget.MaxSteps, task.Agent.MaxSteps),
				MaxSeconds: time.Duration(task.Task.Budget.MaxSeconds) * time.Second,
				MaxTokens:  task.Task.Budget.MaxTokens,
			},
			Verbose:          verbose,
			VerboseWriter:    verboseWriter,
			VerboseLogWriter: verboseLogWriter,
			NoColor:          noColor,
		})
		if runErr != nil {
			logVerbose(verbose, verboseWriter, verboseLogWriter, noColor, styleError, fmt.Sprintf("Task %s attempt %d error=%v", task.Task.ID, attemptIndex, runErr))
		}
		logVerbose(verbose, verboseWriter, verboseLogWriter, noColor, styleMetrics, fmt.Sprintf("Metrics task=%s attempt=%d steps=%d tokens=%d wall_time=%s tool_calls=%s", task.Task.ID, attemptIndex, runMetrics.Steps, runMetrics.Tokens, runMetrics.WallTime, formatToolCounts(runMetrics.ToolCalls)))

		output, ok := latestAssistantMessage(session.History)
		if !ok {
			output = ""
		}

		evalResult := eval.QAResult{
			Status:        "error",
			FailureReason: "runtime_error",
			SchemaValid:   false,
			CitationValid: false,
		}
		if runErr == nil && task.Task.Type == "qa" {
			evalResult = eval.EvaluateQA(output, eval.QAConfig{
				JSONSchemaPath:    task.Task.Eval.JSONSchema,
				MustContain:       task.Task.Eval.MustContainStrings,
				ValidateCitations: task.Task.Eval.ValidateCitations,
				RepoRoot:          repoRoot,
			})
		}

		attempt := AttemptResult{
			Attempt:         attemptIndex,
			Status:          evalResult.Status,
			AgentID:         task.AgentID,
			Model:           task.Model,
			TokensIn:        0,
			TokensOut:       0,
			TokensTotal:     runMetrics.Tokens,
			WallTimeSeconds: runMetrics.WallTime.Seconds(),
			AgentSteps:      runMetrics.Steps,
			ToolCalls:       runMetrics.ToolCalls,
			UniqueFilesRead: 0,
			Eval: EvalResult{
				SchemaValid:        evalResult.SchemaValid,
				CitationValid:      evalResult.CitationValid,
				SchemaErrors:       evalResult.Artifacts.SchemaErrors,
				CitationErrors:     evalResult.Artifacts.CitationErrors,
				MustContainMissing: evalResult.Artifacts.MustContainMissing,
			},
		}
		attempts = append(attempts, attempt)

		if runErr != nil {
			reason := "runtime_error"
			if runErr == agent.ErrBudgetExceeded {
				reason = "budget_exceeded"
			}
			failureReason = &reason
		} else if evalResult.Status != "pass" {
			reason := evalResult.FailureReason
			failureReason = &reason
		}
	}

	result.Attempts = attempts
	if failureReason == nil {
		result.Status = "pass"
		return result
	}
	if *failureReason == "runtime_error" {
		result.Status = "error"
	} else {
		result.Status = "fail"
	}
	result.FailureReason = failureReason
	return result
}

// newSession constructs a session for a task run.
func newSession(task taskRun, repoRoot string, toolsDefs []agent.ToolDefinition, verbose bool) *agent.Session {
	ctx := agent.TurnContext{
		Model:                    task.Model,
		ModelFamily:              agent.ModelFamily{},
		Tools:                    toolsDefs,
		ApprovalPolicy:           "",
		SandboxPolicy:            agent.SandboxPolicy{},
		CWD:                      repoRoot,
		DeveloperInstructions:    "",
		UserInstructions:         "",
		BaseInstructionsOverride: "",
		OutputSchema:             "",
		Features:                 agent.FeatureFlags{},
		Verbose:                  verbose,
	}
	return &agent.Session{
		Ctx:     ctx,
		History: agent.BuildInitialContext(ctx),
	}
}

// latestAssistantMessage returns the most recent assistant text message.
func latestAssistantMessage(history []agent.HistoryItem) (string, bool) {
	for i := len(history) - 1; i >= 0; i-- {
		item := history[i]
		if item.Role != "assistant" {
			continue
		}
		text, ok := item.Content.(agent.HistoryText)
		if ok {
			return text.Text, true
		}
	}
	return "", false
}
