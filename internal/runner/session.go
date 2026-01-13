package runner

import "cogni/internal/agent"

// newSession constructs a session for a task run.
func newSession(task taskRun, repoRoot string, toolsDefs []agent.ToolDefinition, verbose bool) *agent.Session {
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
		Verbose:                  verbose,
	}
	return &agent.Session{
		Ctx:     ctx,
		History: agent.BuildInitialContext(ctx),
	}
}
