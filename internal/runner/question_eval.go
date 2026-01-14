package runner

import (
	"context"
	"io"
	"time"

	"cogni/internal/agent"
	"cogni/internal/question"
	"cogni/internal/ratelimit"
	"cogni/internal/spec"
	"cogni/pkg/ratelimiter"
)

// runQuestionTask executes a question evaluation task end-to-end.
func runQuestionTask(
	ctx context.Context,
	repoRoot string,
	cfg spec.Config,
	task taskRun,
	limiter ratelimiter.Limiter,
	toolDefs []agent.ToolDefinition,
	executor agent.ToolExecutor,
	providerFactory ProviderFactory,
	tokenCounter agent.TokenCounter,
	verbose bool,
	verboseWriter io.Writer,
	verboseLogWriter io.Writer,
	noColor bool,
	observer RunObserver,
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

	jobObserver := newQuestionJobObserver(observer, task.Task.ID, questionSpec.Questions)
	if jobObserver != nil {
		jobObserver.EmitQueuedAll()
	}

	workers := ratelimit.ResolveTaskWorkers(cfg, task.Task)
	scheduler := ratelimiter.NewSchedulerWithObserver(limiter, workers, jobObserver)
	maxOutputTokens := ratelimit.MaxOutputTokens(cfg, task.Task)
	verboseWriter, verboseLogWriter = wrapVerboseWriters(workers, verboseWriter, verboseLogWriter)
	deps := questionJobDeps{
		repoRoot:        repoRoot,
		task:            task,
		toolDefs:        toolDefs,
		executor:        executor,
		providerFactory: providerFactory,
		tokenCounter:    tokenCounter,
		compaction:      compactionConfig,
		verbose:         verbose,
		verboseWriter:   verboseWriter,
		verboseLog:      verboseLogWriter,
		noColor:         noColor,
		maxOutputTokens: maxOutputTokens,
		questionTotal:   len(questionSpec.Questions),
		observer:        jobObserver,
	}

	var (
		questionResults []QuestionResult
		correctCount    int
		runtimeError    bool
		budgetExceeded  bool
	)
	if workers <= 1 {
		questionResults, correctCount, runtimeError, budgetExceeded = runQuestionJobsSequential(ctx, scheduler, questionSpec.Questions, deps)
	} else {
		questionResults, correctCount, runtimeError, budgetExceeded = runQuestionJobsConcurrent(ctx, scheduler, questionSpec.Questions, deps)
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := scheduler.Shutdown(shutdownCtx); err != nil {
		runtimeError = true
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
