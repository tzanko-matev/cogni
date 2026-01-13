# Memo: Reporting “generalized line chart” + DuckDB measurement schema (Cogni research)

**Date:** 2026-01-14
**Scope:** This memo captures the design discussion around (1) Cogni’s core reporting visualization (“generalized line chart”), (2) how to handle dense commit histories via per-day grouping into connected components + candlesticks, and (3) a DuckDB schema for storing *original* measurements (with derived measurements computed from the stored data).
**Non-goal (explicit):** We will **not** add an `agent_calls` trace table for now. Any commit graph computations (transitive reduction edges, connected components, etc.) will be done **client-side**.

---

## 1) Visualization concept: generalized line chart

We want a chart with:

* **X axis:** time
* **Y axis:** metric value
* **Vertices:** each revision (git commit / jj change) that has a measurement becomes a dot at `(timestamp, value)` where timestamp is a “reasonable” commit-associated time (typically committer time normalized to UTC).

**Edges:** connect two measured revisions **A → B** if:

1. A is an ancestor of B, and
2. there is no measured revision C such that `A ancestor of C` and `C ancestor of B`.

This is essentially the “transitive reduction” of the measurement-induced subgraph (computed client-side).

### Dense history fallback (grouping)

When there are too many revisions close in time, we group points by time bins (example: per day):

1. For each day, take all measured revisions in that day.
2. Split them into **connected components** (connectivity determined by revision ancestry restricted to that day’s measured nodes; computed client-side).
3. For each component, draw a **candlestick** (open/close/high/low) for the day:

   * **open:** metric at earliest timestamp in the component (within that day)
   * **close:** metric at latest timestamp in the component
   * **high/low:** max/min metric within the component
4. Candles for the same day (different components) are drawn in **different colors**.
5. Draw **thin linking lines** between consecutive components (across days) to indicate continuity; represent each component with a “vertex” (e.g., midpoint in time, and y at `(open+close)/2`).

> Important: the *connected components* and *links between components* are not computed in SQL; they are computed in the client and stored or emitted as data.

---

## 2) Using Mosaic/vgplot for rendering (summary)

We can implement both views using vgplot primitives:

* Raw commit graph view:

  * `dot` for measured revisions
  * `link` for ancestor edges (requires precomputed edge list between measured revisions)
* Grouped/candlestick view:

  * `ruleX` for wick (low→high) and for body (open→close), or body via `rect`
  * `link` for thin component-to-component lines
  * color encodes component identity (within a day) or a stable component lineage id

The key requirement for vgplot is: **the data must already be in tabular form** (points + edges + optional candle aggregates). vgplot renders; graph logic remains external.

---

## 3) DuckDB storage goals

We need a schema that supports:

1. **Many metric types** (numeric scalar metrics like tokens, but also potentially booleans/strings/json blobs later).
2. A major dimension: **AI agent used to compute the metric**, where the agent description is a **complicated object** with schema unknown in advance. We want to store it “completely”.
3. A second key dimension: **question** asked of the agent (also potentially complex/evolving).
4. Ability to query “plot-ready points” consistently: `(repo, revision, ts, metric, agent, question, value)`.
5. Ability to define and plot **derived metrics** (computed from other metrics stored in the DB), without duplicating raw data.

**Design choice:** Normalize “agent spec” and “question spec” into their own tables as JSON, deduplicate by a stable key, and reference them from a “context” table. This keeps the measurement fact table compact and avoids repeating large JSON blobs per row.

---

## 4) Proposed schema (DuckDB)

### 4.1 Repositories

```sql
CREATE TABLE repos (
  repo_id      UUID PRIMARY KEY,
  name         VARCHAR NOT NULL,
  vcs          VARCHAR NOT NULL,        -- 'git' | 'jj' | ...
  remote_url   VARCHAR,
  created_at   TIMESTAMP DEFAULT now()
);
```

---

### 4.2 Revisions (commits / jj changes)

Store at least one canonical timestamp used for plotting on X.

```sql
CREATE TABLE revisions (
  repo_id        UUID NOT NULL,
  rev_id         VARCHAR NOT NULL,       -- git SHA / jj change id
  ts_utc         TIMESTAMP NOT NULL,      -- chosen plotting timestamp, normalized to UTC

  author         VARCHAR,
  committer      VARCHAR,
  summary        VARCHAR,

  PRIMARY KEY (repo_id, rev_id)
);
```

---

### 4.3 Optional: parent edges (for client-side ancestry reconstruction)

If Cogni wants reports to be self-contained offline, store revision parents:

