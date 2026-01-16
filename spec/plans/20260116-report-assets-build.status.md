# Status: Avoid tracked report asset build output

Date: 2026-01-16
Plan: `spec/plans/20260116-report-assets-build.plan.md`

## Scope

Decouple frontend builds from tracked embedded assets while keeping a manual
sync step for updates.

## Relevant files

- `web/vite.config.ts`
- `Justfile`
- `.gitignore`
- `internal/reportserver/assets/*`

## Progress

- Status: DONE
- Last updated: 2026-01-16
- Completed:
  - Plan/status files created.
  - Step 1: move build output to `web/dist`.
  - Step 2: add explicit asset sync workflow.
  - Step 3: refresh embedded assets using the sync step.
