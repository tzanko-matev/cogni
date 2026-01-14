# Configuration Spec (v1)

This feature adds a `rate_limiter` block and a per-task `concurrency` field.

## Schema additions

```go
type Config struct {
  // existing fields...
  RateLimiter RateLimiterConfig `yaml:"rate_limiter"`
}

type RateLimiterConfig struct {
  Mode             string      `yaml:"mode"`               // disabled | remote | embedded
  BaseURL          string      `yaml:"base_url"`           // remote only
  Limits           []ratelimiter.LimitState `yaml:"limits"` // embedded only (same schema as limits.json)
  LimitsPath       string      `yaml:"limits_path"`        // embedded only
  Workers          int         `yaml:"workers"`            // scheduler workers (default 1)
  RequestTimeoutMs int         `yaml:"request_timeout_ms"` // HTTP timeout (default 2000)
  MaxOutputTokens  uint64      `yaml:"max_output_tokens"`  // fallback when task budget is 0
  Batch            BatchConfig `yaml:"batch"`
}

type BatchConfig struct {
  Size    int `yaml:"size"`     // default 128
  FlushMs int `yaml:"flush_ms"` // default 2
}

type TaskConfig struct {
  // existing fields...
  Concurrency int `yaml:"concurrency"` // question_eval only
}
```

## Defaults

- `rate_limiter.mode`: `"disabled"` (preserve current behavior).
- `rate_limiter.workers`: `1`.
- `rate_limiter.request_timeout_ms`: `2000`.
- `rate_limiter.max_output_tokens`: `2048`.
- `rate_limiter.batch.size`: `128`.
- `rate_limiter.batch.flush_ms`: `2`.
- `task.concurrency`: if unset or <= 0, use `rate_limiter.workers`.

## Validation rules

- `rate_limiter.mode` must be one of: `disabled`, `remote`, `embedded`.
- `remote` mode requires `base_url`.
- `embedded` mode requires exactly one of `limits` or `limits_path`.
- `workers >= 1`.
- `batch.size >= 1`, `batch.flush_ms >= 1`.
- `request_timeout_ms >= 1`.
- `task.concurrency >= 1` when provided.
- `task.concurrency` is only valid for `question_eval` tasks; error otherwise.

## Mapping to rate limiter requirements

For each LLM call, the scheduler builds requirements using:

```
max_output_tokens = if task.budget.max_tokens > 0
                      then task.budget.max_tokens
                      else rate_limiter.max_output_tokens
upper_bound = len([]byte(prompt)) + max_output_tokens
```

This follows ADR 0007. The prompt estimate uses raw bytes for a conservative upper bound.

## YAML examples

### Remote mode

```yaml
rate_limiter:
  mode: "remote"
  base_url: "http://localhost:8080"
  workers: 8
  request_timeout_ms: 2000
  max_output_tokens: 2048
  batch:
    size: 128
    flush_ms: 2

tasks:
  - id: question_eval_core
    type: question_eval
    questions_file: "spec/questions/core.yml"
    concurrency: 8
```

### Embedded mode (inline limits)

```yaml
rate_limiter:
  mode: "embedded"
  limits:
    - definition:
        key: "global:llm:openrouter:model:rpm"
        kind: "rolling"
        capacity: 600
        window_seconds: 60
        unit: "requests"
        description: "example rpm"
        overage: "debt"
      status: "active"
      pending_decrease_to: 0
    - definition:
        key: "global:llm:openrouter:model:tpm"
        kind: "rolling"
        capacity: 50000
        window_seconds: 60
        unit: "tokens"
        description: "example tpm"
        overage: "debt"
      status: "active"
      pending_decrease_to: 0
    - definition:
        key: "global:llm:openrouter:model:concurrency"
        kind: "concurrency"
        capacity: 2
        timeout_seconds: 1
        unit: "requests"
        description: "example concurrency"
        overage: "debt"
      status: "active"
      pending_decrease_to: 0
  workers: 4
  batch:
    size: 64
    flush_ms: 2
```

### Embedded mode (single binary, file)

```yaml
rate_limiter:
  mode: "embedded"
  limits_path: ".cogni/limits.json"
  workers: 4
  batch:
    size: 64
    flush_ms: 2
```

### Disabled mode (explicit)

```yaml
rate_limiter:
  mode: "disabled"
```

## CLI behavior

No new CLI flags are required in v1. `cogni run` and `cogni eval` use the config file’s
`rate_limiter` settings and the task-level `concurrency` (or the global default).
