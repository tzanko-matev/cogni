package agent

import (
	"fmt"
	"strings"
)

// ApplyPatchInstructions is appended when a model family requires explicit patch guidance.
const ApplyPatchInstructions = "When editing files, use the apply_patch tool."

// SessionConfig captures the inputs needed to start an agent session.
type SessionConfig struct {
	Model                    string
	ModelOverride            string
	Provider                 string
	ApprovalPolicy           string
	SandboxPolicy            SandboxPolicy
	CWD                      string
	DeveloperInstructions    string
	UserInstructions         string
	BaseInstructionsOverride string
	OutputSchema             string
	Features                 FeatureFlags
	ToolConfig               ToolConfig
	Verbose                  bool
	AuthMode                 string
}

// SandboxPolicy describes sandbox execution constraints for the agent.
type SandboxPolicy struct {
	Mode          string
	NetworkAccess string
	WritableRoots []string
	Shell         string
}

// FeatureFlags toggles optional agent behaviors.
type FeatureFlags struct {
	ParallelTools bool
	SkillsEnabled bool
}

// ToolConfig declares the tools available to the agent.
type ToolConfig struct {
	Tools []ToolDefinition
}

// ToolDefinition describes a callable tool exposed to the agent.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  *ToolSchema
}

// ModelFamily captures model-specific behavior and prompt templates.
type ModelFamily struct {
	BaseInstructionsTemplate           string
	NeedsSpecialApplyPatchInstructions bool
	SupportsParallelToolCalls          bool
}

// TurnContext holds contextual metadata used to build prompts.
type TurnContext struct {
	Model                    string
	ModelFamily              ModelFamily
	Tools                    []ToolDefinition
	ApprovalPolicy           string
	SandboxPolicy            SandboxPolicy
	CWD                      string
	DeveloperInstructions    string
	UserInstructions         string
	BaseInstructionsOverride string
	OutputSchema             string
	Features                 FeatureFlags
	Verbose                  bool
}

// HistoryItem captures a single turn item with a role and typed content.
type HistoryItem struct {
	Role    string
	Content HistoryContent
}

// Prompt is the fully assembled request sent to a provider.
type Prompt struct {
	Instructions      string
	InputItems        []HistoryItem
	Tools             []ToolDefinition
	ParallelToolCalls bool
	OutputSchema      string
}

// Session tracks conversation history and context for a run.
type Session struct {
	Ctx     TurnContext
	History []HistoryItem
}

// ModelFamilyLoader resolves model family metadata for a provider and model.
type ModelFamilyLoader interface {
	Load(provider, model string) (ModelFamily, error)
}

// StartSession initializes a session using the provided configuration.
func StartSession(config SessionConfig, loader ModelFamilyLoader) (*Session, error) {
	model := strings.TrimSpace(config.Model)
	if config.ModelOverride != "" {
		model = strings.TrimSpace(config.ModelOverride)
	}
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if loader == nil {
		return nil, fmt.Errorf("model family loader is required")
	}
	family, err := loader.Load(config.Provider, model)
	if err != nil {
		return nil, err
	}
	ctx := TurnContext{
		Model:                    model,
		ModelFamily:              family,
		Tools:                    config.ToolConfig.Tools,
		ApprovalPolicy:           config.ApprovalPolicy,
		SandboxPolicy:            config.SandboxPolicy,
		CWD:                      config.CWD,
		DeveloperInstructions:    config.DeveloperInstructions,
		UserInstructions:         config.UserInstructions,
		BaseInstructionsOverride: config.BaseInstructionsOverride,
		OutputSchema:             config.OutputSchema,
		Features:                 config.Features,
		Verbose:                  config.Verbose,
	}
	history := BuildInitialContext(ctx)
	return &Session{
		Ctx:     ctx,
		History: history,
	}, nil
}

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
