# Go Rate Limiter for LLM Calls — Pluggable Backends (TigerBeetle or In‑Memory)

This spec is written for a junior Go developer. If you implement what’s written here, you should not need further clarification.

We are building a **rate-limiting system** that:

* maximizes throughput (keeps providers busy),
* minimizes latency (fast allow/deny),
* supports **multi-dimensional limits** (RPM, TPM, concurrency, budgets),
* supports **unknown token usage** via *reserve upper bound → reconcile*,
* supports two deployment modes:

  1. **Distributed mode**: many clients → **Rate Limiter Server → TigerBeetle**
  2. **Single-binary mode**: one client binary → **in-memory backend** (no TigerBeetle)

TigerBeetle **must not** be exposed to untrusted callers (it does not support authentication). ([docs.tigerbeetle.com][1])
So in distributed mode, clients always call our Go server.

---

## 1) Key ideas (how limiting works)

### 1.1 Rolling-window limits (RPM/TPM/budgets): “pending reservations that expire”

For each limit, we maintain a “capacity” that gets temporarily reduced when we allow a request:

* **Reserve**: create a reservation that holds some amount for `window_seconds`.
* **Replenish**: when the window passes, the reservation expires and capacity is restored.

TigerBeetle implements this using **pending transfers with timeouts**: pending transfers reserve amounts in `debits_pending`/`credits_pending` and can be voided/posted/expired. ([docs.tigerbeetle.com][2])
Expired pending transfers return the full amount to the original accounts, but removal is best-effort and may lag slightly; don’t write tests assuming exact expiry timing. ([docs.tigerbeetle.com][3])

In the memory backend, we implement the same semantics with in-process data structures.

### 1.2 Concurrency limits: “hold a slot until Complete, with a timeout safety”

For concurrency:

* Reserve “1 slot” at the start of the LLM call.
* On completion, release immediately.
* If the worker crashes, the hold must eventually release (timeout safety).

TigerBeetle: reserve a **pending transfer** and **void** it on completion; if not voided, it expires. ([docs.tigerbeetle.com][2])
Memory backend: store the hold in a map and delete it on completion; also clean up expired holds periodically.

### 1.3 Multi-resource requests: “all-or-nothing”

An LLM request usually requires multiple limits at once (e.g., RPM + TPM + concurrency + tenant daily budget).

We need **atomic** behavior:

* either all reservations succeed → ALLOW
* or any fails → DENY and reserve nothing

TigerBeetle provides **linked transfers**: if any transfer in a linked chain fails, the whole chain fails. ([docs.tigerbeetle.com][4])
Memory backend must implement the same all-or-nothing logic (check all, then apply all).

---

## 2) Important TigerBeetle constraints we must design around

These are non-negotiable and affect ID design and retry behavior.

1. **No authentication**: don’t expose TB directly. ([docs.tigerbeetle.com][1])
2. A TB client has **at most one in-flight request**, so batching/multiple sessions matter for throughput. ([docs.tigerbeetle.com][5])
3. Default maximum batch size (server-configurable) is **8189** events. ([docs.tigerbeetle.com][6])
4. Account and transfer IDs are **u128** and must not be `0` or `2^128-1`. ([docs.tigerbeetle.com][7])
5. `ledger` and `code` must not be zero for normal creates; for post/void transfers, `ledger/code` may be zero and will be filled from the pending transfer. ([docs.tigerbeetle.com][7])
6. `create_transfers` response includes **only failures**; successful results are not present and “ok” is generated client-side. ([docs.tigerbeetle.com][8])
7. **Critical**: if a transfer fails with a transient error (including `exceeds_credits`), retrying the same transfer ID later returns `id_already_failed`. Therefore **a denied reservation attempt must use a new transfer ID** if you want to retry later. ([docs.tigerbeetle.com][8])

This last point is why we introduce **Lease IDs** (attempt IDs) below.

---

## 3) Terminology and data types

### 3.1 LimitKey (string)

A LimitKey identifies “what we are limiting”. It must be stable and globally unique as a string.

