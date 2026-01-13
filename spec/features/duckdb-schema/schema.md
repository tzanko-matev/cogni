# Schema DDL (v1)

This file defines the DuckDB schema described in the inbox memo. The goal is a
stable, append-only measurement store with normalized dimensions (agents,
questions, contexts).

## Design principles

- Measurements are append-only facts.
- Large or evolving JSON (agent/question specs) are stored once and referenced
  by id.
- Contexts tie repo+revision+agent+question+dims into a single join key.
- No foreign keys for now (DuckDB limitations); tests enforce integrity.

## DDL (reference)

Use this as the source of truth for the implementation schema file.

```sql
-- Optional but recommended for JSON extraction tests in older DuckDB versions.
-- INSTALL json;
-- LOAD json;

CREATE TABLE repos (
  repo_id      UUID PRIMARY KEY,
  name         VARCHAR NOT NULL,
  vcs          VARCHAR NOT NULL,
  remote_url   VARCHAR,
  created_at   TIMESTAMP DEFAULT now()
);

CREATE TABLE revisions (
  repo_id        UUID NOT NULL,
  rev_id         VARCHAR NOT NULL,
  ts_utc         TIMESTAMP NOT NULL,
  author         VARCHAR,
  committer      VARCHAR,
  summary        VARCHAR,
  PRIMARY KEY (repo_id, rev_id)
);

-- Optional but recommended when reports must be standalone offline.
CREATE TABLE revision_parents (
  repo_id       UUID NOT NULL,
  child_rev_id  VARCHAR NOT NULL,
  parent_rev_id VARCHAR NOT NULL,
  PRIMARY KEY (repo_id, child_rev_id, parent_rev_id)
);

CREATE TABLE metric_defs (
  metric_id      UUID PRIMARY KEY,
  name           VARCHAR NOT NULL UNIQUE,
  description    VARCHAR,
  unit           VARCHAR,
  physical_type  VARCHAR NOT NULL,
  created_at     TIMESTAMP DEFAULT now()
);

CREATE TABLE runs (
  run_id          UUID PRIMARY KEY,
  repo_id         UUID NOT NULL,
  collected_at    TIMESTAMP NOT NULL,
  tool_name       VARCHAR NOT NULL,
  tool_version    VARCHAR,
  schema_version  VARCHAR,
  config          JSON,
  environment     JSON,
  notes           VARCHAR
);

CREATE TABLE agents (
  agent_id      UUID PRIMARY KEY,
  agent_key     VARCHAR NOT NULL UNIQUE,
  spec          JSON NOT NULL,
  display_name  VARCHAR,
  created_at    TIMESTAMP
);

CREATE TABLE questions (
  question_id   UUID PRIMARY KEY,
  question_key  VARCHAR NOT NULL UNIQUE,
  spec          JSON NOT NULL,
  title         VARCHAR,
  created_at    TIMESTAMP
);

CREATE TABLE contexts (
  context_id    UUID PRIMARY KEY,
  context_key   VARCHAR NOT NULL UNIQUE,
  repo_id       UUID NOT NULL,
  rev_id        VARCHAR NOT NULL,
  agent_id      UUID,
  question_id   UUID,
  dims          MAP(VARCHAR, VARCHAR),
  scope         JSON,
  created_at    TIMESTAMP
);

CREATE TABLE measurements (
  run_id        UUID NOT NULL,
  context_id    UUID NOT NULL,
  metric_id     UUID NOT NULL,
  sample_index  INTEGER NOT NULL DEFAULT 0,
  observed_at   TIMESTAMP,
  value_double  DOUBLE,
  value_bigint  BIGINT,
  value_bool    BOOLEAN,
  value_varchar VARCHAR,
  value_json    JSON,
  value_blob    BLOB,
  status        VARCHAR NOT NULL DEFAULT 'ok',
  error_message VARCHAR,
  raw           JSON,
  PRIMARY KEY (run_id, context_id, metric_id, sample_index)
);

CREATE TABLE derived_metric_defs (
  derived_metric_id UUID PRIMARY KEY,
  name              VARCHAR NOT NULL UNIQUE,
  description       VARCHAR,
  unit              VARCHAR,
  sql_select        VARCHAR NOT NULL,
  created_at        TIMESTAMP DEFAULT now(),
  updated_at        TIMESTAMP
);

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

## Table notes

- `revisions.ts_utc` is the plotting timestamp and must be normalized to UTC.
- `metric_defs.physical_type` drives which `value_*` column is valid.
- `contexts.dims` is a simple string map for lightweight dimensions; use
  `scope` JSON for structured context like `{"kind":"path","path":"src/..."}`.
- `measurements.status` values: `ok`, `error`, `skipped`.

## Derived metrics

Derived metrics are stored as SQL text. The query must return a table with the
shape:
`(repo_id, rev_id, ts, run_id, context_id, value)`.

We execute these queries at report time (not stored by default). Tests enforce
that the SQL compiles and returns the correct shape.

Next: `ingestion.md`
