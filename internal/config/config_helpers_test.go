package config

import (
	"os"
	"path/filepath"
	"testing"

	"cogni/internal/spec"
)

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
				ID:            "task1",
				Type:          "question_eval",
				Agent:         "default",
				QuestionsFile: "questions.yml",
			},
		},
	}
}

func writeQuestionSpec(t *testing.T, dir string) {
	t.Helper()
	payload := `version: 1
questions:
  - question: "What is 1+1?"
    answers: ["2"]
    correct_answers: ["2"]
`
	path := filepath.Join(dir, "questions.yml")
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write questions file: %v", err)
	}
}
