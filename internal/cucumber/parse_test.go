package cucumber

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseFeatureExampleIDs verifies example ID generation from features.
func TestParseFeatureExampleIDs(t *testing.T) {
	feature := `Feature: Sample

  @id:with_tag
  Scenario Outline: Tagged with id column
    Given something
    Examples:
      | id | value |
      | e1 | a |
      | e2 | b |

  @id:tag_no_id
  Scenario Outline: Tagged no id column
    Given something
    Examples:
      | value |
      | a |
      | b |

  Scenario Outline: Name id column
    Given something
    Examples:
      | id | value |
      | row-1 | a |

  Scenario: Plain scenario
    Given something
`

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.feature")
	if err := os.WriteFile(path, []byte(feature), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	examples, err := ParseFeatureFile(path)
	if err != nil {
		t.Fatalf("parse feature: %v", err)
	}

	plainLine := findLine(feature, "Scenario: Plain scenario")
	expected := []string{
		"with_tag:e1",
		"with_tag:e2",
		"tag_no_id:1",
		"tag_no_id:2",
		"Name id column#row-1",
		filepath.ToSlash(path) + ":" + plainLine + ":1",
	}

	if len(examples) != len(expected) {
		t.Fatalf("expected %d examples, got %d", len(expected), len(examples))
	}
	for i, example := range examples {
		if example.ID != expected[i] {
			t.Fatalf("example %d: expected %q, got %q", i, expected[i], example.ID)
		}
	}
}

// findLine returns the 1-based line number containing a needle.
func findLine(body, needle string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.Contains(line, needle) {
			return itoa(i + 1)
		}
	}
	return "0"
}

// itoa formats a small integer without strconv for test assertions.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 8)
	for value > 0 {
		digits = append(digits, byte('0'+(value%10)))
		value /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
