# ADR 0013: Refactor LLM Call Pipeline to Enable Rate Limiting Integration

## Status

- Proposed

## Context

- Current LLM execution is embedded in `internal/agent/runner.go` and `internal/runner/run_task.go`.
- We need a clear integration point for Reserve/Complete without tangling concerns.
- Code quality guidelines require small files, SRP, and functional core/imperative shell separation.

## Decision

- Refactor the LLM call pipeline into a dedicated package with explicit hooks for pre/post call behavior.
- Rate limiting will be added later by implementing a hook; the refactor is a prerequisite.

## Specification

### Package layout (target)

```
/internal/agent/call/
  types.go          # data structures
  hooks.go          # CallHook interface
  runner.go         # orchestration of a single call
  stream.go         # stream handling helpers
  tools.go          # tool execution helpers
```

### Types

```go
// CallInput describes one model invocation.
type CallInput struct {
  Prompt   Prompt
  ToolDefs []ToolDefinition
  Limits   RunLimits
}

// CallResult captures the terminal output and metrics.
type CallResult struct {
  Output        string
  Metrics       RunMetrics
  FailureReason string
}

// CallHook allows injecting behaviors around the model call.
type CallHook interface {
  BeforeCall(ctx context.Context, input CallInput) error
  AfterCall(ctx context.Context, input CallInput, result CallResult) error
}
```

### Runner algorithm (pseudo-code)

```go
func RunCall(ctx context.Context, provider Provider, executor ToolExecutor, input CallInput, hooks []CallHook) (CallResult, error) {
  for _, h := range hooks {
    if err := h.BeforeCall(ctx, input); err != nil {
      return CallResult{}, err
    }
  }

  // existing stream + tool loop, refactored into small helpers
  result, err := runStreamLoop(ctx, provider, executor, input)

  for _, h := range hooks {
    _ = h.AfterCall(ctx, input, result)
  }
  return result, err
}
```

### Integration point

- Rate limiter integration becomes a `CallHook` implementation:
  - `BeforeCall`: Reserve
  - `AfterCall`: Complete

### Refactor goals

- Split files to <200 lines and single responsibility.
- Move state-free logic into pure functions to support testing.
- Preserve current behavior, metrics collection, and error handling.

## Consequences

- Positive: Clean integration point; easier testing and future middleware.
- Negative: Requires refactoring effort before rate limiter integration.

## Alternatives considered

- Direct integration in `run_task.go` (rejected: mixes concerns, violates guidelines).
- Provider wrapper without refactor (rejected: still leaves large monolithic runner).
