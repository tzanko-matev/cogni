package agent

import (
	"fmt"
	"strings"
)

// BuildInitialContext builds the initial history items for a session.
func BuildInitialContext(ctx TurnContext) []HistoryItem {
	items := make([]HistoryItem, 0, 3)
	if strings.TrimSpace(ctx.DeveloperInstructions) != "" {
		items = append(items, HistoryItem{
			Role:    "developer",
			Content: HistoryText{Text: ctx.DeveloperInstructions},
		})
	}
	if strings.TrimSpace(ctx.UserInstructions) != "" {
		items = append(items, HistoryItem{
			Role:    "user",
			Content: HistoryText{Text: formatUserInstructions(ctx.CWD, ctx.UserInstructions)},
		})
	}
	items = append(items, HistoryItem{
		Role:    "user",
		Content: HistoryText{Text: formatEnvironmentContext(ctx)},
	})
	return items
}

// BuildPrompt assembles a provider prompt from context and history.
func BuildPrompt(ctx TurnContext, history []HistoryItem) Prompt {
	instructions := strings.TrimSpace(ctx.BaseInstructionsOverride)
	if instructions == "" {
		instructions = ctx.ModelFamily.BaseInstructionsTemplate
	}
	if ctx.ModelFamily.NeedsSpecialApplyPatchInstructions && !hasTool(ctx.Tools, "apply_patch") {
		if instructions != "" {
			instructions += "\n"
		}
		instructions += ApplyPatchInstructions
	}
	return Prompt{
		Instructions:      instructions,
		InputItems:        history,
		Tools:             ctx.Tools,
		ParallelToolCalls: ctx.ModelFamily.SupportsParallelToolCalls && ctx.Features.ParallelTools,
		OutputSchema:      ctx.OutputSchema,
	}
}

// formatUserInstructions wraps AGENTS.md instructions for the model.
func formatUserInstructions(cwd, instructions string) string {
	return fmt.Sprintf("# AGENTS.md instructions for %s\n\n<INSTRUCTIONS>\n%s\n</INSTRUCTIONS>", cwd, instructions)
}

// formatEnvironmentContext renders the environment metadata block.
func formatEnvironmentContext(ctx TurnContext) string {
	var builder strings.Builder
	builder.WriteString("<environment_context>\n")
	builder.WriteString(fmt.Sprintf("  <cwd>%s</cwd>\n", ctx.CWD))
	builder.WriteString(fmt.Sprintf("  <approval_policy>%s</approval_policy>\n", ctx.ApprovalPolicy))
	builder.WriteString(fmt.Sprintf("  <sandbox_mode>%s</sandbox_mode>\n", ctx.SandboxPolicy.Mode))
	builder.WriteString(fmt.Sprintf("  <network_access>%s</network_access>\n", ctx.SandboxPolicy.NetworkAccess))
	builder.WriteString("  <writable_roots>\n")
	for _, root := range ctx.SandboxPolicy.WritableRoots {
		builder.WriteString(fmt.Sprintf("    <root>%s</root>\n", root))
	}
	builder.WriteString("  </writable_roots>\n")
	builder.WriteString(fmt.Sprintf("  <shell>%s</shell>\n", ctx.SandboxPolicy.Shell))
	builder.WriteString("</environment_context>")
	return builder.String()
}

// hasTool reports whether a named tool is present.
func hasTool(tools []ToolDefinition, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
