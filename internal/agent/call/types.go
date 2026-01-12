package call

import "cogni/internal/agent"

// CallInput describes one model invocation.
type CallInput struct {
	Prompt   agent.Prompt
	ToolDefs []agent.ToolDefinition
	Limits   RunLimits
}

// CallResult captures the terminal output and metrics.
type CallResult struct {
	Output        string
	Metrics       RunMetrics
	FailureReason string
}
