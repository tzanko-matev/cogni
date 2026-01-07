package cucumber

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNormalizeGodogResults verifies normalization of godog output.
func TestNormalizeGodogResults(t *testing.T) {
	feature := `Feature: Sample

  @id:with_tag
  Scenario Outline: Tagged with id column
    Given something
    Examples:
      | id | value |
      | e1 | a |
`
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.feature")
	if err := os.WriteFile(path, []byte(feature), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	index, err := BuildExampleIndex("", []string{path})
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	examples, err := ParseFeatureFile(path)
	if err != nil {
		t.Fatalf("parse feature: %v", err)
	}
	if len(examples) == 0 {
		t.Fatalf("expected examples")
	}

	features := []CukeFeatureJSON{{
		URI: path,
		Elements: []CukeElement{{
			Name: "Tagged with id column",
			Line: examples[0].ExampleLine,
			Steps: []CukeStep{{
				Result: CukeResult{Status: "passed"},
			}},
		}},
	}}

	results, err := NormalizeGodogResults("", features, index)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ExampleID != examples[0].ID {
		t.Fatalf("expected example %q, got %q", examples[0].ID, results[0].ExampleID)
	}
	if results[0].Status != "passed" {
		t.Fatalf("expected passed, got %q", results[0].Status)
	}
}

// TestParseGodogJSONStripsWarnings verifies warning prefixes are removed.
func TestParseGodogJSONStripsWarnings(t *testing.T) {
	payload := "\x1b[33mUse of godog CLI is deprecated\x1b[0m\n" +
		"\x1b[33mSee https://example.test\x1b[0m\n" +
		`[{"uri":"sample.feature","elements":[]}]`
	features, err := ParseGodogJSON([]byte(payload))
	if err != nil {
		t.Fatalf("parse godog json: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}
	if features[0].URI != "sample.feature" {
		t.Fatalf("unexpected uri %q", features[0].URI)
	}
}
