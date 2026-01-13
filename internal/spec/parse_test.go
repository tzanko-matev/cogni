package spec

import "testing"

// TestParseConfigValid verifies valid config parsing succeeds.
func TestParseConfigValid(t *testing.T) {
	data := []byte(`version: 1
repo:
  output_dir: "./out"
agents:
  - id: default
    type: builtin
    provider: openrouter
    model: gpt-4.1-mini
default_agent: default
tasks:
  - id: task1
    type: question_eval
    agent: default
    questions_file: "questions.yml"
`)
	if _, err := ParseConfig(data); err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
}

// TestParseConfigUnknownField verifies unknown fields are rejected.
func TestParseConfigUnknownField(t *testing.T) {
	data := []byte(`version: 1
repo:
  output_dir: "./out"
unknown: true
`)
	if _, err := ParseConfig(data); err == nil {
		t.Fatalf("expected parse error for unknown field")
	}
}

// TestParseConfigRejectsMultipleDocs verifies multiple YAML docs are rejected.
func TestParseConfigRejectsMultipleDocs(t *testing.T) {
	data := []byte("version: 1\n---\nversion: 1\n")
	if _, err := ParseConfig(data); err == nil {
		t.Fatalf("expected parse error for multiple documents")
	}
}
