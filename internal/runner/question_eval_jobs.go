package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cogni/internal/agent"
	"cogni/internal/agent/call"
	"cogni/internal/question"
	"cogni/pkg/ratelimiter"
)

// questionJobDeps bundles dependencies for executing a single question job.
type questionJobDeps struct {
	repoRoot        string
	task            taskRun
	toolDefs        []agent.ToolDefinition
	executor        agent.ToolExecutor
	providerFactory ProviderFactory
	tokenCounter    agent.TokenCounter
	compaction      agent.CompactionConfig
	verbose         bool
	verboseWriter   io.Writer
	verboseLog      io.Writer
	noColor         bool
	maxOutputTokens uint64
	questionTotal   int
}

// questionJobResult captures the outcome of a question evaluation job.
type questionJobResult struct {
	index          int
	result         QuestionResult
	correct        bool
	runtimeError   bool
	budgetExceeded bool
	actualTokens   uint64
	runErr         error
}

// runQuestionJobsSequential executes questions one at a time through the scheduler.
func runQuestionJobsSequential(ctx context.Context, sched *ratelimiter.Scheduler, questions []question.Question, deps questionJobDeps) ([]QuestionResult, int, bool, bool) {
	results := make([]QuestionResult, 0, len(questions))
	correctCount := 0
	runtimeError := false
	budgetExceeded := false
	for index, item := range questions {
		promptText := buildQuestionPrompt(item)
		resultCh := make(chan questionJobResult, 1)
		job := ratelimiter.Job{
			JobID:           fmt.Sprintf("%s-%d", deps.task.Task.ID, index+1),
			Provider:        deps.task.Agent.Provider,
			Model:           deps.task.Model,
			Prompt:          promptText,
			MaxOutputTokens: deps.maxOutputTokens,
			Execute: func(_ context.Context) (uint64, error) {
				jobResult := executeQuestionJob(ctx, deps, index, item, promptText)
				resultCh <- jobResult
				return jobResult.actualTokens, jobResult.runErr
			},
		}
		sched.Submit(job)
		jobResult := <-resultCh
		results = append(results, jobResult.result)
		if jobResult.correct {
			correctCount++
		}
		if jobResult.runtimeError {
			runtimeError = true
		}
		if jobResult.budgetExceeded {
			budgetExceeded = true
		}
	}
	return results, correctCount, runtimeError, budgetExceeded
}

// runQuestionJobsConcurrent executes question jobs concurrently and preserves ordering.
func runQuestionJobsConcurrent(ctx context.Context, sched *ratelimiter.Scheduler, questions []question.Question, deps questionJobDeps) ([]QuestionResult, int, bool, bool) {
	results := make([]QuestionResult, len(questions))
	resultCh := make(chan questionJobResult, len(questions))

	for index, item := range questions {
		idx := index
		questionItem := item
		promptText := buildQuestionPrompt(questionItem)
		job := ratelimiter.Job{
			JobID:           fmt.Sprintf("%s-%d", deps.task.Task.ID, idx+1),
			Provider:        deps.task.Agent.Provider,
			Model:           deps.task.Model,
			Prompt:          promptText,
			MaxOutputTokens: deps.maxOutputTokens,
			Execute: func(_ context.Context) (uint64, error) {
				jobResult := executeQuestionJob(ctx, deps, idx, questionItem, promptText)
				resultCh <- jobResult
				return jobResult.actualTokens, jobResult.runErr
			},
		}
		sched.Submit(job)
	}

	correctCount := 0
	runtimeError := false
	budgetExceeded := false
	for i := 0; i < len(questions); i++ {
		jobResult := <-resultCh
		results[jobResult.index] = jobResult.result
		if jobResult.correct {
			correctCount++
		}
		if jobResult.runtimeError {
			runtimeError = true
		}
		if jobResult.budgetExceeded {
			budgetExceeded = true
		}
	}
	return results, correctCount, runtimeError, budgetExceeded
}