We standardize keys as:

* Global provider/model dimensions:

  * `global:llm:<provider>:<model>:rpm`
  * `global:llm:<provider>:<model>:tpm`
  * `global:llm:<provider>:<model>:concurrency`

* Tenant budgets (provider-agnostic):

  * `tenant:<tenant_id>:llm:daily_tokens`

Later, when we add more types of limits (DB, GPUs, etc.), we follow the same pattern:

* `global:<namespace>:...`
* `tenant:<tenant_id>:<namespace>:...`

### 3.2 LimitDefinition

Server-side definition (admin-controlled):

```go
type LimitKind string
const (
  KindRolling     LimitKind = "rolling"     // rolling window: expires after WindowSeconds
  KindConcurrency LimitKind = "concurrency" // held until Complete (or expires after TimeoutSeconds)
)

type LimitDefinition struct {
  Key            string
  Kind           LimitKind
  Capacity       uint64        // max units
  WindowSeconds  int           // only for rolling
  TimeoutSeconds int           // only for concurrency
  Unit           string        // e.g. "requests", "tokens", "inflight" (for humans/logging)
  Description    string
}
```

### 3.3 Lease vs Job (ID design)

We separate “logical job” from “reservation attempt”.

* **JobID**: stable ID for the logical LLM job in your app (ULID/UUID). Optional but recommended for logging.
* **LeaseID**: **unique per reserve attempt** (must be ULID). If a request is denied and you retry later, you generate a **new LeaseID**.

Reason: a TB reservation attempt can fail with `exceeds_credits`; retrying the same transfer ID later is blocked by `id_already_failed`. ([docs.tigerbeetle.com][8])

---

## 4) Public API (same for both backends)

We expose a consistent interface. In distributed mode it’s HTTP+JSON. In single-binary mode it’s a Go interface.

### 4.1 Reserve (attempt to acquire capacity)

#### ReserveRequest

```json
{
  "lease_id": "01J...ULID...",
  "job_id": "01J...ULID...", 
  "requirements": [
    { "key": "global:llm:openai:gpt-4o:rpm",         "amount": 1    },
    { "key": "global:llm:openai:gpt-4o:tpm",         "amount": 1800 },
    { "key": "global:llm:openai:gpt-4o:concurrency", "amount": 1    },
    { "key": "tenant:tenant_a:llm:daily_tokens",     "amount": 1800 }
  ]
}
```

Rules:

* `lease_id` **required** and must be a ULID string.
* `job_id` optional (for logs).
* `requirements` length must be `1..32` (enforced).
* Amounts are `>= 1`.
* The server looks up each `key` in the limit registry and uses its definition (kind/window/timeout).

#### ReserveResponse

Allowed:

```json
{
  "allowed": true,
  "retry_after_ms": 0,
  "reserved_at_unix_ms": 1736660000000
}
```

Denied:

```json
{
  "allowed": false,
  "retry_after_ms": 120,
  "reserved_at_unix_ms": 0
}
```

Notes:

* `reserved_at_unix_ms` is set only when allowed.
* `retry_after_ms` is a hint. Clients should add jitter.

### 4.2 Complete (release concurrency + reconcile tokens)

#### CompleteRequest

```json
{
  "lease_id": "01J...ULID...",
  "job_id": "01J...ULID...",
  "actuals": [
    { "key": "global:llm:openai:gpt-4o:tpm",     "actual_amount": 740 },
    { "key": "tenant:tenant_a:llm:daily_tokens", "actual_amount": 740 }
  ]
}
```

Rules:

* `lease_id` required.
* `actuals` includes only the rolling limits you want to reconcile (typically TPM + budgets).
* Concurrency is always released on Complete.

Response:

```json
{ "ok": true }
```

If Complete is never called (crash), concurrency must still be released by timeout.

### 4.3 Admin API: define/update limits at runtime (no TigerBeetle restart)

Distributed mode must support adding/updating limits without restarting TigerBeetle. Accounts/transfers are created dynamically.

Endpoints (HTTP+JSON):

