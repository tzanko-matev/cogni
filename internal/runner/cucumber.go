package runner

import (
	"context"
	"errors"
	"io"

	"cogni/internal/agent"
	"cogni/internal/cucumber"
	"cogni/internal/spec"
)

// runCucumberTask executes a cucumber evaluation task end-to-end.
func runCucumberTask(
	ctx context.Context,
	repoRoot string,
	task taskRun,
	adapters map[string]spec.AdapterConfig,
	toolDefs []agent.ToolDefinition,
	executor agent.ToolExecutor,
	providerFactory ProviderFactory,
	tokenCounter agent.TokenCounter,
	verbose bool,
	verboseWriter io.Writer,
	noColor bool,
) TaskResult {
	result := TaskResult{TaskID: task.Task.ID, Type: task.Task.Type}
	adapter, ok := adapters[task.Task.Adapter]
	if !ok {
		reason := "invalid_adapter"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	featurePaths, err := cucumber.ExpandFeaturePaths(repoRoot, task.Task.Features)
	if err != nil {
		reason := "invalid_features"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	index, err := cucumber.BuildExampleIndex(repoRoot, featurePaths)
	if err != nil {
		reason := "feature_parse_error"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}
	examples := index.Examples()

	groundTruth, err := loadCucumberGroundTruth(ctx, repoRoot, adapter, featurePaths, index, examples)
	if err != nil {
		reason := "adapter_error"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	exampleResults := make([]CucumberExample, 0, len(examples))
	featureRuns := make([]CucumberFeatureRun, 0)
	correctCount := 0
	implementedCount := 0
	notImplementedCount := 0
	var failureReason *string

	features, examplesByFeature := groupExamplesByFeature(examples)

	for _, featurePath := range features {
		featureExamples := examplesByFeature[featurePath]
		if len(featureExamples) == 0 {
			continue
		}
		featureRun, responseMap, parseErr, runErr := runFeatureBatch(
			ctx,
			repoRoot,
			task,
			featurePath,
			featureExamples,
			toolDefs,
			executor,
			providerFactory,
			tokenCounter,
			verbose,
			verboseWriter,
			noColor,
		)
		featureRuns = append(featureRuns, featureRun)

		if runErr != nil {
			var inputErr featureInputError
			if errors.As(runErr, &inputErr) {
				reason := "invalid_features"
				result.Status = "error"
				result.FailureReason = &reason
				return result
			}
			var providerErr featureProviderError
			if errors.As(runErr, &providerErr) {
				reason := "runtime_error"
				result.Status = "error"
				result.FailureReason = &reason
				return result
			}
			reason := "runtime_error"
			if errors.Is(runErr, agent.ErrBudgetExceeded) {
				reason = "budget_exceeded"
			}
			if failureReason == nil {
				failureReason = &reason
			}
		} else if parseErr != nil {
			reason := "invalid_agent_response"
			if failureReason == nil {
				failureReason = &reason
			}
		}

		evalResult, err := evaluateFeatureExamples(featureExamples, groundTruth, responseMap, parseErr)
		if err != nil {
			reason := "adapter_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		exampleResults = append(exampleResults, evalResult.examples...)
		correctCount += evalResult.correctCount
		implementedCount += evalResult.implementedCount
		notImplementedCount += evalResult.notImplementedCount
	}

	total := len(exampleResults)
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correctCount) / float64(total)
	}
	result.Cucumber = &CucumberEval{
		AdapterID:   task.Task.Adapter,
		AdapterType: adapter.Type,
		FeatureRuns: featureRuns,
		Examples:    exampleResults,
		Summary: CucumberSummary{
			ExamplesTotal:     total,
			ExamplesCorrect:   correctCount,
			ExamplesIncorrect: total - correctCount,
			Accuracy:          accuracy,
			ImplementedTotal:  implementedCount,
			NotImplemented:    notImplementedCount,
		},
	}

	if total == 0 {
		reason := "no_examples"
		result.Status = "error"
		result.FailureReason = &reason
		return result
	}

	if failureReason != nil {
		result.FailureReason = failureReason
		switch *failureReason {
		case "runtime_error", "invalid_agent_response":
			result.Status = "error"
		default:
			result.Status = "fail"
		}
		return result
	}

	if correctCount == total {
		result.Status = "pass"
		return result
	}
	reason := "incorrect_examples"
	result.Status = "fail"
	result.FailureReason = &reason
	return result
}
