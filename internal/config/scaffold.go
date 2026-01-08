package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultSchema is the starter schema created by Scaffold.
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

// Scaffold writes a starter config and schema into a repo.
func Scaffold(specPath, outputDir string) error {
	if specPath == "" {
		return fmt.Errorf("spec path is required")
	}
	dir, err := sanitizeOutputDir(outputDir)
	if err != nil {
		return err
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

	configBody, err := renderScaffoldConfig(dir)
	if err != nil {
		return fmt.Errorf("render scaffold config: %w", err)
	}
	if err := os.WriteFile(specPath, []byte(configBody), 0o644); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}
	if err := os.WriteFile(schemaPath, []byte(defaultSchema), 0o644); err != nil {
		return fmt.Errorf("write schema file: %w", err)
	}
	return nil
}

// sanitizeOutputDir prepares the output directory value for YAML output.
func sanitizeOutputDir(outputDir string) (string, error) {
	dir := strings.TrimSpace(outputDir)
	if dir == "" {
		dir = DefaultOutputDir
	}
	if strings.Contains(dir, "\n") {
		return "", fmt.Errorf("output dir must be a single line")
	}
	return strings.ReplaceAll(dir, "\"", "\\\""), nil
}