* `PUT /v1/admin/limits` — create/update a LimitDefinition
* `GET /v1/admin/limits` — list
* `GET /v1/admin/limits/{key}` — get one

#### DefineLimitRequest

```json
{
  "key": "global:llm:openai:gpt-4o:rpm",
  "kind": "rolling",
  "capacity": 3000,
  "window_seconds": 60,
  "timeout_seconds": 0,
  "unit": "requests",
  "description": "OpenAI gpt-4o requests per minute"
}
```

For concurrency:

```json
{
  "key": "global:llm:openai:gpt-4o:concurrency",
  "kind": "concurrency",
  "capacity": 200,
  "window_seconds": 0,
  "timeout_seconds": 300,
  "unit": "inflight",
  "description": "Max in-flight calls"
}
```

Admin behavior:

* Persist definition in a local registry file (see §7).
* Provision backend state:

  * TB: create account (if needed) + set capacity without TB restart
  * Memory: create/update in-memory limiter config

---

## 5) LLM-specific wrapper (client convenience)

We provide a helper so app developers don’t manually assemble keys.

### 5.1 Token upper bound rule (v1)

We do not know exact tokens before the call, so we reserve an upper bound.

Upper bound formula:

* `token_upper_bound = EstimatePromptTokens(prompt) + max_output_tokens`
* v1 prompt estimator: **byte count** of UTF-8 prompt as a conservative proxy:

  * `EstimatePromptTokens(prompt) = uint64(len([]byte(prompt)))`
  * (This is conservative; real token count is usually smaller.)

### 5.2 LLM helper in client lib

```go
type LLMReserveInput struct {
  LeaseID         string // ULID, unique per attempt
  JobID           string // optional
  TenantID        string
  Provider        string
  Model           string
  Prompt          string
  MaxOutputTokens uint64
  WantDailyBudget bool // if true, include tenant daily budget key
}

func BuildLLMRequirements(in LLMReserveInput) []Requirement
```

Requirements produced:

* RPM: `global:llm:<provider>:<model>:rpm` amount=1
* TPM: `global:llm:<provider>:<model>:tpm` amount=token_upper_bound
* Concurrency: `global:llm:<provider>:<model>:concurrency` amount=1
* Optional daily: `tenant:<tenant_id>:llm:daily_tokens` amount=token_upper_bound

---

## 6) Avoiding head-of-line blocking (client-side scheduler)

Head-of-line blocking happens when a client has a FIFO queue and the first job can’t run (e.g., OpenAI TPM exhausted) while later jobs *could* run (e.g., Anthropic).

We fix this with an optional in-process **Scheduler** in the client library.

### 6.1 Scheduler rules

* Maintain a queue per `(provider, model)` (work-class).
* Each queue has two lists:

  * `ready`: runnable now
  * `blocked`: jobs with `not_before` timestamps
* Worker loop:

  1. round-robin across queues that have ready work
  2. attempt Reserve
  3. if allowed → execute LLM call → Complete
  4. if denied → set `not_before = now + retry_after + jitter` and move to blocked
  5. keep going; never let a blocked job stop other queues

This maximizes throughput without centralizing global queueing in the server.

### 6.2 Scheduler API

```go
type Job struct {
  LeaseID string // must be new per attempt; scheduler will regenerate on retry
  JobID   string

  TenantID, Provider, Model string
  Prompt string
  MaxOutputTokens uint64
  WantDailyBudget bool

  Execute func(ctx context.Context) (actualTokens uint64, err error)
}

type Scheduler struct { ... }

func NewScheduler(l Limiter, workers int) *Scheduler
func (s *Scheduler) Submit(job Job)
func (s *Scheduler) Shutdown(ctx context.Context) error
```

**Important:** scheduler must regenerate a new LeaseID for retries after deny (because of TB `id_already_failed` semantics). ([docs.tigerbeetle.com][8])
Implementation: store `JobID` stable; set `LeaseID` only when about to attempt Reserve.

---

## 7) Limit registry (definition + persistence)

