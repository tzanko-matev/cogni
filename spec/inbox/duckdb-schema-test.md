# Memo: Comprehensive test suite for Cogni DuckDB measurement schema

**Date:** 2026-01-14
**Scope:** This memo defines an *extensive, correctness-first* test plan for the current DuckDB schema:

* `repos`
* `revisions`
* `revision_parents` (optional but supported)
* `metric_defs`
* `runs`
* `agents` (schema-unknown JSON spec + fingerprint key)
* `questions` (schema-unknown JSON spec + fingerprint key)
* `contexts` (ties repo+rev+agent+question+extra dims)
* `measurements` (append-only fact table with typed value columns)
* `derived_metric_defs` (stores derived metric SQL definitions)
* `v_points` view (plot-ready join)

**Key assumptions:**

* All git/jj graph computations (transitive reduction edges, per-day connected components, etc.) are computed **client-side** and are out of scope for DB tests (except for data availability and join correctness).
* We are **not** adding an `agent_calls` table right now.

---

## 1) Goals and non-goals

### Primary goals

1. **Data correctness & integrity**: PK/unique constraints behave as expected; no silent corruption; no orphan references.
2. **Type correctness**: JSON/MAP and value columns behave consistently; metric physical types are respected.
3. **Idempotent ingestion**: “upsert by fingerprint” works for `agents`, `questions`, `contexts`.
4. **Stable query contracts**: `v_points` returns correct rows/columns/types and is safe for plotting.
5. **Derived metric reliability**: stored `sql_select` definitions compile, produce correct shapes, and remain deterministic (where required).
6. **Operational robustness**: transactions, concurrency patterns, and crash-safety behave as expected in DuckDB.

### Non-goals

* Verifying correctness of client-side graph algorithms (components, ancestry, etc.) beyond “DB contains required inputs”.

---

## 2) Testing approach and tiers

Because “cost is not an issue”, the suite can include **many layers**:

### Tier A: “Schema & invariants” (always-on)

* Fast, deterministic, runs on every PR/commit.
* Executes schema creation and a battery of invariant checks on small + medium datasets.

### Tier B: “Property-based & fuzz” (always-on + long-running mode)

* Randomized generators for agent specs, question specs, contexts, and measurement distributions.
* Repeats thousands of seeds; maintains a regression corpus.

### Tier C: “Stress/performance/soak” (continuous / nightly)

* Very large synthetic datasets (millions of contexts; tens/hundreds of millions of measurements).
* Measures query latency for the “top 20” canonical report queries.
* Concurrency tests; kill/restart durability tests.

### Tier D: “Compatibility matrix”

* Run Tier A/B/C across multiple DuckDB versions (e.g., pinned stable/LTS + latest stable).
* Optional: verify read-only compatibility in DuckDB-WASM if the report viewer uses it.

---

## 3) Test harness recommendations

### Language/tooling

* Use whichever language Cogni’s ingestion/report code is written in (Rust/Go/Python).
* Additionally, maintain a **pure-SQL test pack** using DuckDB’s **sqllogictest** framework for transactional + multi-connection assertions (DuckDB provides constructs for multiple labeled connections and concurrency). ([DuckDB][1])

### Determinism

* All randomized tests must:

  * log the RNG seed,
  * save failing datasets as a minimal “repro fixture” (Parquet/JSON) plus schema version.

### DB modes

Run every applicable test in:

* **in-memory** (`:memory:`) for speed,
* **on-disk** (temporary `.duckdb` files) for durability, WAL/commit behavior, and concurrency model coverage.

---

## 4) Schema creation and DDL correctness tests

### 4.1 “Schema can be created from scratch”

**Test:** execute the full DDL on a new database; verify no warnings/errors.

**Assertions:**

* All expected tables exist.
* Columns exist with expected types and nullability.
* All primary keys and unique constraints exist and are enforced.

### 4.2 “Schema objects are what we think they are”

For each table/view:

* `PRAGMA table_info('…')` matches expected columns.
* `DESCRIBE SELECT * FROM v_points` matches contract shape.

---

## 5) Constraint behavior test suite

DuckDB enforces **primary key** and **unique** constraints (duplicate inserts should fail). ([DuckDB][2])

### 5.1 Primary key enforcement tests (per table)

For each PK table:

* Insert a valid row
* Attempt to insert the same PK again
* Expect a constraint error

Tables:

