# Rate Limiter Feature Spec (v1)

Audience: junior Go developer. This spec is self-contained. Follow the files in this folder in the order below.

## Read order

1) `overview.md` (this file)
2) `api.md`
3) `backend-tb.md`
4) `backend-memory.md`
5) `client-lib.md`
6) `implementation-plan.md`
7) `testing.feature`

## Goals

- Enforce multi-dimensional limits for LLM calls (RPM/TPM/concurrency/tenant budget).
- Handle unknown token usage via reserve upper-bound and reconcile after the call.
- Support two deployment modes:
  - Distributed: `ratelimiterd` server + TigerBeetle.
  - Single-binary: in-memory backend in-process.
- Maximize throughput and avoid head-of-line blocking.

## Non-goals (v1)

- Authentication or authorization.
- Persistent reservation metadata across server restarts.
- Provider-specific tokenization (use a simple estimator).
- Capacity decreases for existing limits (deferred; see below).

## Decisions (source of truth)

See ADRs:

- `spec/architecture/decisions/0001-pluggable-rate-limiter-backends.md`
- `spec/architecture/decisions/0002-limit-registry-admin-api.md`
- `spec/architecture/decisions/0003-rate-limit-semantics-pending-linked.md`
- `spec/architecture/decisions/0004-lease-id-and-tb-id-scheme.md`
- `spec/architecture/decisions/0005-tb-microbatching.md`
- `spec/architecture/decisions/0006-client-scheduler-hol.md`
- `spec/architecture/decisions/0007-llm-token-upper-bound-estimation.md`
- `spec/architecture/decisions/0008-client-side-batching.md`
- `spec/architecture/decisions/0009-overage-debt-tracking.md`
- `spec/architecture/decisions/0010-retry-after-heuristics.md`
- `spec/architecture/decisions/0011-no-auth-v1.md`
- `spec/architecture/decisions/0012-error-model-idempotency.md`
- `spec/architecture/decisions/0013-refactor-llm-call-pipeline.md`

## Glossary

- **LimitKey**: string identifying a limit (e.g., `global:llm:openai:gpt-4o:tpm`).
- **LimitDefinition**: admin-defined capacity and semantics for a LimitKey.
- **Rolling limit**: capacity restored after a window (RPM/TPM, daily budget).
- **Concurrency limit**: count of in-flight calls; released on Complete or timeout.
- **LeaseID**: unique ID per reserve attempt; must change after denial.
- **JobID**: optional stable ID for a logical job.
- **Reservation**: pending transfer that temporarily reduces capacity.
- **Overage**: actual usage exceeds reserved amount.
- **Debt tracking**: record overage in a debt account when additional reservation fails.

## High-level architecture

```
Distributed mode:

  Client (Scheduler + Batcher)
            |
            v
      HTTP Limiter Client
            |
            v
       ratelimiterd
      /      |      \
  Registry  Backend  Submitter
              |
              v
         TigerBeetle

Single-binary mode:

  Client (Scheduler + Batcher)
            |
            v
       Memory Backend
```

## Data model summary

- Limit keys follow a stable string convention:
  - Global provider/model:
    - `global:llm:<provider>:<model>:rpm`
    - `global:llm:<provider>:<model>:tpm`
    - `global:llm:<provider>:<model>:concurrency`
  - Tenant budgets:
    - `tenant:<tenant_id>:llm:daily_tokens`
- IDs:
  - `JobID`: ULID or UUID (optional).
  - `LeaseID`: ULID, unique per attempt.

## Core workflows

### Reserve (allowed)

1) Client builds requirements (RPM/TPM/concurrency/budget).
2) Client sends `Reserve` (or batch Reserve).
3) Server validates keys and writes linked pending transfers.
4) Server returns `allowed=true` and `reserved_at_unix_ms`.
5) Client executes LLM call.

### Reserve (denied)

1) Server fails linked transfer chain (e.g., `exceeds_credits`).
2) Server returns `allowed=false` and `retry_after_ms`.
3) Client schedules retry with a new LeaseID.

### Reserve (unknown outcome)

1) Client times out or gets transport error.
2) Client retries with the SAME LeaseID until it receives allow/deny or the caller aborts.

### Complete

1) Client sends `Complete` with actual usage (if known).
2) Server releases concurrency holds.
3) If actual < reserved, server reconciles by voiding/re-reserving (rolling limits).
4) If actual > reserved, server attempts to reserve the diff; on failure it records debt.
5) If reservation metadata is missing (server restart), reconciliation is skipped and reservations expire naturally.

## Pessimistic accounting (important)

When in doubt, overestimate usage to avoid letting requests exceed limits. This is why:

- Unknown Reserve outcomes are retried with the same LeaseID (idempotent).
- Failed Complete calls do not free capacity; pending reservations expire naturally.
- Missing metadata skips reconciliation; this overestimates usage.

## Security (v1)

- No authn/authz; ratelimiterd must be on a trusted network only.

## Deferred items (explicitly out of scope for v1)

- Actual token usage extraction from streaming providers (if not available, omit actuals).
- Capacity decreases when the new cap is below current usage.
- Authn/authz and tenant scoping enforcement.

Next: `api.md`
