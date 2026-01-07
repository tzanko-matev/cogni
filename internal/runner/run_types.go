package runner

import (
	"io"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/tools"
)

// ProviderFactory builds an agent provider for a given config and model.
type ProviderFactory func(agentConfig spec.AgentConfig, model string) (agent.Provider, error)

// ToolRunnerFactory constructs the tool runner for a repo root.
type ToolRunnerFactory func(root string) (*tools.Runner, error)

// RunDependencies allows injecting factories and clocks for a run.
type RunDependencies struct {
	ProviderFactory   ProviderFactory
	ToolRunnerFactory ToolRunnerFactory
	RunID             func() (string, error)
	Now               func() time.Time
	TokenCounter      agent.TokenCounter
}

// RunParams configures a run invocation.
type RunParams struct {
	RepoRoot      string
	OutputDir     string
	AgentOverride string
	Selectors     []TaskSelector
	Repeat        int
	Verbose       bool
	VerboseWriter io.Writer
	NoColor       bool
	Deps          RunDependencies
}

// taskRun couples a task with its resolved agent and model.
type taskRun struct {
	Task    spec.TaskConfig
	Agent   spec.AgentConfig
	Model   string
	AgentID string
}
