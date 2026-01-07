package agent

import (
	"testing"
)

func TestBuildInitialContext(t *testing.T) {
	ctx := TurnContext{
		CWD:                   "/repo",
		ApprovalPolicy:        "on-request",
		SandboxPolicy:         SandboxPolicy{Mode: "workspace-write", NetworkAccess: "enabled", WritableRoots: []string{"/tmp", "/work"}, Shell: "bash"},
		DeveloperInstructions: "dev instructions",
		UserInstructions:      "user instructions",
	}

	items := BuildInitialContext(ctx)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Role != "developer" || items[0].Content != (HistoryText{Text: "dev instructions"}) {
		t.Fatalf("unexpected developer item: %+v", items[0])
	}
	expectedUser := "# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>\nuser instructions\n</INSTRUCTIONS>"
	if items[1].Role != "user" || items[1].Content != (HistoryText{Text: expectedUser}) {
		t.Fatalf("unexpected user item: %+v", items[1])
	}
	expectedEnv := "<environment_context>\n" +
		"  <cwd>/repo</cwd>\n" +
		"  <approval_policy>on-request</approval_policy>\n" +
		"  <sandbox_mode>workspace-write</sandbox_mode>\n" +
		"  <network_access>enabled</network_access>\n" +
		"  <writable_roots>\n" +
		"    <root>/tmp</root>\n" +
		"    <root>/work</root>\n" +
		"  </writable_roots>\n" +
		"  <shell>bash</shell>\n" +
		"</environment_context>"
	if items[2].Role != "user" || items[2].Content != (HistoryText{Text: expectedEnv}) {
		t.Fatalf("unexpected environment item: %+v", items[2])
	}
}

func TestBuildPromptAddsApplyPatchInstructions(t *testing.T) {
	ctx := TurnContext{
		ModelFamily: ModelFamily{
			BaseInstructionsTemplate:           "base",
			NeedsSpecialApplyPatchInstructions: true,
			SupportsParallelToolCalls:          true,
		},
		Tools: []ToolDefinition{
			{Name: "list_files"},
		},
		Features: FeatureFlags{ParallelTools: true},
	}
	history := []HistoryItem{{Role: "user", Content: HistoryText{Text: "hello"}}}

	prompt := BuildPrompt(ctx, history)
	expected := "base\n" + ApplyPatchInstructions
	if prompt.Instructions != expected {
		t.Fatalf("unexpected instructions: %q", prompt.Instructions)
	}
	if !prompt.ParallelToolCalls {
		t.Fatalf("expected parallel tool calls enabled")
	}
	if len(prompt.InputItems) != 1 {
		t.Fatalf("expected history forwarded")
	}
}

func TestBuildPromptOverrideAndApplyPatchTool(t *testing.T) {
	ctx := TurnContext{
		ModelFamily: ModelFamily{
			BaseInstructionsTemplate:           "ignored",
			NeedsSpecialApplyPatchInstructions: true,
			SupportsParallelToolCalls:          false,
		},
		BaseInstructionsOverride: "override",
		Tools: []ToolDefinition{
			{Name: "apply_patch"},
		},
		Features: FeatureFlags{ParallelTools: true},
	}
	prompt := BuildPrompt(ctx, nil)
	if prompt.Instructions != "override" {
		t.Fatalf("unexpected instructions: %q", prompt.Instructions)
	}
	if prompt.ParallelToolCalls {
		t.Fatalf("expected parallel tool calls disabled")
	}
}
