package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"cogni/internal/agent"
	"cogni/internal/cucumber"
	"cogni/internal/prompt"
)

// featureInputError signals a failure to read feature inputs.
type featureInputError struct {
	err error
}

// Error returns the wrapped error message.
func (e featureInputError) Error() string {
	return e.err.Error()
}

// featureProviderError signals a failure to initialize the provider.
type featureProviderError struct {
	err error
}

// Error returns the wrapped error message.
func (e featureProviderError) Error() string {
	return e.err.Error()
}

// featureEvaluation captures per-feature evaluation outcomes.
type featureEvaluation struct {
	examples            []CucumberExample
	correctCount        int
	implementedCount    int
	notImplementedCount int
}

// runFeatureBatch executes one feature prompt and parses the agent response.
func runFeatureBatch(
	ctx context.Context,
	repoRoot string,
	task taskRun,
	featurePath string,
	featureExamples []cucumber.Example,
	toolDefs []agent.ToolDefinition,
	executor agent.ToolExecutor,
	providerFactory ProviderFactory,
	tokenCounter agent.TokenCounter,
	verbose bool,
	verboseWriter io.Writer,
	verboseLogWriter io.Writer,
	noColor bool,
) (CucumberFeatureRun, map[string]cucumber.AgentResponse, error, error) {
	expectedIDs := make([]string, 0, len(featureExamples))
	for _, example := range featureExamples {
		expectedIDs = append(expectedIDs, example.ID)
	}

	featureText, err := os.ReadFile(featurePath)
	if err != nil {
		return CucumberFeatureRun{}, nil, nil, featureInputError{err: fmt.Errorf("read feature %s: %w", featurePath, err)}
	}

	promptText, err := prompt.RenderCucumberPrompt(ctx, featurePath, string(featureText), expectedIDs)
	if err != nil {
		return CucumberFeatureRun{}, nil, nil, featureInputError{err: fmt.Errorf("render cucumber prompt: %w", err)}
	}
	provider, err := providerFactory(task.Agent, task.Model)
	if err != nil {
		return CucumberFeatureRun{}, nil, nil, featureProviderError{err: err}
	}
	session := newSession(task, repoRoot, toolDefs, verbose)
	runMetrics, runErr := agent.RunTurn(ctx, session, provider, executor, promptText, agent.RunOptions{
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

	featureRun := CucumberFeatureRun{
		FeaturePath:     featurePath,
		ExamplesTotal:   len(featureExamples),
		TokensTotal:     runMetrics.Tokens,
		WallTimeSeconds: runMetrics.WallTime.Seconds(),
		AgentSteps:      runMetrics.Steps,
		ToolCalls:       runMetrics.ToolCalls,
	}

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

	return featureRun, responseMap, parseErr, runErr
}

// evaluateFeatureExamples builds example results from a feature batch.
func evaluateFeatureExamples(
	featureExamples []cucumber.Example,
	groundTruth map[string]cucumberGroundTruth,
	responseMap map[string]cucumber.AgentResponse,
	parseErr error,
) (featureEvaluation, error) {
	var result featureEvaluation
	var batchValidation cucumber.BatchValidationError
	hasBatchValidation := errors.As(parseErr, &batchValidation)

	for _, example := range featureExamples {
		gt, ok := groundTruth[example.ID]
		if !ok {
			return featureEvaluation{}, fmt.Errorf("missing ground truth for %s", example.ID)
		}
		if gt.Implemented {
			result.implementedCount++
		} else {
			result.notImplementedCount++
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
			result.correctCount++
		}

		result.examples = append(result.examples, CucumberExample{
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

	return result, nil
}
