package runner

import (
	"context"
	"io"
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
	correctCount := 0
	implementedCount := 0
	notImplementedCount := 0

	for _, example := range examples {
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

		prompt := renderCucumberPrompt(task.Task.PromptTemplate, example)
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

		output, ok := latestAssistantMessage(session.History)
		if !ok {
			output = ""
		}

		agentResult := &CucumberAgentResult{}
		correct := false
		if runErr == nil {
			response, err := cucumber.ParseAgentResponse(output)
			if err != nil {
				agentResult.ParseError = err.Error()
			} else {
				agentResult.ExampleID = response.ExampleID
				agentResult.Implemented = response.Implemented
				agentResult.Notes = response.Notes
				agentResult.Evidence = convertEvidence(response.Evidence)
				correct = response.Implemented == gt.Implemented
			}
		} else {
			agentResult.ParseError = runErr.Error()
		}

		if correct {
			correctCount++
		}

		exampleResults = append(exampleResults, CucumberExample{
			ExampleID:       example.ID,
			FeaturePath:     example.FeaturePath,
			ScenarioName:    example.ScenarioName,
			ScenarioLine:    example.ScenarioLine,
			ExampleLine:     example.ExampleLine,
			GroundTruth:     truthLabel(gt.Implemented),
			Agent:           agentResult,
			Correct:         correct,
			TokensTotal:     runMetrics.Tokens,
			WallTimeSeconds: runMetrics.WallTime.Seconds(),
			ToolCalls:       runMetrics.ToolCalls,
		})
	}

	total := len(exampleResults)
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correctCount) / float64(total)
	}
	result.Cucumber = &CucumberEval{
		AdapterID:   task.Task.Adapter,
		AdapterType: adapter.Type,
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

	if correctCount == total {
		result.Status = "pass"
		return result
	}
	reason := "incorrect_examples"
	result.Status = "fail"
	result.FailureReason = &reason
	return result
}

func renderCucumberPrompt(template string, example cucumber.Example) string {
	replacer := strings.NewReplacer(
		"{example_id}", example.ID,
		"{feature_path}", example.FeaturePath,
		"{scenario_name}", example.ScenarioName,
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