```sql
CREATE TABLE revision_parents (
  repo_id       UUID NOT NULL,
  child_rev_id  VARCHAR NOT NULL,
  parent_rev_id VARCHAR NOT NULL,

  PRIMARY KEY (repo_id, child_rev_id, parent_rev_id)
);
```

---

### 4.4 Metric definitions

Defines what a “metric” is, its unit, and how values are stored.

```sql
CREATE TABLE metric_defs (
  metric_id      UUID PRIMARY KEY,
  name           VARCHAR NOT NULL UNIQUE,   -- e.g. "tokens"
  description    VARCHAR,
  unit           VARCHAR,                  -- e.g. "tokens", "ms", "%", "count"
  physical_type  VARCHAR NOT NULL,         -- 'DOUBLE'|'BIGINT'|'BOOLEAN'|'VARCHAR'|'JSON'|'BLOB'
  created_at     TIMESTAMP DEFAULT now()
);
```

---

### 4.5 Runs (provenance of collection)

A “run” groups a measurement collection pass (tool version, config, etc.).

```sql
CREATE TABLE runs (
  run_id          UUID PRIMARY KEY,
  repo_id         UUID NOT NULL,
  collected_at    TIMESTAMP NOT NULL,      -- could be TIMESTAMPTZ if you prefer

  tool_name       VARCHAR NOT NULL,        -- "cogni"
  tool_version    VARCHAR,
  schema_version  VARCHAR,

  config          JSON,
  environment     JSON,
  notes           VARCHAR
);
```

---

## 5) Reflecting the AI agent dimension properly

### 5.1 Agents

Agents are complicated schema-unknown objects. Store them as JSON once, deduplicate with a stable “fingerprint”.

```sql
CREATE TABLE agents (
  agent_id      UUID PRIMARY KEY,
  agent_key     VARCHAR NOT NULL UNIQUE,  -- fingerprint of canonicalized JSON spec
  spec          JSON NOT NULL,            -- full agent description (schema-unknown)
  display_name  VARCHAR,                  -- optional label
  created_at    TIMESTAMP
);
```

**Agent identity recommendation:**

* Canonicalize `spec` client-side (stable key ordering, consistent formatting).
* Hash canonical JSON to get `agent_key` (e.g., sha256 hex).
* Upsert into `agents` by `agent_key`.

This guarantees:

* stable dedup
* stable join key
* avoids storing large JSON repeatedly in the fact table

---

### 5.2 Questions

Treat questions similarly; they evolve (prompt templates, retrieval parameters, evaluation harness, etc.).

```sql
CREATE TABLE questions (
  question_id   UUID PRIMARY KEY,
  question_key  VARCHAR NOT NULL UNIQUE,  -- fingerprint of canonicalized JSON spec
  spec          JSON NOT NULL,            -- full question description
  title         VARCHAR,
  created_at    TIMESTAMP
);
```

---

## 6) Contexts: where dimensions come together

Instead of stuffing agent/question + many optional dimensions into `measurements`, use a normalized “context” row that represents:

> “At repo R, revision V, for agent A, for question Q, and other dims D”

```sql
CREATE TABLE contexts (
  context_id    UUID PRIMARY KEY,
  context_key   VARCHAR NOT NULL UNIQUE,      -- fingerprint of canonicalized context object

  repo_id       UUID NOT NULL,
  rev_id        VARCHAR NOT NULL,

  agent_id      UUID,                         -- NULL for non-agent metrics
  question_id   UUID,                         -- NULL if not question-based

  -- Lightweight variable labels (string->string):
  dims          MAP(VARCHAR, VARCHAR),

  -- Optional structured dimension (schema-unknown but smaller than agent/question):
  scope         JSON,                         -- e.g. {"kind":"repo"} or {"kind":"path","path":"src/foo.rs"}

  created_at    TIMESTAMP
);
```

### Why a `contexts` table?

* Keeps `measurements` compact and generic.
* Avoids schema churn as we add more dimensions.
* Lets us define “series” as `context_id` (or by grouping context fields), rather than a single string `series_key`.

### Context identity recommendation

Compute a stable `context_key` client-side from:

* `repo_id`, `rev_id`
* `agent_key` (or `agent_id`)
* `question_key` (or `question_id`)
* canonicalized `dims`
* canonicalized `scope`

Then upsert by `context_key`.

---

## 7) Measurements: append-only fact table

One row = one observation of one metric for one context in one run.

