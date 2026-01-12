package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"cogni/internal/agent"
	"cogni/internal/agent/call"
	"cogni/internal/question"
)

// runQuestionTask executes a question evaluation task end-to-end.
func runQuestionTask(
	ctx context.Context,
	repoRoot string,
	task taskRun,
	toolDefs []agent.ToolDefinition,
	executor agent.ToolExecutor,
	providerFactory ProviderFactory,
	tokenCounter agent.TokenCounter,
	verbose bool,
	verboseWriter io.Writer,
	verboseLogWriter io.Writer,
	noColor bool,
) TaskResult {
	result := TaskResult{TaskID: task.Task.ID, Type: task.Task.Type}
	questionsPath := resolveQuestionsFile(repoRoot, task.Task.QuestionsFile)
	questionSpec, err := question.LoadSpec(questionsPath)
	if err != nil {
		reason := "invalid_questions_file"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	if len(questionSpec.Questions) == 0 {
		reason := "no_questions"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	compactionConfig, compactionErr := buildCompactionConfig(task.Task, repoRoot)
	if compactionErr != nil {
		reason := "runtime_error"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	questionResults := make([]QuestionResult, 0, len(questionSpec.Questions))
	correctCount := 0
	runtimeError := false
	budgetExceeded := false

	for index, questionItem := range questionSpec.Questions {
		logVerbose(verbose, verboseWriter, verboseLogWriter, noColor, styleTask, fmt.Sprintf("Task %s question %d/%d agent=%s model=%s", task.Task.ID, index+1, len(questionSpec.Questions), task.AgentID, task.Model))
		provider, err := providerFactory(task.Agent, task.Model)
		if err != nil {
			reason := "runtime_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		session := newSession(task, repoRoot, toolDefs, verbose)
		promptText := buildQuestionPrompt(questionItem)
		callResult, runErr := call.RunCall(ctx, session, provider, executor, promptText, call.RunOptions{
			TokenCounter: tokenCounter,
			Compaction:   compactionConfig,
			Limits: call.RunLimits{
				MaxSteps:   limitOrDefault(task.Task.Budget.MaxSteps, task.Agent.MaxSteps),
				MaxSeconds: time.Duration(task.Task.Budget.MaxSeconds) * time.Second,
				MaxTokens:  task.Task.Budget.MaxTokens,
			},
			Verbose:          verbose,
			VerboseWriter:    verboseWriter,
			VerboseLogWriter: verboseLogWriter,
			NoColor:          noColor,
		}, nil)
		runMetrics := callResult.Metrics
		if runErr != nil {
			logVerbose(verbose, verboseWriter, verboseLogWriter, noColor, styleError, fmt.Sprintf("Task %s question %d error=%v", task.Task.ID, index+1, runErr))
		}
		logVerbose(verbose, verboseWriter, verboseLogWriter, noColor, styleMetrics, fmt.Sprintf("Metrics task=%s question=%d steps=%d tokens=%d wall_time=%s tool_calls=%s", task.Task.ID, index+1, runMetrics.Steps, runMetrics.Tokens, runMetrics.WallTime, formatToolCounts(runMetrics.ToolCalls)))

		output := callResult.Output

		questionResult := QuestionResult{
			ID:                questionItem.ID,
			Question:          questionItem.Prompt,
			Answers:           questionItem.Answers,
			CorrectAnswers:    questionItem.CorrectAnswers,
			Correct:           false,
			TokensTotal:       runMetrics.Tokens,
			WallTimeSeconds:   runMetrics.WallTime.Seconds(),
			AgentSteps:        runMetrics.Steps,
			ToolCalls:         runMetrics.ToolCalls,
			Compactions:       runMetrics.Compactions,
			LastSummaryTokens: runMetrics.LastSummaryTokens,
		}

		if runErr != nil {
			questionResult.RunError = runErr.Error()
			if errors.Is(runErr, call.ErrBudgetExceeded) {
				budgetExceeded = true
			} else {
				runtimeError = true
			}
			questionResults = append(questionResults, questionResult)
			continue
		}

		answer, err := question.ParseAnswerFromOutput(output)
		if err != nil {
			questionResult.ParseError = err.Error()
			questionResults = append(questionResults, questionResult)
			continue
		}
		questionResult.AgentAnswer = answer.Raw
		if isCorrectAnswer(answer.Normalized, questionItem.CorrectAnswers) {
			questionResult.Correct = true
			correctCount++
		}
		questionResults = append(questionResults, questionResult)
	}

	total := len(questionResults)
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correctCount) / float64(total)
	}
	result.QuestionEval = &QuestionEval{
		QuestionsFile: task.Task.QuestionsFile,
		Questions:     questionResults,
		Summary: QuestionSummary{
			QuestionsTotal:     total,
			QuestionsCorrect:   correctCount,
			QuestionsIncorrect: total - correctCount,
			Accuracy:           accuracy,
		},
	}

	if runtimeError {
		reason := "runtime_error"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}
	if budgetExceeded {
		reason := "budget_exceeded"
		result.Status = "fail"
		result.FailureReason = &reason
		return result
	}
	if correctCount == total {
		result.Status = "pass"
		return result
	}
	reason := "incorrect_answers"
	result.Status = "fail"
	result.FailureReason = &reason
	return result
}

// resolveQuestionsFile resolves the questions file path against the repo root.
func resolveQuestionsFile(repoRoot, questionsFile string) string {
	if strings.TrimSpace(questionsFile) == "" || filepath.IsAbs(questionsFile) {
		return questionsFile
	}
	return filepath.Join(repoRoot, questionsFile)
}

// isCorrectAnswer checks whether a normalized answer matches any correct answer.
func isCorrectAnswer(normalized string, correctAnswers []string) bool {
	for _, candidate := range correctAnswers {
		if question.NormalizeAnswerText(candidate) == normalized {
			return true
		}
	}
	return false
}
