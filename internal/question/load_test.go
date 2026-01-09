package question

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadSpecYAML verifies YAML specs load and normalize properly.
func TestLoadSpecYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "questions.yml")
	payload := `version: 1
questions:
  - id: q1
    question: "  What is 2+2? "
    answers: [" 4 ", "5"]
    correct_answers: ["4"]
`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if spec.Version != 1 {
		t.Fatalf("expected version 1, got %d", spec.Version)
	}
	if len(spec.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(spec.Questions))
	}
	q := spec.Questions[0]
	if q.ID != "q1" {
		t.Fatalf("expected id q1, got %q", q.ID)
	}
	if q.Prompt != "What is 2+2?" {
		t.Fatalf("expected trimmed prompt, got %q", q.Prompt)
	}
	if len(q.Answers) != 2 || q.Answers[0] != "4" {
		t.Fatalf("unexpected answers: %+v", q.Answers)
	}
}

// TestLoadSpecJSON verifies JSON specs are parsed and validated.
func TestLoadSpecJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "questions.json")
	payload := `{
  "version": 1,
  "questions": [
    {
      "id": "q2",
      "question": "Which color?",
      "answers": ["red", "blue"],
      "correct_answers": ["blue"]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if len(spec.Questions) != 1 || spec.Questions[0].ID != "q2" {
		t.Fatalf("unexpected spec: %+v", spec.Questions)
	}
}

// TestLoadSpecValidationErrors verifies invalid specs return validation errors.
func TestLoadSpecValidationErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "questions.yml")
	payload := `version: 1
questions:
  - id: dup
    question: "Q1"
    answers: ["yes", "no"]
    correct_answers: ["maybe"]
  - id: dup
    question: "Q2"
    answers: ["a"]
    correct_answers: ["a"]
`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	_, err := LoadSpec(path)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if len(validationErr.Issues) == 0 {
		t.Fatalf("expected issues to be populated")
	}
}
