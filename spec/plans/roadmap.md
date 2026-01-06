# Roadmap

## Guiding principles

- Keep the MVP local, read-only, and deterministic.
- Optimize for time-to-first-benchmark under 15 minutes.
- Make reports understandable to non-engineering stakeholders.

## Near term (MVP)

- CLI commands: `init`, `validate`, `run`, `compare`, `report` with stable flags and help text.
- Config and schema support for tasks, agents, budgets, and output location.
- Built-in agent with tool allowlist, citation validation, and QA plus Cucumber evaluation task formats.
- Metrics capture per attempt (tokens, time, tool calls, files read, model, agent ID).
- Deterministic outputs under `<output_dir>/<commit>/<run-id>/` including `results.json` and `report.html`.
- Range-based comparison and trend reporting over commit ranges.
- Example configs, sample question suite, and a clear getting-started guide.

## Medium term (V1)

- Multi-provider support (OpenAI, Anthropic) with minimal config changes.
- External agent adapters (Codex, Claude Code, Gemini) with a stable interface.
- Multi-VCS support for commit metadata (jujutsu plus Git).
- Improved report UX: filtering, export, and task grouping.
- CI smoke tests with a fixture repo and a golden report.

## Long term (post v1)

- Sandboxed runners for reproducible and isolated benchmarking.
- Team dashboards and hosted report sharing.
- Baseline packs for common stacks and project types.
- Paid offering with enterprise admin controls (optional).
