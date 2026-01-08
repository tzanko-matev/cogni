# Plan: Context Compaction Roadmap

Date: 2026-01-08

## Goal
Introduce a Codex-style compaction pipeline that prevents unbounded prompt growth by summarizing older history while preserving critical recent context, tool state, and instructions.

## Non-Goals
- No new provider integrations beyond the existing OpenRouter path.
- No changes to task evaluation or scoring logic.
- No new storage format for runs outside of optional compaction metadata.

## Current Behavior (Summary)
- `agent.RunTurn` compacts only when `ApproxTokenCount(history)` exceeds `TaskBudget.MaxTokens`.
- `CompactHistory` keeps developer/user instructions, the latest environment block/diff, the latest user message, and all tool outputs.
- No summary message is created; older context is dropped outright.
- Tool outputs are truncated by tool runners, but history still grows linearly and can overflow model context.

## Roadmap

### Phase 1: Policy + Config
- Define a compaction policy with **soft** (auto-compact) and **hard** (budget fail) limits.
- Add optional config fields:
  - `compaction.max_tokens` (soft limit; default to a fraction of budget)
  - `compaction.summary_prompt` or `compaction.summary_prompt_file`
  - `compaction.recent_user_token_budget`
  - `compaction.recent_tool_output_limit`
- Update docs:
  - `spec/engineering/builtin-agent.md` (compaction behavior)
  - `spec/engineering/configuration.md` (new config examples)

### Phase 2: Summarization + History Rebuild
- Add a summary prompt template and a summary prefix constant.
- Implement `SummarizeHistory` using the existing provider stream (single-turn call, no tools).
- Replace `CompactHistory` with summary-aware compaction:
  - Preserve developer instructions, user instructions, most recent environment block/diff.
  - Keep the most recent user messages within a token budget (drop oldest).
  - Append a summary message with a fixed prefix.
  - Retain tool call inputs/outputs per policy (e.g., last N tool outputs or tool outputs after the most recent user message).
  - Drop prior summary messages to avoid summary-of-summary drift.
- If the summarization prompt itself is too large, iteratively trim oldest items before retrying (Codex-style).

### Phase 3: Run Loop Integration
- Trigger auto-compaction before **each** model request when token count exceeds the soft limit, including follow-up turns after tools.
- Use the hard limit only for `ErrBudgetExceeded` after compaction attempts.
- Emit verbose log events when compaction occurs (tokens before/after, summary length).

### Phase 4: Testing
- Unit tests for:
  - Summary prefix detection and filtering.
  - History rebuild order (instructions/env + recent user + summary).
  - Tool output retention policy.
  - No compaction when under soft limit.
- Integration test with a forced low compaction limit to verify summary insertion in prompts.

### Phase 5: Optional Enhancements
- Provider capability flags for "remote compaction" if a provider supports it.
- Persist compaction metadata (count, last summary size) in `results.json` and/or verbose logs.

## Files / Areas to Touch
- `internal/agent/compaction.go`
- `internal/agent/runner.go`
- `internal/agent/tokens.go`
- `internal/agent/openrouter.go` (or a new summarization helper)
- `internal/spec/types.go` and config scaffolding (if adding fields)
- `spec/engineering/builtin-agent.md`
- `spec/engineering/configuration.md`
- Agent/runner unit tests

## Risks
- Summary quality may omit critical tool output; mitigate by retaining recent tool outputs and making retention policy configurable.
- Token estimation is coarse; keep a safety buffer below the true model limit.
- Providers may emit tool calls during summarization; enforce tool-less summarization or treat tool calls as errors.

## Done Criteria
- Compaction produces a summary + recent context and keeps prompts within limits.
- Runs no longer fail from history growth when a summary can compress the context.
- Tests cover compaction behavior and summary insertion.

## Status
DONE (2026-01-08)
