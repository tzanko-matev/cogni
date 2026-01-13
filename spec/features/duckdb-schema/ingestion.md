# Ingestion & Idempotency

This file explains how to compute stable fingerprints and how to upsert the
normalized dimension tables (`agents`, `questions`, `contexts`). The DB schema
assumes the application is responsible for these keys.

## Recommended ingestion order

1) Upsert `repos`
2) Upsert `revisions` (+ optional `revision_parents`)
3) Upsert `metric_defs`
4) Upsert `agents` by `agent_key`
5) Upsert `questions` by `question_key`
6) Upsert `contexts` by `context_key`
7) Insert `measurements`

## Fingerprint keys

We use stable string fingerprints to deduplicate JSON specs:
- `agent_key` = sha256(canonical_json(agent_spec))
- `question_key` = sha256(canonical_json(question_spec))
- `context_key` = sha256(canonical_json({repo_id, rev_id, agent_key, question_key, dims, scope}))

### Canonical JSON (Go snippet)

```go
// CanonicalJSON returns deterministic JSON for hashing.
// It relies on encoding/json's sorted map keys after normalization.
func CanonicalJSON(value any) ([]byte, error) {
	normalized, err := normalizeJSON(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func normalizeJSON(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, inner := range v {
			norm, err := normalizeJSON(inner)
			if err != nil {
				return nil, err
			}
			out[k] = norm
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i := range v {
			norm, err := normalizeJSON(v[i])
			if err != nil {
				return nil, err
			}
			out[i] = norm
		}
		return out, nil
	default:
		return v, nil
	}
}

func FingerprintJSON(value any) (string, error) {
	data, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
```

Notes for juniors:
- `encoding/json` sorts map keys when marshaling, which gives stable output.
- Always normalize nested objects/arrays before marshaling.

### dims canonicalization

`dims` is a `map[string]string` and should be made deterministic before hashing.
Example: convert to a sorted `[]string` of `key=value` pairs and include it in
`context_key` input.

```go
func CanonicalDims(dims map[string]string) []string {
	keys := make([]string, 0, len(dims))
	for k := range dims {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+dims[k])
	}
	return out
}
```

## Upsert patterns (SQL)

We deduplicate by `*_key` but use UUID primary keys for join efficiency.

```sql
-- Agents
INSERT INTO agents (agent_id, agent_key, spec, display_name, created_at)
VALUES (?, ?, ?, ?, now())
ON CONFLICT (agent_key)
DO UPDATE SET agent_key = excluded.agent_key
RETURNING agent_id;

-- Questions
INSERT INTO questions (question_id, question_key, spec, title, created_at)
VALUES (?, ?, ?, ?, now())
ON CONFLICT (question_key)
DO UPDATE SET question_key = excluded.question_key
RETURNING question_id;

-- Contexts
INSERT INTO contexts (
  context_id, context_key, repo_id, rev_id, agent_id, question_id, dims, scope, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, now())
ON CONFLICT (context_key)
DO UPDATE SET context_key = excluded.context_key
RETURNING context_id;
```

Notes:
- `ON CONFLICT` requires a UNIQUE constraint on the key columns (already in
  schema).
- We keep updates minimal to avoid unexpected edits to existing records.

## Measurement inserts

Measurements are append-only; no upserts.

```sql
INSERT INTO measurements (
  run_id, context_id, metric_id, sample_index, observed_at,
  value_double, value_bigint, value_bool, value_varchar, value_json, value_blob,
  status, error_message, raw
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
```

The application enforces the invariant: exactly one value column is set for
`status='ok'` and it matches `metric_defs.physical_type`.

Next: `test-suite.md`
