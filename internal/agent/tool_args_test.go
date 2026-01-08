package agent

import (
	"encoding/json"
	"testing"
)

// TestOptionalStringTreatsNullAsAbsent ensures optional string args ignore explicit nulls.
func TestOptionalStringTreatsNullAsAbsent(t *testing.T) {
	args := ToolCallArgs{
		"glob": json.RawMessage("null"),
	}
	value, ok, err := args.OptionalString("glob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected null to be treated as absent")
	}
	if value != "" {
		t.Fatalf("expected empty value, got %q", value)
	}
}
