# Goals and Non-Goals

## Goals

- Provide a repeatable cognitive benchmark for a codebase.
- Allow teams to define question suites tied to key product features.
- Measure correctness and effort metrics (tokens, time, tool usage).
- Track trends over commits via `compare` and `report`.
- Keep the MVP simple, local, and read-only.

## Non-goals

- Code-changing tasks (patches/tests/linting).
- Sandboxed runners or hosted SaaS dashboards.
- Multi-tenant permissions, RBAC, or SSO.
- External agent integrations or multi-provider support in MVP.
- LLM-as-judge evaluation.
