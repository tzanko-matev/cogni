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
- State: NOT STARTED
- Progress:
  - Plan and status files created.

## Next Actions
- Update config schema/validation to remove adapters and add question_eval.
- Implement question spec loader and XML answer parsing.
- Add `cogni eval` command and update results/report.
- Remove cucumber packages/tests/docs.