// executeQuestionJob runs a single question evaluation and returns its outcome.
func executeQuestionJob(ctx context.Context, deps questionJobDeps, index int, item question.Question, promptText string) questionJobResult {
	logVerbose(deps.verbose, deps.verboseWriter, deps.verboseLog, deps.noColor, styleTask,
		fmt.Sprintf("Task %s question %d/%d agent=%s model=%s", deps.task.Task.ID, index+1, deps.questionTotal, deps.task.AgentID, deps.task.Model))
	provider, err := deps.providerFactory(deps.task.Agent, deps.task.Model)
	if err != nil {
		result := buildQuestionResult(item, call.RunMetrics{}, err)
		return questionJobResult{index: index, result: result, runtimeError: true, actualTokens: 0, runErr: err}
	}
	session := newSession(deps.task, deps.repoRoot, deps.toolDefs, deps.verbose)
	callResult, runErr := call.RunCall(ctx, session, provider, deps.executor, promptText, call.RunOptions{
		TokenCounter: deps.tokenCounter,
		Compaction:   deps.compaction,
		Limits: call.RunLimits{
			MaxSteps:   limitOrDefault(deps.task.Task.Budget.MaxSteps, deps.task.Agent.MaxSteps),
			MaxSeconds: time.Duration(deps.task.Task.Budget.MaxSeconds) * time.Second,
			MaxTokens:  deps.task.Task.Budget.MaxTokens,
		},
		Verbose:          deps.verbose,
		VerboseWriter:    deps.verboseWriter,
		VerboseLogWriter: deps.verboseLog,
		NoColor:          deps.noColor,
	}, nil)
	metrics := callResult.Metrics
	if runErr != nil {
		logVerbose(deps.verbose, deps.verboseWriter, deps.verboseLog, deps.noColor, styleError,
			fmt.Sprintf("Task %s question %d error=%v", deps.task.Task.ID, index+1, runErr))
	}
	logVerbose(deps.verbose, deps.verboseWriter, deps.verboseLog, deps.noColor, styleMetrics,
		fmt.Sprintf("Metrics task=%s question=%d steps=%d tokens=%d wall_time=%s tool_calls=%s", deps.task.Task.ID, index+1, metrics.Steps, metrics.Tokens, metrics.WallTime, formatToolCounts(metrics.ToolCalls)))

	result := buildQuestionResult(item, metrics, runErr)
	jobResult := questionJobResult{
		index:        index,
		result:       result,
		actualTokens: uint64(metrics.Tokens),
		runErr:       runErr,
	}
	if runErr != nil {
		if errors.Is(runErr, call.ErrBudgetExceeded) {
			jobResult.budgetExceeded = true
		} else {
			jobResult.runtimeError = true
		}
		return jobResult
	}

	answer, parseErr := question.ParseAnswerFromOutput(callResult.Output)
	if parseErr != nil {
		jobResult.result.ParseError = parseErr.Error()
		return jobResult
	}
	jobResult.result.AgentAnswer = answer.Raw
	if isCorrectAnswer(answer.Normalized, item.CorrectAnswers) {
		jobResult.result.Correct = true
		jobResult.correct = true
	}
	return jobResult
}

// buildQuestionResult assembles a QuestionResult from metrics and errors.
func buildQuestionResult(item question.Question, metrics call.RunMetrics, runErr error) QuestionResult {
	result := QuestionResult{
		ID:                item.ID,
		Question:          item.Prompt,
		Answers:           item.Answers,
		CorrectAnswers:    item.CorrectAnswers,
		Correct:           false,
		TokensTotal:       metrics.Tokens,
		WallTimeSeconds:   metrics.WallTime.Seconds(),
		AgentSteps:        metrics.Steps,
		ToolCalls:         metrics.ToolCalls,
		Compactions:       metrics.Compactions,
		LastSummaryTokens: metrics.LastSummaryTokens,
	}
	if runErr != nil {
		result.RunError = runErr.Error()
	}
	return result
}
