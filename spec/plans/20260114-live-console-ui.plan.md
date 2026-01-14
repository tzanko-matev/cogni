# Plan: Live console UI for concurrent question_eval (2026-01-14)

## Goal
Implement a live, non-verbose console UI that lists each question with an
updating status field (including tool call activity) using Bubble Tea + Bubbles
+ Lip Gloss. The UI should only run when stdout is a TTY and `--verbose` is not
set. Otherwise, preserve current plain output behavior.

## References
- spec/features/output-live-ui/overview.md
- spec/features/output-live-ui/ui.md
- spec/features/output-live-ui/integration.md
- spec/features/output-live-ui/test-suite.md
- spec/features/output-live-ui/implementation-plan.md
- spec/features/output-live-ui/testing.feature
- spec/features/output-console-summary.feature

## Steps
1) UI state + reducer
   - Add `internal/ui/live` package (or equivalent).
   - Implement event types and a pure reducer for question rows.
   - Build Bubble Tea model with list/table layout.
   - Tests: reducer transitions and tool sub-status updates.
   - Tests: `nix develop -c go test ./internal/ui/live`.

2) Runner event emission
   - Add `RunObserver` to `runner.RunParams`.
   - Emit events from `runQuestionTask` and `executeQuestionJob`.
   - Add scheduler hooks for reserve/deny/error.
   - Tests: runner event ordering and backoff event.
   - Tests: `nix develop -c go test ./internal/runner`.

3) Tool activity wiring
   - Wrap `ToolExecutor` to emit tool start/finish events.
   - Surface tool status in question rows.
   - Tests: tool activity integration tests.
   - Tests: `nix develop -c go test ./internal/agent ./internal/runner`.

4) CLI integration
   - Add `--ui=auto|live|plain` flag for `cogni run` and `cogni eval`.
   - Auto mode uses TTY detection + non-verbose requirement.
   - Stop UI on completion and print existing summary output.
   - Tests: UI mode selection logic.
   - Tests: `nix develop -c go test ./internal/cli`.

5) Docs + BDD
   - Ensure docs remain in sync with behavior.
   - Implement BDD scenarios under `spec/features/output-live-ui/testing.feature`.
   - Tests: `nix develop -c go test ./tests/... -tags=cucumber` (if applicable).

## Completion
Mark this plan and status file as DONE when all steps and tests are complete.
