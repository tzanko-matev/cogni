package agent

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