We need an internal registry of `LimitDefinition` so we can:

* define/update limits at runtime (TB mode)
* keep memory backend configured
* add new limits later without redeploying TB

### 7.1 Persistence format

Use a single JSON file (atomic rewrite) on server:

* `data/limits.json`

Schema: array of `LimitDefinition`.

Write strategy:

* write to `limits.json.tmp`
* `fsync`
* rename to `limits.json`

Load at startup; keep in-memory map `key -> LimitDefinition`.

### 7.2 Registry behavior

* On `PUT /v1/admin/limits`: validate + upsert in map + persist + call backend `ApplyDefinition(def)`.
* On Reserve: if any requirement key missing, return HTTP 404 with JSON error:

  ```json
  { "allowed": false, "retry_after_ms": 0, "error": "unknown_limit_key: ..." }
  ```

---

## 8) Backend interface (pluggable)

Define a common interface so we can swap TigerBeetle vs memory.

```go
type Backend interface {
  ApplyDefinition(ctx context.Context, def LimitDefinition) error
  Reserve(ctx context.Context, leaseID, jobID string, reqs []Requirement, reservedAt time.Time) (allowed bool, denyReason string, err error)
  Complete(ctx context.Context, leaseID, jobID string, actuals []Actual) error
}
```

Where:

```go
type Requirement struct {
  Key    string
  Amount uint64
}

type Actual struct {
  Key          string
  ActualAmount uint64
}
```

The HTTP server wraps this interface.

The in-process client can call the backend directly without HTTP.

---

## 9) TigerBeetle backend design (distributed mode)

### 9.1 Account model

We implement the **rate limiting recipe** using:

* one operator account
* one resource account per LimitKey

Resource accounts have `debits_must_not_exceed_credits` so reservations can’t exceed capacity. ([docs.tigerbeetle.com][9])

**Ledger design (v1):**

* Use a single ledger for all limiter accounts: `LEDGER_LIMITS = 1`.
* Use `CODE_LIMIT = 1` for reserve transfers.

Ledger and code must not be zero for account and non-post/void transfers. ([docs.tigerbeetle.com][7])

### 9.2 ID scheme (deterministic)

TigerBeetle IDs are u128 and must not be `0` or `2^128-1`. ([docs.tigerbeetle.com][7])

We derive u128 IDs via SHA-256 and taking the first 16 bytes:

* `u128 = sha256(label)[:16]` interpreted as **little-endian**.
* If u128 is `0` or `max`, flip the last bit and recheck.

#### Account IDs

* Operator: `acct:operator`
* Limit resource account: `acct:limit:<LimitKey>`

#### Transfer IDs

All derived from LeaseID (not JobID), so each retry attempt can be unique:

* Reserve transfer: `xfer:reserve:<lease_id>:<LimitKey>`
* Concurrency void: `xfer:void:<lease_id>:<concurrency_key>`
* Reconcile (void old rolling): `xfer:void:<lease_id>:<rolling_key>`
* Re-reserve actual rolling: `xfer:rereserve:<lease_id>:<rolling_key>`

**Reason:** denied attempts must use new IDs to retry later. ([docs.tigerbeetle.com][8])

### 9.3 Provisioning a new limit (ApplyDefinition)

For a new LimitKey:

1. `create_accounts` for:

   * operator (once)
   * resource account for the key with `debits_must_not_exceed_credits`
2. Set capacity by ensuring resource account net balance equals `Capacity`:

   * Lookup current account balances (admin path can read).
   * If net < capacity → posted transfer operator → resource amount=(capacity - net)
   * If net > capacity → posted transfer resource → operator amount=(net - capacity), **but only up to currently unused amount**. If usage is higher than new capacity, apply decrease gradually; store “pending decrease” and retry periodically until reached.

(The rate-limiting recipe funds accounts up front and then uses pending transfers with timeouts. ([docs.tigerbeetle.com][9]))

### 9.4 Reserve flow (write-only hot path)

Reserve must not perform reads (for latency). It only does writes.

Given requirements `reqs[]`, for each `req` we:

