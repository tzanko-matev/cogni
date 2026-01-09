package config

import "cogni/internal/spec"

// validConfig returns a minimal config used by validation tests.
func validConfig() spec.Config {
	return spec.Config{
		Version: 1,
		Repo: spec.RepoConfig{
			OutputDir: "./out",
		},
		Agents: []spec.AgentConfig{
			{
				ID:       "default",
				Type:     "builtin",
				Provider: "openrouter",
				Model:    "gpt-4.1-mini",
			},
		},
		DefaultAgent: "default",
		Tasks: []spec.TaskConfig{
			{
				ID:     "task1",
				Type:   "qa",
				Agent:  "default",
				Prompt: "hello",
			},
		},
	}
}