```sql
CREATE TABLE measurements (
  run_id        UUID NOT NULL,
  context_id    UUID NOT NULL,
  metric_id     UUID NOT NULL,

  sample_index  INTEGER NOT NULL DEFAULT 0, -- repeated trials
  observed_at   TIMESTAMP,                  -- when produced (optional)

  value_double  DOUBLE,
  value_bigint  BIGINT,
  value_bool    BOOLEAN,
  value_varchar VARCHAR,
  value_json    JSON,
  value_blob    BLOB,

  status        VARCHAR NOT NULL DEFAULT 'ok',  -- 'ok'|'error'|'skipped'
  error_message VARCHAR,
  raw           JSON,                          -- optional lossless payload

  PRIMARY KEY (run_id, context_id, metric_id, sample_index)
);
```

### How “tokens” fits

* Define a metric def `name = 'tokens'`, `physical_type = 'BIGINT'`, `unit = 'tokens'`.
* For each (commit, agent, question) context, insert a measurement row with:

  * `value_bigint = <token_count>`
  * `metric_id = tokens_metric_id`

This matches Cogni’s core value proposition:

> “For each question and agent, count tokens used for the agent to answer questions about the codebase at this commit.”

---

## 8) Plot-friendly views

To plot time series consistently, define a view that joins the needed dimensions:

```sql
CREATE VIEW v_points AS
SELECT
  c.repo_id,
  c.rev_id,
  r.ts_utc AS ts,

  m.run_id,
  md.name  AS metric,

  c.agent_id,
  c.question_id,
  c.dims,
  c.scope,

  m.sample_index,

  CASE md.physical_type
    WHEN 'DOUBLE' THEN m.value_double
    WHEN 'BIGINT' THEN CAST(m.value_bigint AS DOUBLE)
    ELSE NULL
  END AS value,

  m.status,
  m.error_message
FROM measurements m
JOIN contexts c
  ON c.context_id = m.context_id
JOIN revisions r
  ON r.repo_id = c.repo_id AND r.rev_id = c.rev_id
JOIN metric_defs md
  ON md.metric_id = m.metric_id;
```

Example plotting query (tokens over time for a specific question + agent):

```sql
SELECT ts, value
FROM v_points
WHERE metric = 'tokens'
  AND status = 'ok'
  AND agent_id = ?
  AND question_id = ?
ORDER BY ts;
```

---

## 9) Derived metrics (computed from stored measurements)

We want “derived measurements” whose values are computed from other measurements in the DB.

### Minimal definition table

```sql
CREATE TABLE derived_metric_defs (
  derived_metric_id UUID PRIMARY KEY,
  name              VARCHAR NOT NULL UNIQUE,
  description       VARCHAR,
  unit              VARCHAR,

  -- Must return a table shaped like points:
  -- (repo_id, rev_id, ts, run_id, context_id, value)
  sql_select         VARCHAR NOT NULL,

  created_at        TIMESTAMP DEFAULT now(),
  updated_at        TIMESTAMP
);
```

### Execution model options

* **Option A:** Cogni loads a derived metric definition and runs `sql_select` directly as a query (with parameters via prepared statements).
* **Option B:** Cogni materializes derived metrics as DuckDB views/macros at open time (useful when you want `SELECT * FROM derived_metric_name(...)` ergonomics). (We discussed this as a future-friendly path, even if you start with Option A.)

Example derived metric idea:

* `tokens_per_kloc` = `tokens / (loc_changed / 1000)` (assuming you have a `loc_changed` metric).

---

## 10) How this supports the visualization requirements

**Raw commit graph view:**

* `v_points` gives dots: `(ts, value)` per context.
* Client computes:

  * measured commit set per metric/context filter
  * reduced ancestor edges between measured revisions
* vgplot can render:

  * points from `v_points`
  * edges from a client-produced edge table (or in-memory edge list)

**Grouped candlestick view:**

* Client groups by day and computes connected components and OHLC per component.
* Client emits a small “candles” table with columns: `(day, component_id, open, close, high, low, x)` and optionally component-node midpoints and linking edges.
* vgplot renders candles and thin linking lines.

---

## 11) Open questions / future extensions

* Whether to persist `revision_parents` in the DB for offline reconstruction (recommended if reports should be standalone).
* Whether to promote common dims (besides agent/question) into explicit columns (only if query performance demands it).
* Whether to add an `agent_calls` trace table later (out of scope now, but a natural extension if we later want prompt/completion token splits, retries, latencies, etc.).

---

## Appendix A: Ingestion workflow (recommended)

1. Upsert repo + revisions (+ optional revision_parents).
2. Upsert metric_defs.
3. Upsert agents by `agent_key` (hash of canonical JSON spec).
4. Upsert questions by `question_key`.
5. For each measurement, upsert context by `context_key` and then insert measurement row keyed by `(run_id, context_id, metric_id, sample_index)`.

This keeps ingestion idempotent and avoids duplicate agent/question JSON.
