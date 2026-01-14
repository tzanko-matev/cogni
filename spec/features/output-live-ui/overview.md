# Live Console UI for Concurrent Question Eval (v1)

Audience: junior Go developer. This spec is self-contained. Read files in order.

## Read order

1) `overview.md` (this file)
2) `ui.md`
3) `integration.md`
4) `test-suite.md`
5) `implementation-plan.md`
6) `testing.feature`

## Goal

Provide a live, non-verbose console UI that lists each question with a constantly
updating status field. This is intended for concurrent question evaluation so
users can see progress without enabling verbose logs.

## Scope

- Applies to `question_eval` tasks only (the only concurrent task type).
- Activates only when `--verbose` is **not** used.
- Uses a full-screen TUI when stdout is a TTY; otherwise falls back to plain text.
- Shows tool call activity when tools are invoked.

## Non-goals

- Changing task results or evaluation logic.
- Adding UI for other task types (not supported yet).
- Streaming verbose logs into the live UI.
- Persisting UI state to disk.

## Decisions (source of truth)

- Use Bubble Tea + Bubbles + Lip Gloss for the live UI.
- Default UI mode is **auto**:
  - TTY + not verbose -> live UI.
  - Non-TTY -> plain output (current behavior).
- The final summary output remains the same as current CLI summary.

## Glossary

- **Live UI**: full-screen terminal interface that refreshes in place.
- **Plain output**: current non-verbose output (summary + paths).
- **Question row**: one line in the UI for a specific question.
- **Tool activity**: an in-flight tool call shown as a sub-status for a question.
