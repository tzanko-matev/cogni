# Milestones

## M1 - CLI foundation and configuration

- Goal: establish the core CLI surface and config validation.
- Deliverables: `cogni init`, `cogni validate`, schema folder, config parsing, and clear error messages.
- Exit criteria: sample config passes validation; invalid configs fail fast with actionable errors.

## M2 - Task runtime and metrics

- Goal: run QA-only tasks with a built-in agent and capture metrics.
- Deliverables: OpenRouter integration, tool allowlist, citation checks, per-task budgets, `results.json` output.
- Exit criteria: a sample repo run completes with partial-failure handling and stable output paths.

## M3 - Compare and report

- Goal: make trend analysis and reporting usable for stakeholders.
- Deliverables: `cogni compare` for base/head or commit ranges, `cogni report` generating `report.html` with charts.
- Exit criteria: a commit range produces a report with clear task-level and trend summaries.

## M4 - Docs and examples

- Goal: reduce time-to-first-benchmark and clarify operations.
- Deliverables: example configs, sample question suite, setup/build/run docs, troubleshooting notes.
- Exit criteria: a new repo can complete a benchmark in under 15 minutes using docs only.

## M5 - Beta readiness and feedback

- Goal: validate utility with early adopters and stabilize UX.
- Deliverables: release notes, versioned CLI artifacts, feedback loop with tracked issues.
- Exit criteria: at least one pilot repo reports actionable insights and no blocking UX gaps.

## Tracking

- Track milestones in the issue tracker with linked PRs.
- Record milestone completion in [spec/roles/working/legal-procurement/legal/changelog.md](/working/legal-procurement/legal/changelog/).