* find its definition (rolling or concurrency)
* create a pending transfer resource → operator:

  * amount = req.Amount
  * timeout:

    * rolling: def.WindowSeconds
    * concurrency: def.TimeoutSeconds
  * flags:

    * `pending = true`
    * `linked = true` for all except the last transfer in the chain

Submit all transfers for this lease in a single `create_transfers` call.

Linked transfers succeed/fail as a unit. ([docs.tigerbeetle.com][4])

If any fails due to `exceeds_credits`, deny. (This is how the recipe enforces the rate limit: limited credit balance + pending transfer fails.) ([docs.tigerbeetle.com][9])

### 9.5 Complete flow

On completion we do:

1. **Release concurrency**: submit a void-pending transfer referencing the concurrency reserve transfer. Void restores the pending amount to original accounts. ([docs.tigerbeetle.com][2])

2. **Reconcile rolling limits** if actual < reserved:

   * Void the original pending transfer for that key
   * Create a new pending transfer for `actual_amount` with timeout = remaining window:

     * `remaining = max(1, window - elapsedSecondsSinceReservedAt)`
   * This frees unused slack earlier, increasing throughput.

Notes:

* Once a pending transfer expires, it cannot be voided or posted, and pending balances may persist briefly after expiry. Treat `pending_transfer_expired` as “already released”. ([docs.tigerbeetle.com][3])

### 9.6 Server batching (important for throughput)

TigerBeetle clients can have at most one in-flight request. ([docs.tigerbeetle.com][5])
Default maximum batch size is 8189. ([docs.tigerbeetle.com][6])
So the server must microbatch across many incoming HTTP requests:

* Maintain a `TBSubmitter` goroutine:

  * reads work items: `{transfers []Transfer, done chan result}`
  * concatenates work items into a batch up to `maxEvents` (e.g. 8000)
  * flushes every ~200µs or when full
  * sends to one TB client session from a pool

Important: never split a work item across batches because it may end with `flags.linked=true` which would make the batch invalid (`linked_event_chain_open`). ([docs.tigerbeetle.com][8])

Response handling:

* `create_transfers` returns only failures; “ok” is not in the protocol. ([docs.tigerbeetle.com][8])
  So:
* build a `status[]` sized `len(batch)` default OK
* fill errors for returned indices
* map back to each work item and respond

---

## 10) In-memory backend design (single-binary mode)

The in-memory backend must match the same semantics:

* rolling reservations expire after a window,
* concurrency reservations release on Complete or timeout,
* all-or-nothing reserve across multiple keys.

### 10.1 Data structures

For each rolling limit key:

```go
type rollingLimit struct {
  cap   uint64
  used  uint64
  heap  reservationHeap // min-heap by expiresAt
}
type reservation struct {
  id        string // derived from lease_id+key
  amount    uint64
  expiresAt time.Time
  heapIndex int
  canceled  bool
}
```

For concurrency limit:

```go
type concLimit struct {
  cap   uint64
  holds map[string]time.Time // lease_id -> expiresAt
  heap  concHeap             // min-heap by expiresAt for cleanup
}
```

Global:

* map `key -> LimitDefinition`
* map `key -> rollingLimit/concLimit`
* single mutex `sync.Mutex` (v1) to keep it correct and easy

### 10.2 Reserve algorithm (atomic)

Steps:

1. Lock.
2. For each requirement key:

   * fetch definition; if missing → error
   * cleanup expired reservations for that key (pop heap while expiresAt <= now; decrement used)
3. Check feasibility for all requirements:

   * rolling: `used + amount <= cap`
   * concurrency: `len(holds)+1 <= cap`
4. If any infeasible:

   * compute retry_after = soonest expiry among the failing key’s heap (or small default like 50ms)
   * unlock and return denied
5. If feasible, apply all:

   * push reservations into heaps
   * update used / holds
6. Unlock and allow.

### 10.3 Complete algorithm

