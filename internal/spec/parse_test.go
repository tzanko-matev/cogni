package spec

import "testing"

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
    type: qa
    agent: default
    prompt: "hello"
`)
	if _, err := ParseConfig(data); err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
}

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

func TestParseConfigRejectsMultipleDocs(t *testing.T) {
	data := []byte("version: 1\n---\nversion: 1\n")
	if _, err := ParseConfig(data); err == nil {
		t.Fatalf("expected parse error for multiple documents")
	}
}
