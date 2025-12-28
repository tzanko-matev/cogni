package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultConfig = `version: 1
repo:
  output_dir: "./cogni-results"
  setup_commands:
    - "go mod download"

agents:
  - id: default
    type: builtin
    provider: "openrouter"
    model: "gpt-4.1-mini"
    max_steps: 25
    temperature: 0.0

default_agent: "default"

tasks:
  - id: auth_flow_summary
    type: qa
    agent: "default"
    prompt: >
      Explain how authorization is enforced for API requests.
      Return JSON with keys:
      {"entrypoints":[...],"middleware":[...],"checks":[...],"citations":[{"path":...,"lines":[start,end]}]}
    eval:
      json_schema: "schemas/auth_flow_summary.schema.json"
      must_contain_strings:
        - "middleware"
        - "citations"
      validate_citations: true
    budget:
      max_tokens: 12000
      max_seconds: 120
`

const defaultSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["entrypoints", "middleware", "checks", "citations"],
  "properties": {
    "entrypoints": { "type": "array", "items": { "type": "string" } },
    "middleware": { "type": "array", "items": { "type": "string" } },
    "checks": { "type": "array", "items": { "type": "string" } },
    "citations": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["path", "lines"],
        "properties": {
          "path": { "type": "string" },
          "lines": {
            "type": "array",
            "minItems": 2,
            "maxItems": 2,
            "items": { "type": "integer" }
          }
        }
      }
    }
  }
}
`

func Scaffold(specPath string) error {
	if specPath == "" {
		return fmt.Errorf("spec path is required")
	}
	if info, err := os.Stat(specPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("spec path %q is a directory", specPath)
		}
		return fmt.Errorf("spec file already exists at %q", specPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat spec file: %w", err)
	}

	baseDir := filepath.Dir(specPath)
	schemasDir := filepath.Join(baseDir, "schemas")
	if err := os.MkdirAll(schemasDir, 0o755); err != nil {
		return fmt.Errorf("create schemas dir: %w", err)
	}

	schemaPath := filepath.Join(schemasDir, "auth_flow_summary.schema.json")
	if info, err := os.Stat(schemaPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("schema path %q is a directory", schemaPath)
		}
		return fmt.Errorf("schema file already exists at %q", schemaPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat schema file: %w", err)
	}

	if err := os.WriteFile(specPath, []byte(defaultConfig), 0o644); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}
	if err := os.WriteFile(schemaPath, []byte(defaultSchema), 0o644); err != nil {
		return fmt.Errorf("write schema file: %w", err)
	}
	return nil
}
