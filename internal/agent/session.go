package agent

import (
	"fmt"
	"strings"
)

const ApplyPatchInstructions = "When editing files, use the apply_patch tool."

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
	AuthMode                 string
}

type SandboxPolicy struct {
	Mode          string
	NetworkAccess string
	WritableRoots []string
	Shell         string
}

type FeatureFlags struct {
	ParallelTools bool
	SkillsEnabled bool
}

type ToolConfig struct {
	Tools []ToolDefinition
}

type ToolDefinition struct {
	Name        string
	Description string
}

type ModelFamily struct {
	BaseInstructionsTemplate           string
	NeedsSpecialApplyPatchInstructions bool
	SupportsParallelToolCalls          bool
}

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
}

type HistoryItem struct {
	Role    string
	Content any
}

type Prompt struct {
	Instructions      string
	InputItems        []HistoryItem
	Tools             []ToolDefinition
	ParallelToolCalls bool
	OutputSchema      string
}

type Session struct {
	Ctx     TurnContext
	History []HistoryItem
}

type ModelFamilyLoader interface {
	Load(provider, model string) (ModelFamily, error)
}

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
	}
	history := BuildInitialContext(ctx)
	return &Session{
		Ctx:     ctx,
		History: history,
	}, nil
}

func BuildInitialContext(ctx TurnContext) []HistoryItem {
	items := make([]HistoryItem, 0, 3)
	if strings.TrimSpace(ctx.DeveloperInstructions) != "" {
		items = append(items, HistoryItem{
			Role:    "developer",
			Content: ctx.DeveloperInstructions,
		})
	}
	if strings.TrimSpace(ctx.UserInstructions) != "" {
		items = append(items, HistoryItem{
			Role:    "user",
			Content: formatUserInstructions(ctx.CWD, ctx.UserInstructions),
		})
	}
	items = append(items, HistoryItem{
		Role:    "user",
		Content: formatEnvironmentContext(ctx),
	})
	return items
}

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

func formatUserInstructions(cwd, instructions string) string {
	return fmt.Sprintf("# AGENTS.md instructions for %s\n\n<INSTRUCTIONS>\n%s\n</INSTRUCTIONS>", cwd, instructions)
}

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

func hasTool(tools []ToolDefinition, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