* `repos(repo_id)`
* `revisions(repo_id, rev_id)`
* `revision_parents(repo_id, child_rev_id, parent_rev_id)`
* `metric_defs(metric_id)`
* `runs(run_id)`
* `agents(agent_id)` plus unique `agent_key`
* `questions(question_id)` plus unique `question_key`
* `contexts(context_id)` plus unique `context_key`
* `measurements(run_id, context_id, metric_id, sample_index)`
* `derived_metric_defs(derived_metric_id)` plus unique `name`

### 5.2 Unique constraint tests

* `agents.agent_key` is unique
* `questions.question_key` is unique
* `contexts.context_key` is unique
* `metric_defs.name` is unique
* `derived_metric_defs.name` is unique

**Edge-case tests:**

* Null handling: PK implies NOT NULL on key columns (DuckDB notes PK also enforces not-null). ([DuckDB][2])
* “Eager constraint evaluation” corner cases (insert/update sequences that may trigger constraint checks earlier than expected); keep regression tests for any issues encountered. ([DuckDB][2])

### 5.3 Foreign keys (future-proofing)

We are *not* relying on FK constraints today, but if we ever add them:

* Add tests covering FK enforcement and DuckDB’s limitations:

  * `ON DELETE CASCADE` is not supported
  * inserting into tables with **self-referencing foreign keys** is not supported ([DuckDB][3])

---

## 6) Type-system and value-column correctness

The schema intentionally stores measurements in typed columns. This must be tested rigorously because the DB won’t automatically prevent “wrong column filled” unless we add CHECK constraints.

### 6.1 JSON columns (`agents.spec`, `questions.spec`, `runs.config`, `runs.environment`, `contexts.scope`, `measurements.value_json`, `measurements.raw`)

**Tests:**

* Insert valid JSON strings into JSON-typed columns.
* Attempt to insert invalid JSON; verify it errors.
* JSON extraction behavior used by reports:

  * verify `->` and `->>` extraction works (e.g., `spec->>'$.model'`). ([DuckDB][4])
* Query stability: ensure JSON extraction behaves the same across versions in the compatibility matrix.

DuckDB supports JSON with the `json` extension and provides arrow operators (`->`, `->>`) and `json_extract`. ([DuckDB][4])

### 6.2 MAP dims (`contexts.dims`)

MAP semantics to test:

* MAPs don’t need the same keys per row (good for unknown/varying schemas). ([DuckDB][5])
* No duplicate keys allowed; missing keys return NULL. ([DuckDB][5])

**Tests:**

* Insert contexts with:

  * `dims = NULL`
  * `dims = {}` (empty)
  * `dims` with varying keys across rows
* Retrieval tests:

  * bracket extraction `dims['benchmark']`
  * missing key returns NULL

### 6.3 Timestamp tests (`revisions.ts_utc`, `runs.collected_at`, `measurements.observed_at`)

* Verify that ordering by `revisions.ts_utc` yields stable chart ordering.
* Verify that timestamps round-trip without losing precision.
* Verify that derived-day binning in client logic is consistent with UTC timestamp.

### 6.4 Measurement type invariants (critical)

For every measurement row, enforce (via tests) the invariant:

> Exactly one of `{value_double, value_bigint, value_bool, value_varchar, value_json, value_blob}` is non-null, and it matches `metric_defs.physical_type`.

**Test cases:**

* “Happy path” for each `physical_type`.
* “Wrong column filled” should be detected by an invariant query (even if DB allows it).
* “Multiple value columns set” should be detected.
* “No value columns set” (status ok) should be detected.

**Invariant query example (used in tests):**

* count rows where `status='ok'` but type mismatch occurs
* must be zero

---

## 7) Ingestion correctness and idempotency

These tests validate the *application-level contract* of the schema.

### 7.1 Fingerprint/key canonicalization tests (agents/questions/contexts)

Because `agent_key`, `question_key`, `context_key` are computed client-side:

* **Canonicalization correctness tests**

  * Different JSON key orders produce the same key
  * Whitespace differences produce the same key
  * Equivalent MAPs/dims orderings produce the same key

* **Collision resistance tests**

  * Generate large random corpora of specs and ensure no collisions in practice (statistical test; log any collision event and fail hard)

### 7.2 Upsert behavior tests

Even if the app uses `INSERT … ON CONFLICT`, the test intent is:

* Inserting the same agent/question/context multiple times results in:

  * one row in the corresponding table
  * stable `*_id` returned/used thereafter

### 7.3 Partial ingestion tests

Simulate failures midway through ingestion and validate:

* Transaction rollback leaves DB unchanged (no partial writes).
* A retry results in consistent final state.

DuckDB transactions provide isolation and rollback semantics (changes aren’t visible until commit; rollback discards changes). ([DuckDB][6])

