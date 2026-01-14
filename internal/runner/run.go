package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cogni/internal/agent"
	"cogni/internal/ratelimit"
	"cogni/internal/spec"
	"cogni/internal/tools"
)

// Run executes tasks and returns results without writing outputs.
func Run(ctx context.Context, cfg spec.Config, params RunParams) (Results, error) {
	repoRootResolver := params.Deps.RepoRootResolver
	if repoRootResolver == nil {
		repoRootResolver = resolveRepoRoot
	}
	repoRoot, err := repoRootResolver(ctx, params.RepoRoot)
	if err != nil {
		return Results{}, err
	}
	metadataLoader := params.Deps.RepoMetadataLoader
	if metadataLoader == nil {
		metadataLoader = loadRepoMetadata
	}
	repoMeta, err := metadataLoader(ctx, repoRoot)
	if err != nil {
		return Results{}, err
	}

	setupRunner := params.Deps.SetupRunner
	if err := runSetupCommands(ctx, repoRoot, cfg.Repo.SetupCommands, setupRunner); err != nil {
		return Results{}, err
	}

	runID, err := ensureRunID(params.Deps.RunID)
	if err != nil {
		return Results{}, err
	}
	observer := params.Observer
	if observer != nil {
		observer.OnRunStart(runID, repoMeta.Name)
	}
	now := params.Deps.Now
	if now == nil {
		now = time.Now
	}
	startedAt := now()

	taskRuns, err := planTaskRuns(cfg, params.Selectors, params.AgentOverride)
	if err != nil {
		return Results{}, err
	}

	providerFactory := params.Deps.ProviderFactory
	if providerFactory == nil {
		providerFactory = func(agentConfig spec.AgentConfig, model string) (agent.Provider, error) {
			return agent.ProviderFromEnv(agentConfig.Provider, model, nil)
		}
	}
	toolRunnerFactory := params.Deps.ToolRunnerFactory
	if toolRunnerFactory == nil {
		toolRunnerFactory = func(root string) (*tools.Runner, error) {
			return tools.NewRunner(root)
		}
	}
	tokenCounter := params.Deps.TokenCounter
	if tokenCounter == nil {
		tokenCounter = agent.ApproxTokenCount
	}

	limiterFactory := params.Deps.LimiterFactory
	if limiterFactory == nil {
		limiterFactory = ratelimit.BuildLimiter
	}
	limiter, err := limiterFactory(cfg, repoRoot)
	if err != nil {
		return Results{}, err
	}

	toolRunner, err := toolRunnerFactory(repoRoot)
	if err != nil {
		return Results{}, err
	}
	executor := agent.RunnerExecutor{Runner: toolRunner}

	toolDefs := defaultToolDefinitions()
	taskResults := make([]TaskResult, 0, len(taskRuns))
	usedAgents := map[string]spec.AgentConfig{}
	verboseWriter := params.VerboseWriter
	if params.Verbose && verboseWriter == nil {
		verboseWriter = os.Stdout
	}
	verboseLogWriter := params.VerboseLogWriter

	for _, taskRun := range taskRuns {
		usedAgents[taskRun.Agent.ID] = taskRun.Agent
		if observer != nil {
			observer.OnTaskStart(taskRun.Task.ID, taskRun.Task.Type, taskRun.Task.QuestionsFile, taskRun.AgentID, taskRun.Model)
		}
		switch taskRun.Task.Type {
		case "question_eval":
			result := runQuestionTask(ctx, repoRoot, cfg, taskRun, limiter, toolDefs, executor, providerFactory, tokenCounter, params.Verbose, verboseWriter, verboseLogWriter, params.NoColor, observer)
			taskResults = append(taskResults, result)
			if observer != nil {
				observer.OnTaskEnd(taskRun.Task.ID, result.Status, result.FailureReason)
			}
		default:
			return Results{}, fmt.Errorf("unsupported task type %q", taskRun.Task.Type)
		}
	}

	agents := make([]AgentInfo, 0, len(usedAgents))
	for _, agentConfig := range usedAgents {
		agents = append(agents, AgentInfo{
			ID:             agentConfig.ID,
			Type:           agentConfig.Type,
			Provider:       agentConfig.Provider,
			Model:          agentConfig.Model,
			Temperature:    agentConfig.Temperature,
			MaxSteps:       agentConfig.MaxSteps,
			ToolingVersion: "cogni/0.1.0",
		})
	}

	finishedAt := now()
	results := Results{
		RunID:      runID,
		Repo:       RepoMetadata{Name: repoMeta.Name, VCS: repoMeta.VCS, Commit: repoMeta.Commit, Branch: repoMeta.Branch, Dirty: repoMeta.Dirty},
		Agents:     agents,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Tasks:      taskResults,
		Summary:    summarize(taskResults),
	}
	if observer != nil {
		observer.OnRunEnd(results)
	}
	_ = params.OutputDir
	return results, nil
}

// RunAndWrite executes a run and writes outputs to disk.
func RunAndWrite(ctx context.Context, cfg spec.Config, params RunParams) (Results, OutputPaths, error) {
	repoRootResolver := params.Deps.RepoRootResolver
	if repoRootResolver == nil {
		repoRootResolver = resolveRepoRoot
	}
	repoRoot, err := repoRootResolver(ctx, params.RepoRoot)
	if err != nil {
		return Results{}, OutputPaths{}, err
	}
	params.RepoRoot = repoRoot
	results, err := Run(ctx, cfg, params)
	if err != nil {
		return Results{}, OutputPaths{}, err
	}
	outputDir := params.OutputDir
	if strings.TrimSpace(outputDir) == "" {
		outputDir = cfg.Repo.OutputDir
	}
	outputDir = resolveOutputDir(repoRoot, outputDir)
	paths, err := WriteRunOutputs(results, outputDir)
	if err != nil {
		return results, OutputPaths{}, err
	}
	return results, paths, nil
}
