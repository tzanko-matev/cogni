package agent

import (
	"fmt"
	"strings"
)

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