---

## 8) Referential integrity tests (since we’re not using FKs)

Even without FK constraints, we can make integrity “real” through test-enforced invariants.

Create a battery of “orphan checks”:

1. **Runs reference existing repos**

   * `runs.repo_id` must exist in `repos`.

2. **Revisions belong to existing repos**

   * `revisions.repo_id` exists in `repos`.

3. **Revision parents reference existing revisions**

   * `(repo_id, child_rev_id)` exists in `revisions`
   * `(repo_id, parent_rev_id)` exists in `revisions`

4. **Contexts reference existing revisions**

   * `(contexts.repo_id, contexts.rev_id)` exists in `revisions`

5. **Contexts reference valid agents/questions when non-null**

   * `contexts.agent_id` exists in `agents` when not null
   * `contexts.question_id` exists in `questions` when not null

6. **Measurements reference existing runs/contexts/metrics**

   * `measurements.run_id` exists in `runs`
   * `measurements.context_id` exists in `contexts`
   * `measurements.metric_id` exists in `metric_defs`

**All orphans must be 0.**
Add tests that create intentional orphans to ensure the check queries fail as expected.

---

## 9) `v_points` view contract tests

`v_points` is the main “plot-ready” API. Treat it like a public interface.

### 9.1 Shape contract

* Columns exist: `repo_id, rev_id, ts, run_id, metric, agent_id, question_id, dims, scope, sample_index, value, status, error_message`
* Types are stable:

  * `value` is numeric (DOUBLE) for DOUBLE/BIGINT metrics (BIGINT cast to DOUBLE)
  * `value` is NULL for non-numeric physical types

### 9.2 Row semantics tests

* Join correctness: each measurement row that has valid refs appears exactly once.
* `ts` comes from `revisions.ts_utc`.
* Filtering semantics:

  * `WHERE metric='tokens' AND status='ok'` returns expected rows.
  * Distinguish between multiple samples via `sample_index`.

### 9.3 “Null & error” tests

* A measurement with `status='error'` should:

  * appear in `v_points` if we intentionally include all statuses
  * retain `error_message`
* `status='ok'` should imply `error_message IS NULL` (enforce as invariant).

---

## 10) Derived metrics test suite (`derived_metric_defs`)

We store derived metric definitions as SQL text. Tests must ensure they remain safe and useful.

### 10.1 Parse/execute tests

For each row in `derived_metric_defs`:

* Prepare and execute the `sql_select` in isolation.
* Validate it runs successfully against a fixture DB that includes the required source metrics.

### 10.2 Output contract tests

Each derived metric query must return a standardized shape (define this contract explicitly in the tests). For example:

* Required columns: `(repo_id, rev_id, ts, run_id, context_id, value)`
* `value` must be numeric (DOUBLE)
* No duplicate `(run_id, context_id)` rows unless the derived metric explicitly supports multiple samples (then include `sample_index` in the contract)

### 10.3 Determinism tests

Run each derived metric query twice in the same DB snapshot and verify identical results.

Additionally:

* Ban or flag volatile functions (`now()`, `random()`, etc.) unless explicitly allowed.

### 10.4 Dependency coverage tests

* For each derived metric definition, verify the referenced base metrics exist in `metric_defs` (static analysis + runtime).
* Negative tests where a dependency is missing should yield:

  * empty result set, or
  * NULL value rows,
    depending on the intended semantics (codify per derived metric).

### 10.5 “SQL safety” tests (application-layer)

Even though DuckDB supports powerful DDL/DML, we should enforce:

* Derived definitions must be **SELECT-only** (no `DROP`, `DELETE`, etc.)
* Enforce via:

  * a conservative parser/regex gate in app code + tests, or
  * executing in a restricted environment (if possible)

---

## 11) Transaction, concurrency, and durability tests

DuckDB has a defined concurrency model:

* One process can read/write.
* Multiple processes can read only in `READ_ONLY` mode.
* Within one process, multiple writer threads can succeed unless they conflict (optimistic concurrency). ([DuckDB][7])

DuckDB also provides ACID transactions with isolation and rollback. ([DuckDB][6])

### 11.1 Multi-connection transactional visibility

**Test pattern:**

* Connection A: `BEGIN;` insert row(s); do not commit.
* Connection B: verify row(s) not visible.
* Connection A: `COMMIT;`
* Connection B: verify row(s) now visible.

DuckDB’s sqllogictest framework explicitly documents this test pattern and how to label multiple connections. ([DuckDB][1])

### 11.2 Concurrency stress (single process)

