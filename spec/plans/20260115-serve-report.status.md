# Status: Serve browser report from DuckDB (2026-01-15)

## Plan
- spec/plans/20260115-serve-report.plan.md

## References
- spec/inbox/vgplot-research.md
- spec/features/output-report-html.feature
- spec/features/cli.feature

## Relevant files
- cmd/cogni/main.go
- internal/cli/cli.go
- internal/cli/serve.go (new)
- internal/reportserver (new)
- web/ (new)
- flake.nix
- Justfile
- spec/features/output-report-serve.feature (new)

## Status
- State: IN PROGRESS
- Completed steps: None
- Current step: Step 1 (CLI command + server skeleton)
- Notes: Plan and status files created.
