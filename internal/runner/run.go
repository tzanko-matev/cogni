package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cogni/internal/agent"
	"cogni/internal/config"
	"cogni/internal/eval"
	"cogni/internal/spec"
	"cogni/internal/tools"
	"cogni/internal/vcs"
)

type ProviderFactory func(agentConfig spec.AgentConfig, model string) (agent.Provider, error)
type ToolRunnerFactory func(root string) (*tools.Runner, error)

type RunDependencies struct {
	ProviderFactory   ProviderFactory
	ToolRunnerFactory ToolRunnerFactory
	RunID             func() (string, error)
	Now               func() time.Time
	TokenCounter      agent.TokenCounter
}

type RunParams struct {
	RepoRoot      string
	OutputDir     string
	AgentOverride string
	Selectors     []TaskSelector
	Repeat        int
	Deps          RunDependencies
}

func Run(ctx context.Context, cfg spec.Config, params RunParams) (Results, error) {
	repoRoot, err := resolveRepoRoot(ctx, params.RepoRoot)
	if err != nil {
		return Results{}, err
	}
	repoMeta, err := loadRepoMetadata(ctx, repoRoot)
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

	for _, taskRun := range taskRuns {
		usedAgents[taskRun.Agent.ID] = taskRun.Agent
		repeat := params.Repeat
		if repeat <= 0 {
			repeat = 1
		}
		taskResults = append(taskResults, runTask(ctx, repoRoot, taskRun, toolDefs, executor, providerFactory, tokenCounter, repeat))
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

func RunAndWrite(ctx context.Context, cfg spec.Config, params RunParams) (Results, OutputPaths, error) {
	repoRoot, err := resolveRepoRoot(ctx, params.RepoRoot)
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

type taskRun struct {
	Task    spec.TaskConfig
	Agent   spec.AgentConfig
	Model   string
	AgentID string
}

func planTaskRuns(cfg spec.Config, selectors []TaskSelector, agentOverride string) ([]taskRun, error) {
	if err := ValidateSelectors(cfg, selectors); err != nil {
		return nil, err
	}
	agentByID := make(map[string]spec.AgentConfig, len(cfg.Agents))
	for _, agentConfig := range cfg.Agents {
		agentByID[agentConfig.ID] = agentConfig
	}

	selectedAgent := map[string]string{}
	if len(selectors) > 0 {
		for _, selector := range selectors {
			if selector.AgentID != "" {
				selectedAgent[selector.TaskID] = selector.AgentID
			}
		}
	}

	selectedIDs := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		selectedIDs = append(selectedIDs, selector.TaskID)
	}
	orderedTasks, err := config.OrderedTasks(cfg, selectedIDs)
	if err != nil {
		return nil, err
	}

	runs := make([]taskRun, 0, len(orderedTasks))
	for _, task := range orderedTasks {
		agentID := agentOverride
		if agentID == "" {
			if selectorAgent, ok := selectedAgent[task.ID]; ok {
				agentID = selectorAgent
			} else if task.Agent != "" {
				agentID = task.Agent
			} else {
				agentID = cfg.DefaultAgent
			}
		}
		agentConfig, ok := agentByID[agentID]
		if !ok {
			return nil, fmt.Errorf("unknown agent id %q", agentID)
		}
		model := task.Model
		if strings.TrimSpace(model) == "" {
			model = agentConfig.Model
		}
		runs = append(runs, taskRun{
			Task:    task,
			Agent:   agentConfig,
			Model:   model,
			AgentID: agentID,
		})
	}
	return runs, nil
}

func runTask(
	ctx context.Context,
	repoRoot string,
	task taskRun,
	toolsDefs []agent.ToolDefinition,
	executor agent.ToolExecutor,
	providerFactory ProviderFactory,
	tokenCounter agent.TokenCounter,
	repeat int,
) TaskResult {
	result := TaskResult{TaskID: task.Task.ID, Type: task.Task.Type}
	attempts := make([]AttemptResult, 0, repeat)
	var failureReason *string

	for attemptIndex := 1; attemptIndex <= repeat; attemptIndex++ {
		provider, err := providerFactory(task.Agent, task.Model)
		if err != nil {
			reason := "runtime_error"
			result.Status = "error"
			result.FailureReason = &reason
			return result
		}
		session := newSession(task, repoRoot, toolsDefs)
		runMetrics, runErr := agent.RunTurn(ctx, session, provider, executor, task.Task.Prompt, agent.RunOptions{
			TokenCounter:    tokenCounter,
			CompactionLimit: task.Task.Budget.MaxTokens,
			Limits: agent.RunLimits{
				MaxSteps:   limitOrDefault(task.Task.Budget.MaxSteps, task.Agent.MaxSteps),
				MaxSeconds: time.Duration(task.Task.Budget.MaxSeconds) * time.Second,
				MaxTokens:  task.Task.Budget.MaxTokens,
			},
		})

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
				SchemaValid:   evalResult.SchemaValid,
				CitationValid: evalResult.CitationValid,
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
	result.Status = "fail"
	result.FailureReason = failureReason
	return result
}

func newSession(task taskRun, repoRoot string, toolsDefs []agent.ToolDefinition) *agent.Session {
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
	}
	return &agent.Session{
		Ctx:     ctx,
		History: agent.BuildInitialContext(ctx),
	}
}

func latestAssistantMessage(history []agent.HistoryItem) (string, bool) {
	for i := len(history) - 1; i >= 0; i-- {
		item := history[i]
		if item.Role != "assistant" {
			continue
		}
		text, ok := item.Content.(string)
		if ok {
			return text, true
		}
	}
	return "", false
}

func summarize(tasks []TaskResult) RunSummary {
	summary := RunSummary{
		TasksTotal: len(tasks),
	}
	for _, task := range tasks {
		switch task.Status {
		case "pass":
			summary.TasksPassed++
		case "fail":
			summary.TasksFailed++
		}
		for _, attempt := range task.Attempts {
			summary.TokensTotal += attempt.TokensTotal
		}
	}
	if summary.TasksTotal > 0 {
		summary.PassRate = float64(summary.TasksPassed) / float64(summary.TasksTotal)
	}
	return summary
}

func limitOrDefault(limit, fallback int) int {
	if limit > 0 {
		return limit
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func resolveRepoRoot(ctx context.Context, repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		repoRoot = wd
	}
	return vcs.DiscoverRepoRoot(ctx, repoRoot)
}

func resolveOutputDir(repoRoot, outputDir string) string {
	if outputDir == "" || filepath.IsAbs(outputDir) {
		return outputDir
	}
	return filepath.Join(repoRoot, outputDir)
}

func loadRepoMetadata(ctx context.Context, repoRoot string) (vcs.Metadata, error) {
	repo, err := vcs.Discover(ctx, repoRoot)
	if err != nil {
		return vcs.Metadata{}, err
	}
	return repo.Metadata(ctx)
}

func ensureRunID(generator func() (string, error)) (string, error) {
	if generator != nil {
		return generator()
	}
	return NewRunID()
}

func runSetupCommands(ctx context.Context, root string, commands []string) error {
	for _, command := range commands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("setup command failed: %w", err)
		}
	}
	return nil
}

func defaultToolDefinitions() []agent.ToolDefinition {
	return []agent.ToolDefinition{
		{
			Name:        "list_files",
			Description: "List files in the repository",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"glob": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "search",
			Description: "Search for a query string in files",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"paths": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "read_file",
			Description: "Read a file from the repository",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":       map[string]any{"type": "string"},
					"start_line": map[string]any{"type": "integer"},
					"end_line":   map[string]any{"type": "integer"},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
	}
}
