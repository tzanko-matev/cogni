# Status: Question Spec Core Evaluation

Date: 2026-01-09

## Plan
- Plan: `spec/plans/20260109-question-spec-core.plan.md`

## Scope
- Replace cucumber evaluation with a core question-spec evaluation (JSON/YAML input).
- Add `cogni eval <questions_file> --agent <id>` CLI.
- Keep config-defined `question_eval` tasks runnable via `cogni run`.
- Remove cucumber/adapters logic and docs.

## Relevant Specs & Notes
- `spec/engineering/configuration.md`
- (New) `spec/design/question-evaluation.md`

## Relevant Code
- `internal/spec/types.go`
- `internal/config/*`
- `internal/runner/*`
- `internal/cli/*`
- `internal/report/*`

## Status
- State: IN PROGRESS
- Progress:
  - Plan and status files created.
  - Added question spec package with JSON/YAML parsing, normalization, and answer XML parsing tests.

## Next Actions
- Wire question spec evaluation into runner/results/summary.
- Add `cogni eval` command and question_eval config validation.
- Remove cucumber packages/tests/docs and adapters.
