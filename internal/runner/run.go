package runner

import (
	"context"
	"os"
	"strings"
	"time"

	"cogni/internal/agent"
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

	if err := runSetupCommands(ctx, repoRoot, cfg.Repo.SetupCommands); err != nil {
		return Results{}, err
	}

	runID, err := ensureRunID(params.Deps.RunID)
	if err != nil {
		return Results{}, err
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
	adapterByID := make(map[string]spec.AdapterConfig, len(cfg.Adapters))
	for _, adapter := range cfg.Adapters {
		adapterByID[adapter.ID] = adapter
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

	for _, taskRun := range taskRuns {
		usedAgents[taskRun.Agent.ID] = taskRun.Agent
		if taskRun.Task.Type == "cucumber_eval" {
			taskResults = append(taskResults, runCucumberTask(ctx, repoRoot, taskRun, adapterByID, toolDefs, executor, providerFactory, tokenCounter, params.Verbose, verboseWriter, params.NoColor))
		} else {
			repeat := params.Repeat
			if repeat <= 0 {
				repeat = 1
			}
			taskResults = append(taskResults, runTask(ctx, repoRoot, taskRun, toolDefs, executor, providerFactory, tokenCounter, repeat, params.Verbose, verboseWriter, params.NoColor))
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
