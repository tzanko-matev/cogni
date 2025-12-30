# Built-in Agent

This document defines the built-in agent behavior used by `cogni run`. It is
the implementation target for `internal/agent` and related packages.

## Scope

- Implement only the built-in agent described here.
- External agent adapters are out of scope for this document.

## Where the code lives

- `internal/agent`: session lifecycle, prompt building, streaming, and tool loop.
- `internal/tools`: tool registry and tool execution.
- `internal/metrics`: token/tool timing and counts.
- `internal/config` and `internal/spec`: configuration and agent/task parsing.

## Core data structures

Use simple structs or records that mirror the fields below. Keep the naming
stable so it is easy to map the spec to code.

### SessionConfig

Inputs used to start a session. Fields:

- `model_override`: optional string; if set, prefer it over config model.
- `provider`: provider id, for example `openrouter`.
- `approval_policy`: how tool calls are approved.
- `sandbox_policy`: sandbox settings, including:
  - `mode` (read-only, workspace-write, danger-full-access)
  - `network_access`
  - `writable_roots`
  - `shell`
- `cwd`: workspace root for tools and instruction wrapping.
- `developer_instructions`: string for a developer message.
- `user_instructions`: string read from project docs or config.
- `base_instructions_override`: optional; if present, replace model family template.
- `output_schema`: optional schema for providers that support it.
- `features`: flags such as `parallel_tools` and `skills_enabled`.
- `verbose`: boolean; when true, emit detailed console logs (LLM input/output, tool calls and results, metrics).
- `tool_config`: tool list and tool settings.
- `auth_mode`: auth strategy for provider credentials.

### ModelFamily

Defines model-specific defaults and capabilities:

- `base_instructions_template`: default system/developer instructions.
- `needs_special_apply_patch_instructions`: boolean.
- `supports_parallel_tool_calls`: boolean.

### TurnContext

Runtime context created from the config:

- `model`, `model_family`, `tools`
- `approval_policy`, `sandbox_policy`, `cwd`
- `developer_instructions`, `user_instructions`
- `base_instructions_override`, `output_schema`
- `verbose`

### Prompt

Payload sent to the model:

- `instructions`: base instructions (template or override).
- `input_items`: history items (messages + tool calls + tool outputs).
- `tools`: tool definitions.
- `parallel_tool_calls`: boolean flag.
- `output_schema`: optional.

### HistoryItem

Typed items that make up the conversation:

- `role`: user, developer, assistant, or tool.
- `content`: text or structured tool payload.

## Session initialization

Create a single entry point like `start_session(config)` and keep it pure:

1. Select the model (respect `model_override`).
2. Load the model family for the selected model and provider.
3. Build tool definitions from `tool_config`.
4. Build a `TurnContext`.
5. Build the initial history by calling `build_initial_context(ctx)`.

Pseudocode:

```text
start_session(config):
  model = select_model(config.model_override, config.auth_mode)
  model_family = load_model_family(model, config.provider)
  ctx = TurnContext(...fields...)
  history = []
  history += build_initial_context(ctx)
  return Session(ctx, history)
```

### Initial context contents

`build_initial_context(ctx)` should add messages in this order:

1. Developer instructions (if present).
2. User instructions (if present), wrapped in the AGENTS format:

```text
# AGENTS.md instructions for <cwd>

<INSTRUCTIONS>
...user instructions...
</INSTRUCTIONS>
```

3. The environment context block:

```text
<environment_context>
  <cwd>...</cwd>
  <approval_policy>...</approval_policy>
  <sandbox_mode>...</sandbox_mode>
  <network_access>...</network_access>
  <writable_roots>...</writable_roots>
  <shell>...</shell>
</environment_context>
```

## Turn loop

Implement a REPL loop similar to `on_user_input(session, user_text)`:

1. Append the user message to history.
2. If `skills_enabled` is true, append skill injections from the user text.
3. If token usage exceeds the compaction limit, compact history.
4. Enter a loop:
   - Drain any pending inputs and append to history.
   - Build a prompt from the current context and history.
   - Stream a model response.
   - Handle the stream; if tools were called, run them and loop again.
   - Exit when no tool calls were produced.

The key rule: every tool call triggers another model request after tool outputs
are added to history.

## Prompt construction

Implement `build_prompt(ctx, history)` with these steps:

1. Choose `instructions`:
   - Use `ctx.base_instructions_override` if set.
   - Otherwise use `ctx.model_family.base_instructions_template`.
2. If the model family needs special apply_patch instructions and the
   `apply_patch` tool is not included, append the extra instructions.
3. Format history into `input_items`.
   - If `apply_patch` is a freeform tool, normalize shell outputs to avoid
     confusing the model.
4. Set `parallel_tool_calls` if both conditions are true:
   - Model family supports parallel tool calls.
   - `features.parallel_tools` is enabled.
5. Return the prompt object.

## Streaming and tool handling

### Model request dispatch

Build request payloads as structured objects, not a single concatenated string.
Use provider-specific fields:

- If the provider uses a Responses-style API, include `output_schema` and
  `reasoning` settings in the request.
- Otherwise, omit those fields.

### Response stream handling

Implement `handle_response_stream(session, stream)`:

1. For assistant messages, append to history as a message item.
2. For tool calls:
   - Execute the tool immediately.
   - Append the tool output to history.
   - Mark `needs_follow_up = true`.
3. Return `needs_follow_up`. The turn loop should continue if true.

Always append tool outputs, even on error. The model must see the error output
to decide how to proceed.

## Verbose logging

When `verbose` is true, emit console logs for:

- Model request inputs and response outputs.
- Tool calls and tool outputs (subject to truncation limits).
- Per-task metrics updates and the final metrics snapshot.

Verbose logging is diagnostic only and must not change task execution or outputs.

## Environment updates mid-session

If environment settings change after session start, append a *diff* message
instead of repeating the full environment block. Example:

```text
<environment_diff>
  <cwd>...new cwd...</cwd>
</environment_diff>
```

## History compaction

Compaction is allowed only when token usage exceeds the limit. The compacted
history must preserve:

- Developer instructions (if any).
- User instructions (if any).
- The most recent environment block or diff.
- The latest user request.
- Tool call inputs and outputs that affect state.

## Error handling and validation

Add explicit error handling for:

- Unknown model family for the selected model/provider.
- Missing tool definitions referenced in tool calls.
- Tool execution errors (returned as tool outputs).
- Stream failures (surface a clear error and stop the loop).
- Invalid config fields (fail fast during session init).

## Implementation checklist

Use this as a step-by-step guide:

1. Define the core data structures.
2. Implement `start_session` and `build_initial_context`.
3. Implement `build_prompt` with the apply_patch rule.
4. Implement the REPL loop with tool follow-ups.
5. Add stream handling that appends messages and tool outputs.
6. Wire tool execution through `internal/tools`.
7. Add compaction logic and env diff handling.
8. Add metrics hooks for tokens, tool calls, and wall time.

## Testing expectations

At minimum, add:

- Unit tests for `build_initial_context`, `build_prompt`, and history compaction.
- A fake streaming provider that emits: assistant message -> tool call -> tool
  output -> assistant message.
- An integration test that runs a small task end-to-end and produces tool calls.
