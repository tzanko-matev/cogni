package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cogni/internal/agent"
	"cogni/internal/cucumber"
	"cogni/internal/spec"
)

type cucumberGroundTruth struct {
	Implemented bool
}

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

	groundTruth := make(map[string]cucumberGroundTruth)
	switch adapter.Type {
	case "cucumber":
		features, err := cucumber.RunGodogJSON(ctx, repoRoot, featurePaths, adapter.Tags)
		if err != nil {
			reason := "adapter_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		normalized, err := cucumber.NormalizeGodogResults(repoRoot, features, index)
		if err != nil {
			reason := "adapter_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		for _, entry := range normalized {
			groundTruth[entry.ExampleID] = cucumberGroundTruth{Implemented: entry.Status == "passed"}
		}
	case "cucumber_manual":
		expectationsDir := strings.TrimSpace(adapter.ExpectationsDir)
		if expectationsDir != "" && !filepath.IsAbs(expectationsDir) {
			expectationsDir = filepath.Join(repoRoot, expectationsDir)
		}
		expectations, err := cucumber.LoadExpectations(expectationsDir)
		if err != nil {
			reason := "adapter_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		if err := cucumber.ValidateExpectations(expectations, examples); err != nil {
			reason := "adapter_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		for id, expectation := range expectations {
			groundTruth[id] = cucumberGroundTruth{Implemented: expectation.Implemented}
		}
	default:
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

	features := make([]string, 0)
	examplesByFeature := make(map[string][]cucumber.Example)
	seenFeatures := make(map[string]struct{})
	for _, example := range examples {
		if _, seen := seenFeatures[example.FeaturePath]; !seen {
			features = append(features, example.FeaturePath)
			seenFeatures[example.FeaturePath] = struct{}{}
		}
		examplesByFeature[example.FeaturePath] = append(examplesByFeature[example.FeaturePath], example)
	}

	for _, featurePath := range features {
		featureExamples := examplesByFeature[featurePath]
		if len(featureExamples) == 0 {
			continue
		}
		expectedIDs := make([]string, 0, len(featureExamples))
		for _, example := range featureExamples {
			expectedIDs = append(expectedIDs, example.ID)
		}

		featureText, err := os.ReadFile(featurePath)
		if err != nil {
			reason := "invalid_features"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}

		prompt := renderCucumberPrompt(task.Task.PromptTemplate, featurePath, string(featureText), expectedIDs)
		provider, err := providerFactory(task.Agent, task.Model)
		if err != nil {
			reason := "runtime_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		session := newSession(task, repoRoot, toolDefs, verbose)
		runMetrics, runErr := agent.RunTurn(ctx, session, provider, executor, prompt, agent.RunOptions{
			TokenCounter:    tokenCounter,
			CompactionLimit: task.Task.Budget.MaxTokens,
			Limits: agent.RunLimits{
				MaxSteps:   limitOrDefault(task.Task.Budget.MaxSteps, task.Agent.MaxSteps),
				MaxSeconds: time.Duration(task.Task.Budget.MaxSeconds) * time.Second,
				MaxTokens:  task.Task.Budget.MaxTokens,
			},
			Verbose:       verbose,
			VerboseWriter: verboseWriter,
			NoColor:       noColor,
		})

		featureRuns = append(featureRuns, CucumberFeatureRun{
			FeaturePath:     featurePath,
			ExamplesTotal:   len(featureExamples),
			TokensTotal:     runMetrics.Tokens,
			WallTimeSeconds: runMetrics.WallTime.Seconds(),
			AgentSteps:      runMetrics.Steps,
			ToolCalls:       runMetrics.ToolCalls,
		})

		output, ok := latestAssistantMessage(session.History)
		if !ok {
			output = ""
		}

		var responseMap map[string]cucumber.AgentResponse
		var parseErr error
		if runErr == nil {
			response, err := cucumber.ParseAgentBatchResponse(output)
			if err != nil {
				parseErr = err
			} else {
				responseMap, parseErr = cucumber.ValidateAgentBatchResponse(expectedIDs, response)
			}
		} else {
			parseErr = runErr
		}

		if runErr != nil {
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

		var batchValidation cucumber.BatchValidationError
		hasBatchValidation := errors.As(parseErr, &batchValidation)

		for _, example := range featureExamples {
			gt, ok := groundTruth[example.ID]
			if !ok {
				reason := "adapter_error"
				result.Status = "error"
				result.FailureReason = &reason
				return result
			}
			if gt.Implemented {
				implementedCount++
			} else {
				notImplementedCount++
			}

			agentResult := &CucumberAgentResult{}
			correct := false
			if parseErr == nil || hasBatchValidation {
				if response, ok := responseMap[example.ID]; ok {
					agentResult.ExampleID = response.ExampleID
					agentResult.Implemented = response.Implemented
					agentResult.Notes = response.Notes
					agentResult.Evidence = convertEvidence(response.Evidence)
					correct = response.Implemented == gt.Implemented
				} else if parseErr != nil {
					agentResult.ParseError = fmt.Sprintf("missing example_id %q", example.ID)
				}
			} else {
				agentResult.ParseError = parseErr.Error()
			}

			if correct {
				correctCount++
			}

			exampleResults = append(exampleResults, CucumberExample{
				ExampleID:    example.ID,
				FeaturePath:  example.FeaturePath,
				ScenarioName: example.ScenarioName,
				ScenarioLine: example.ScenarioLine,
				ExampleLine:  example.ExampleLine,
				GroundTruth:  truthLabel(gt.Implemented),
				Agent:        agentResult,
				Correct:      correct,
			})
		}
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

func renderCucumberPrompt(template, featurePath, featureText string, exampleIDs []string) string {
	replacer := strings.NewReplacer(
		"{feature_path}", featurePath,
		"{feature_text}", featureText,
		"{example_ids}", strings.Join(exampleIDs, "\n"),
	)
	return replacer.Replace(template)
}

func truthLabel(implemented bool) string {
	if implemented {
		return "implemented"
	}
	return "not_implemented"
}

func convertEvidence(items []cucumber.Evidence) []CucumberEvidence {
	if len(items) == 0 {
		return nil
	}
	out := make([]CucumberEvidence, 0, len(items))
	for _, item := range items {
		out = append(out, CucumberEvidence{
			Path:  strings.TrimSpace(item.Path),
			Lines: item.Lines,
		})
	}
	return out
}
