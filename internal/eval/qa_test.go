package eval

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEvaluateQAInvalidJSON verifies invalid JSON handling.
func TestEvaluateQAInvalidJSON(t *testing.T) {
	result := EvaluateQA("not json", QAConfig{})
	if result.Status != "fail" || result.FailureReason != "invalid_json" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// TestEvaluateQASchemaValidation verifies schema validation outcomes.
func TestEvaluateQASchemaValidation(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	schema := `{
  "type": "object",
  "required": ["name"],
  "properties": {
    "name": { "type": "string" }
  }
}`
	if err := os.WriteFile(schemaPath, []byte(schema), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	pass := EvaluateQA(`{"name":"ok"}`, QAConfig{JSONSchemaPath: schemaPath})
	if pass.Status != "pass" {
		t.Fatalf("expected pass, got %+v", pass)
	}

	fail := EvaluateQA(`{"name":1}`, QAConfig{JSONSchemaPath: schemaPath})
	if fail.Status != "fail" || fail.FailureReason != "schema_validation_failed" {
		t.Fatalf("expected schema failure, got %+v", fail)
	}
}

// TestEvaluateQAMustContain verifies required tokens enforcement.
func TestEvaluateQAMustContain(t *testing.T) {
	result := EvaluateQA(`{"foo":"bar"}`, QAConfig{
		MustContain: []string{"citations"},
	})
	if result.Status != "fail" || result.FailureReason != "must_contain_failed" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Artifacts.MustContainMissing) != 1 {
		t.Fatalf("expected missing list, got %+v", result.Artifacts.MustContainMissing)
	}
}

// TestEvaluateQACitations verifies citation validation behavior.
func TestEvaluateQACitations(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	output := `{"citations":[{"path":"file.txt","lines":[1,2]}]}`
	result := EvaluateQA(output, QAConfig{
		ValidateCitations: true,
		RepoRoot:          dir,
	})
	if result.Status != "pass" {
		t.Fatalf("expected pass, got %+v", result)
	}

	badOutput := `{"citations":[{"path":"file.txt","lines":[5,6]}]}`
	fail := EvaluateQA(badOutput, QAConfig{
		ValidateCitations: true,
		RepoRoot:          dir,
	})
	if fail.Status != "fail" || fail.FailureReason != "citation_validation_failed" {
		t.Fatalf("expected citation failure, got %+v", fail)
	}
}
