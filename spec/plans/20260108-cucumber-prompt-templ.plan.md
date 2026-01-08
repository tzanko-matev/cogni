# Cucumber Prompt Templating Plan

Status: Done

ID: 20260108-cucumber-prompt-templ

Created: 2026-01-08
Completed: 2026-01-08

Linked status: [spec/plans/20260108-cucumber-prompt-templ.status.md](/plans/20260108-cucumber-prompt-templ.status/)

## Goal
Replace manual string templating with compiled `templ` components where we
currently use placeholder-style templates, starting with `cucumber_eval`
prompts and extending to HTML reports and scaffolded config output.

## Scope
- Introduce a `templ` component for the cucumber prompt.
- Replace runtime placeholder replacement with the generated template renderer.
- Convert HTML report rendering (both the report builder and the placeholder
  report) to `templ`.
- Convert the scaffolded config template to `templ` (with safe escaping rules
  for YAML output).
- Remove or deprecate `prompt_template` from `cucumber_eval` config (favoring
  the built-in prompt), and update validation accordingly.
- Update specs/docs/examples/tests to match the new UX.
- Wire `templ generate` into the workflow so generated code is kept in sync.

## Non-goals
- Changing the `cucumber_eval` evaluation logic or response schema.
- Converting all string formatting to templates across the repo beyond the
  identified prompt/report/scaffold targets.
- Introducing new task types or adapter behavior.

## Inputs and references
- spec/design/cucumber-evaluation.md
- spec/engineering/configuration.md
- spec/requirements/functional.md
- spec/features/cucumber-adapter-godog.feature
- spec/features/cucumber-adapter-manual.feature
- internal/runner/cucumber_feature.go
- internal/runner/cucumber_helpers.go
- internal/spec/types.go
- internal/config/validate_tasks.go
- internal/report/report.go
- internal/runner/output_writer.go
- internal/config/scaffold.go

## Plan conventions
- Phases are sequential.
- Build/test gate: run `go test ./...` after code or tests change.

## Phases

### Phase 0 - Templ integration decision
- Work:
  - Choose template location (e.g., `internal/prompt/`), naming, and render API.
  - Add `github.com/a-h/templ` dependency.
  - Decide how `templ generate` runs (manual, `go:generate`, Justfile target).
- Verification:
  - Generated code compiles without touching runtime logic.
- Exit criteria: templ component location and generation workflow are defined.

### Phase 1 - Replace cucumber prompt rendering
- Work:
  - Implement `CucumberPrompt` in a `.templ` file with required inputs:
    `featurePath`, `featureText`, `exampleIDs`.
  - Add a small helper to render the component to a string.
  - Update the cucumber runner to use the templ render helper instead of
    `renderCucumberPrompt` and remove placeholder replacement logic.
- Verification:
  - Unit tests compile and prompt includes path/text/IDs in the expected order.
- Exit criteria: runtime prompt rendering uses templ only.

### Phase 2 - Report HTML templating
- Work:
  - Implement `templ` component(s) for HTML report output.
  - Replace string builders in `internal/report/report.go` with template render.
  - Replace the HTML stub in `internal/runner/output_writer.go` with template render.
- Verification:
  - Existing report outputs remain equivalent (structure + fields).
- Exit criteria: HTML report paths use `templ` exclusively.

### Phase 3 - Scaffold config templating
- Work:
  - Implement a `templ` component that renders the scaffolded YAML config.
  - Ensure YAML-appropriate escaping (avoid HTML escaping artifacts).
  - Replace placeholder replacement in `internal/config/scaffold.go`.
- Verification:
  - Scaffold output matches existing template content for common paths.
- Exit criteria: scaffold config is rendered via `templ`.

### Phase 4 - Config/spec updates
- Work:
  - Update `spec.TaskConfig` / validation to remove the `prompt_template`
    requirement for `cucumber_eval` (either forbid it or ignore it).
  - Update specs and config examples to describe the built-in prompt.
  - Update any references that document `prompt_template` as required.
- Verification:
  - Config validation tests updated and passing.
- Exit criteria: docs and validation align with built-in prompt UX.

### Phase 5 - Test and fixture updates
- Work:
  - Update cucumber runner tests and CLI/e2e tests that set `prompt_template`.
  - Update feature specs or fixtures that mention prompt templates explicitly.
- Verification:
  - `go test ./...` passes (in `nix develop` if required).
- Exit criteria: tests pass with built-in prompt.

### Phase 6 - Cleanup
- Work:
  - Remove obsolete helpers tied to placeholder-based templating.
  - Ensure docs and examples do not reference removed templates.
- Verification:
  - `rg` shows no remaining prompt/report/scaffold placeholder replacements.
- Exit criteria: codebase is consistent with templ-based rendering.

## Acceptance criteria
- `cucumber_eval` prompt is rendered via a compiled `templ` component.
- HTML report outputs are rendered via `templ`.
- Scaffolded config template is rendered via `templ`.
- No runtime placeholder replacement is used for cucumber prompts or scaffold config.
- Config validation/docs no longer require `prompt_template` for `cucumber_eval`.
- Tests and feature specs are updated accordingly.