1. Lock.
2. Release concurrency hold for this lease_id if present.
3. For each `actual`:

   * find the original rolling reservation amount by scanning map of lease_id+key to reservation (store this map at reserve time)
   * if `actual < reserved`:

     * reduce the reservation amount to actual
     * adjust `used` accordingly
   * if `actual >= reserved`: do nothing
4. Unlock.

---

## 11) Go components to implement

### 11.1 Packages and binaries

Repo layout:

```
/cmd/ratelimiterd/               # HTTP server (distributed mode)
/cmd/prototypes/...              # prototype programs (see §12)

/internal/api/                   # HTTP handlers + JSON types
/internal/registry/              # limit registry (load/save JSON, in-memory map)
/internal/backend/tb/            # TigerBeetle backend
/internal/backend/memory/        # in-memory backend
/internal/tbutil/                # hash-to-u128, ID helpers, TB client pool, submitter
/pkg/ratelimiter/                # public Go interface + types (Limiter, Scheduler)
/pkg/ratelimiter/httpclient/     # remote client implementation
/pkg/ratelimiter/local/          # local in-process implementation (memory backend)
```

### 11.2 `ratelimiterd` (HTTP server)

Endpoints:

* `POST /v1/reserve`
* `POST /v1/complete`
* `PUT /v1/admin/limits`
* `GET /v1/admin/limits`
* `GET /v1/admin/limits/{key}`
* `GET /healthz`

Config file `config.yaml`:

```yaml
server:
  listen_addr: ":8080"
  backend: "tigerbeetle" # or "memory" (dev)

registry:
  path: "./data/limits.json"

tigerbeetle:
  cluster_id: "0"
  addresses: ["127.0.0.1:3000"]
  sessions: 8
  max_batch_events: 8000
  flush_interval_micros: 200
```

### 11.3 Client library usage patterns

#### Distributed mode (remote)

```go
c := httpclient.New("http://ratelimiter:8080")
sched := ratelimiter.NewScheduler(c, 32)
sched.Submit(job)
```

#### Single-binary mode (local)

```go
lim := local.NewMemoryLimiterFromFile("./limits.json")
sched := ratelimiter.NewScheduler(lim, 32)
```

No TigerBeetle, no extra deployment.

---

## 12) Prototype plan (do this first; test risky assumptions)

We will not build the “final” code first. We write prototypes to validate the riskiest assumptions.

### Prototype 1 — TB semantics + ID behavior (highest risk)

Goal: prove linked + pending + id_already_failed behavior works with our LeaseID scheme.

Implement: `/cmd/prototypes/proto_tb_semantics/main.go`

Steps:

1. Create operator + one rolling limit account with capacity=2.
2. Attempt Reserve 3 times with **three different lease_ids** → expect 2 allowed, 1 denied.
3. Retry the denied attempt with the **same lease_id** → it must still be denied (because the transfer ID already failed). ([docs.tigerbeetle.com][8])
4. Retry with a **new lease_id** after waiting a bit → should eventually allow.

Acceptance:

* Confirms we must regenerate LeaseID for retry after denial.

### Prototype 2 — TB microbatcher throughput/latency

Goal: validate our submitter design and pick initial flush interval/session count.

Implement: `/cmd/prototypes/proto_tb_batching/main.go`

* Spin up N goroutines generating reserve work items.
* Microbatch in submitter (same as server).
* Measure p50/p95/p99 latency per reserve and throughput.

Acceptance:

* Stable under load, no deadlocks, no batch splitting errors.

### Prototype 3 — Reconciliation frees capacity early

Goal: ensure void+re-reserve pattern increases throughput.

Implement: `/cmd/prototypes/proto_tb_reconcile/main.go`

Scenario:

* Set TPM capacity to 100
* Reserve with amount 100 (allowed)
* Complete with actual 10 (void + re-reserve 10)
* Immediately Reserve another with amount 90 should allow (or at least allow much sooner than waiting full window)

Acceptance:

* Observed capacity is freed early enough to matter (allow within <1s).

### Prototype 4 — Dynamic limit creation without restarting TB

Goal: prove we can add a new limit key at runtime.

