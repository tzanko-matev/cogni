# Cogni Rate Limiter Integration + Concurrency Spec (v1)

Audience: junior Go developer. This spec is self-contained. Follow the files in this folder in the order below.

## Read order

1) `overview.md` (this file)
2) `config.md`
3) `integration.md`
4) `concurrency.md`
5) `test-suite.md`
6) `implementation-plan.md`
7) `testing.feature`

## Goals

- Add rate limiting to **all** LLM calls in Cogni.
- Support two run modes:
  - **Remote mode:** Cogni connects to `ratelimiterd` over HTTP.
  - **Embedded mode:** Cogni embeds the in-memory limiter and loads limits from a local file.
- Enable concurrent execution of `question_eval` tasks to reduce wall time.
- Preserve deterministic outputs and stable summaries.
- Keep defaults backward-compatible (no rate limiting and sequential execution unless configured).

## Non-goals (v1)

- Running TigerBeetle inside Cogni.
- Managing `ratelimiterd` admin state (no automatic PUT/GET of limits).
- Authentication or authorization.
- Parallelizing `qa` tasks or `repeat` attempts beyond existing semantics.
- Changing rate limiter semantics or API.

## Decisions (source of truth)

- ADR 0006: client-side scheduler for head-of-line blocking avoidance.
- ADR 0007: token upper-bound estimation for reservations.
- ADR 0013: call pipeline hooks (used indirectly; scheduler owns Reserve/Complete).

## Glossary

- **Limiter mode**: `disabled`, `remote`, or `embedded`.
- **Embedded limiter**: in-process memory backend, loaded from a limits JSON file.
- **Scheduler**: `pkg/ratelimiter.Scheduler`, handles Reserve/Complete retries and concurrency.

## High-level architecture

```
Remote mode:

  Cogni runner
      |
      v
  Scheduler (client-side)
      |
      v
  HTTP client -> ratelimiterd -> backend (TB or memory)
```

```
Embedded mode:

  Cogni runner
      |
      v
  Scheduler (client-side)
      |
      v
  Local limiter -> memory backend
```

## Core behavior

- Build one limiter per Cogni run and reuse it across tasks.
- For every LLM call:
  - Reserve capacity via the scheduler.
  - Execute the call.
  - Complete with actual token usage (best-effort).
- `question_eval` tasks run concurrently based on configured workers.
- Results are ordered deterministically by original question order.
- Verbose output is safe for concurrent writes (no interleaved lines).

## Error handling

- Transport errors from the limiter are retried with a short delay.
- Reservation denials use `retry_after_ms` and regenerate LeaseID.
- Cancellation or budget exceedance marks the question/task as `runtime_error` or `budget_exceeded` (same as today).

## Acceptance criteria

- Embedded mode can run as a single binary with in-memory limits.
- Remote mode uses `ratelimiterd` over HTTP.
- `question_eval` tasks can run concurrently and complete faster than sequential runs.
- Default config preserves current behavior (sequential, no rate limiting).
