package runner

import (
	"context"
	"io"
	"time"

	"cogni/internal/agent"
	"cogni/internal/spec"
	"cogni/internal/tools"
	"cogni/internal/vcs"
)

// ProviderFactory builds an agent provider for a given config and model.
type ProviderFactory func(agentConfig spec.AgentConfig, model string) (agent.Provider, error)

// ToolRunnerFactory constructs the tool runner for a repo root.
type ToolRunnerFactory func(root string) (*tools.Runner, error)

// RepoRootResolver resolves the repository root for a run.
type RepoRootResolver func(ctx context.Context, repoRoot string) (string, error)

// RepoMetadataLoader resolves VCS metadata for a repository.
type RepoMetadataLoader func(ctx context.Context, repoRoot string) (vcs.Metadata, error)

// SetupCommandRunner executes repo setup commands.
type SetupCommandRunner interface {
	Run(ctx context.Context, dir string, command string) error
}

// RunDependencies allows injecting factories and clocks for a run.
type RunDependencies struct {
	ProviderFactory    ProviderFactory
	ToolRunnerFactory  ToolRunnerFactory
	RepoRootResolver   RepoRootResolver
	RepoMetadataLoader RepoMetadataLoader
	SetupRunner        SetupCommandRunner
	RunID              func() (string, error)
	Now                func() time.Time
	TokenCounter       agent.TokenCounter
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