Implement: `/cmd/prototypes/proto_server_dynamic_limits/main.go`

* Start `ratelimiterd` in TB mode
* Call `PUT /v1/admin/limits` with a new key
* Immediately call `POST /v1/reserve` using that key

Acceptance:

* Works without restarting TigerBeetle or the server.

### Prototype 5 — Memory backend correctness

Goal: verify memory backend matches semantics for rolling + concurrency.

Implement: `/cmd/prototypes/proto_memory/main.go`

* Same scenarios as proto 1 and 3, but without TB.
* Confirm scheduler avoids HOL blocking using two providers, one saturated.

Acceptance:

* Correct allow/deny; no data races; decent performance.

---

## 13) Final implementation steps (after prototypes pass)

1. Implement registry + admin endpoints (definitions first).
2. Implement memory backend end-to-end (Reserve/Complete).
3. Implement client library:

   * interface `Limiter`
   * `httpclient` and `local` implementations
   * scheduler (per provider/model queues) with LeaseID regeneration on retry
4. Implement TB backend:

   * ApplyDefinition provisioning
   * Reserve (linked pending transfers)
   * Complete (void concurrency, reconcile rolling)
   * TBSubmitter batching + client pool
5. Integration tests:

   * run TB locally, run `ratelimiterd`, run client tests.

---

## 14) Testing checklist (must implement)

### Unit tests

* LimitKey formatting helper functions.
* Registry JSON load/save (atomic write).
* Memory backend reserve/complete correctness.

### Integration tests (TB)

* Reserve denial + retry with same LeaseID stays denied (`id_already_failed`). ([docs.tigerbeetle.com][8])
* Linked multi-requirement is atomic (either all reserved or none). ([docs.tigerbeetle.com][4])
* Concurrency released on Complete via void. ([docs.tigerbeetle.com][2])

### Load test (optional but recommended)

* Compare p99 latency with batching on/off.

---

## 15) Operational notes

* **Security:** TB has no auth; keep TB on a private network; only `ratelimiterd` talks to it. ([docs.tigerbeetle.com][1])
* **Retry policy:**

  * If Reserve request times out and you don’t know the result: retry with the **same LeaseID** (idempotent).
  * If Reserve returns denied: retry later with a **new LeaseID**.
* **Expiry timing:** TB expiry cleanup is best-effort; don’t test exact expiry times. ([docs.tigerbeetle.com][3])

---

## 16) Definition + replenishment summary (what the system supports)

### Defining limits

* Use admin endpoint `PUT /v1/admin/limits` (distributed) or load JSON file (single-binary).

### Spending limits

* Reserve creates rolling reservations + concurrency hold atomically.
* If any key is at capacity, Reserve denies.

### Replenishing limits

* Rolling: automatic via expiry after window. ([docs.tigerbeetle.com][2])
* Concurrency: released on Complete or timeout safety.
* Capacity changes:

  * Increase: immediate (fund more in TB; update cap in memory).
  * Decrease: takes effect as current usage decays; apply gradually if needed.

---

If you implement exactly the above, we will have:

* a single codebase with two interchangeable backends,
* a distributed deployment that is safe and fast,
* and a single-binary deployment that works with no external dependencies.

[1]: https://docs.tigerbeetle.com/single-page/ "TigerBeetle"
[2]: https://docs.tigerbeetle.com/coding/two-phase-transfers/ "Two-Phase Transfers"
[3]: https://docs.tigerbeetle.com/reference/transfer/ "Transfer"
[4]: https://docs.tigerbeetle.com/coding/linked-events/ "Linked Events"
[5]: https://docs.tigerbeetle.com/coding/requests/ "Requests"
[6]: https://docs.tigerbeetle.com/coding/clients/go/ "tigerbeetle-go"
[7]: https://docs.tigerbeetle.com/reference/account/ "Account"
[8]: https://docs.tigerbeetle.com/reference/requests/create_transfers/ "create_transfers"
[9]: https://docs.tigerbeetle.com/coding/recipes/rate-limiting/ "Rate Limiting"
