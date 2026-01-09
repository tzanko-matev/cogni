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
- State: DONE
- Progress:
  - Plan and status files created.
  - Added question spec package with JSON/YAML parsing, normalization, and answer XML parsing tests.
  - Implemented question_eval runner/results/summary, CLI eval command, and report updates with tests.
  - Removed cucumber pipeline/adapters/tests, cleaned dependencies, and updated docs with Question Spec evaluation.
  - Added an example Question Spec file inspired by `spec/features`.
  - Updated `.cogni/config.yml` to use `question_eval` with the example question spec.
  - Fixed `cogni eval` to accept flags after the questions file and added CLI coverage.

## Next Actions
- None. Completed on 2026-01-09.
