# ADR 0007: Token Upper-Bound Estimation for LLM Reservations

## Status

- Proposed

## Context

- Exact token usage is unknown before an LLM call.
- Rate limiting needs a conservative estimate to reserve capacity.
- A lightweight, provider-agnostic estimator is required for v1.

## Decision

- Use a conservative upper bound formula in the LLM helper:
  - `token_upper_bound = len([]byte(prompt)) + max_output_tokens`
- Apply this estimate to token-based limits (TPM and tenant daily token budget).

## Specification

### Types

```go
type LLMReserveInput struct {
  LeaseID         string
  JobID           string
  TenantID        string
  Provider        string
  Model           string
  Prompt          string
  MaxOutputTokens uint64
  WantDailyBudget bool
}

func BuildLLMRequirements(in LLMReserveInput) []Requirement
```

### Algorithm (pseudo-code)

```go
func EstimatePromptTokens(prompt string) uint64 {
  return uint64(len([]byte(prompt)))
}

func BuildLLMRequirements(in LLMReserveInput) []Requirement {
  upper := EstimatePromptTokens(in.Prompt) + in.MaxOutputTokens
  reqs := []Requirement{
    {Key: fmt.Sprintf("global:llm:%s:%s:rpm", in.Provider, in.Model), Amount: 1},
    {Key: fmt.Sprintf("global:llm:%s:%s:tpm", in.Provider, in.Model), Amount: upper},
    {Key: fmt.Sprintf("global:llm:%s:%s:concurrency", in.Provider, in.Model), Amount: 1},
  }
  if in.WantDailyBudget {
    reqs = append(reqs, Requirement{
      Key: fmt.Sprintf("tenant:%s:llm:daily_tokens", in.TenantID),
      Amount: upper,
    })
  }
  return reqs
}
```

### API usage

- `Reserve` is called with `Requirements` built from the helper.
- `Complete` provides actual token usage to reconcile with the reservation.

## Consequences

- Positive: Simple, deterministic, provider-agnostic upper bound.
- Negative: Overestimation can reduce throughput; depends on reconciliation to free unused capacity.

## Alternatives considered

- Provider-specific tokenizers (rejected for v1 complexity).
- No reservation until completion (rejected: cannot prevent quota overruns).