* Spawn multiple threads inserting:

  * disjoint contexts (should succeed)
  * same PK keys (should conflict -> expected failures)
* Verify:

  * no corruption
  * expected constraint errors only
  * DB remains readable and passes invariants

sqllogictest also supports concurrent loops (multi-thread query execution), with guidance that conflicts may occur and can be accepted when expected. ([DuckDB][1])

### 11.3 Multi-process access rules

* Process A opens DB read/write and performs writes.
* Process B attempts read/write: ensure it fails or is blocked according to expected behavior (assert the exact error message / failure mode that Cogni should handle).
* Process B opens read-only (`access_mode='READ_ONLY'`) and runs reads concurrently: should succeed. ([DuckDB][7])

### 11.4 Crash safety / durability

On-disk DB only:

* Start a transaction, insert many rows, crash/kill the process before commit.
* Reopen DB and verify:

  * no partial rows exist
  * all invariants still hold
* Repeat with commit just before kill to validate persisted state.

---

## 12) Performance and scalability tests

Even though correctness is primary, Cogni’s reporting use case can get huge.

### 12.1 Bulk ingestion benchmarks

* Insert N agents/questions/contexts and M measurements.
* Measure:

  * rows/sec
  * DB file growth
  * impact of constraints (unique keys create ART indexes automatically for PK/unique constraints). ([DuckDB][2])

### 12.2 Canonical plotting query benchmarks

Benchmark queries that mirror report usage:

* tokens over time for one (agent, question)
* tokens over time grouped by model/provider extracted from `agents.spec`
* compare two runs for same contexts/metric
* retrieve latest value per context

### 12.3 Worst-case dimension cardinality

Stress:

* very high cardinality `agent_id` and `question_id`
* many unique `dims` keys
* large JSON `spec` blobs

Define pass criteria:

* query latency bounds (team-defined)
* no OOM for typical “large repo” fixtures

---

## 13) Golden fixtures and regression artifacts

Maintain canonical fixtures:

1. **Tiny fixture** (human-readable, ~10 revisions)
2. **Medium fixture** (~10k revisions, many contexts)
3. **Large fixture** (stress scale)

For each fixture:

* store expected outputs for:

  * `v_points` queries
  * orphan checks
  * derived metrics results
* treat these as snapshot tests (diff outputs in CI)

Additionally:

* maintain a **fuzz regression corpus**:

  * any failing randomized case becomes a saved fixture.

---

## 14) Suggested “always-on” invariant checks (SQL)

Create a reusable test helper (in code or as a SQL script) that runs:

* Orphan checks (Section 8)
* Measurement type invariant (Section 6.4)
* Status/error_message invariants
* Duplicate detection in tables that should be unique by design
* `v_points` shape sanity: no NULL `ts` in rows that should plot

These invariants become your “DB health check” and can be run:

* in tests
* after ingestion in development
* optionally in production when generating a report artifact

---

## 15) Notes on future evolution

If we later add:

* explicit FKs (with DuckDB’s limitations) ([DuckDB][3])
* additional views/macros for derived metrics (DuckDB supports table macros that return tables of arbitrary shape) ([DuckDB][8])

…extend the suite with:

* FK enforcement tests + deletion behavior tests
* macro creation + upgrade tests
* backward compatibility tests for persisted DBs

---

### Bottom line

This plan treats the schema as a **hard contract** and tests it from every angle: DDL correctness, constraint enforcement, application-level integrity invariants, typed value correctness, JSON/MAP semantics, view contracts, derived metrics contracts, concurrency/transactions, durability, and performance at extreme scales—using both deterministic fixtures and heavy randomized/fuzz generation.

[1]: https://duckdb.org/docs/stable/dev/sqllogictest/multiple_connections.html "Multiple Connections – DuckDB"
[2]: https://duckdb.org/docs/stable/sql/constraints.html "Constraints – DuckDB"
[3]: https://duckdb.org/docs/stable/sql/statements/create_table.html "CREATE TABLE Statement – DuckDB"
[4]: https://duckdb.org/docs/stable/data/json/overview.html "JSON Overview – DuckDB"
[5]: https://duckdb.org/docs/stable/sql/data_types/map.html "Map Type – DuckDB"
[6]: https://duckdb.org/docs/stable/sql/statements/transactions.html "Transaction Management – DuckDB"
[7]: https://duckdb.org/docs/stable/connect/concurrency.html "Concurrency – DuckDB"
[8]: https://duckdb.org/docs/stable/sql/statements/create_macro.html "CREATE MACRO Statement – DuckDB"
