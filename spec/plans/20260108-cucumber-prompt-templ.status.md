# Cucumber Prompt Templating Status

Status: In Progress

ID: 20260108-cucumber-prompt-templ.status

Created: 2026-01-08

Linked plan: [spec/plans/20260108-cucumber-prompt-templ.plan.md](/plans/20260108-cucumber-prompt-templ.plan/)

## Current status
- Cucumber prompt rendering now uses a compiled `templ` component.
- `prompt_template` was removed from task config/types and validation; tests now
  rely on the built-in prompt.
- Report HTML and single-run placeholder reports now render via `templ`.
- Scaffold config templating still pending.

## What was done so far
- Added `internal/prompt` with a compiled `templ` cucumber prompt and render helper.
- Switched the cucumber runner to the built-in prompt renderer.
- Removed `prompt_template` from `spec.TaskConfig` and cucumber validation.
- Updated cucumber tests and fixtures to drop `prompt_template`.
- Regenerated templ output and updated `go.mod`, `go.sum`, and `vendor/`.
- Added `templ` report templates for both multi-run reports and single-run stubs.
- Updated report rendering and output writer to use the compiled templates.

## Next steps
- Convert scaffolded config output to templ.
- Update specs/docs to describe the built-in cucumber prompt.
- Clean up remaining placeholder-based templating references.

## Latest test run
- 2026-01-08: `go test ./...`

## Relevant source files (current or planned)
- internal/runner/cucumber_helpers.go
- internal/runner/cucumber_feature.go
- internal/spec/types.go
- internal/config/validate_tasks.go
- internal/cli/e2e_compare_test.go
- internal/runner/cucumber_batch_test.go
- internal/runner/cucumber_errors_test.go
- internal/report/report.go
- internal/runner/output_writer.go
- internal/config/scaffold.go

## Relevant spec documents
- spec/design/cucumber-evaluation.md
- spec/engineering/configuration.md
- spec/requirements/functional.md
- spec/features/cucumber-adapter-godog.feature
- spec/features/cucumber-adapter-manual.feature
