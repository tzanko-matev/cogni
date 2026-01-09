package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/spec"
)

// TestValidateQuestionEvalRequiresQuestionsFile verifies questions_file is required.
func TestValidateQuestionEvalRequiresQuestionsFile(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = []spec.TaskConfig{{
		ID:    "question-task",
		Type:  "question_eval",
		Agent: "default",
	}}

	err := Validate(&cfg, t.TempDir())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "questions_file") {
		t.Fatalf("expected questions_file error, got %q", err.Error())
	}
}

// TestValidateQuestionEvalValidConfig verifies valid question_eval tasks pass.
func TestValidateQuestionEvalValidConfig(t *testing.T) {
	baseDir := t.TempDir()
	specPath := filepath.Join(baseDir, "questions.yml")
	payload := `version: 1
questions:
  - question: "What is 1+1?"
    answers: ["2"]
    correct_answers: ["2"]
`
	if err := os.WriteFile(specPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg := validConfig()
	cfg.Tasks = []spec.TaskConfig{{
		ID:            "question-task",
		Type:          "question_eval",
		Agent:         "default",
		QuestionsFile: "questions.yml",
	}}

	if err := Validate(&cfg, baseDir); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
