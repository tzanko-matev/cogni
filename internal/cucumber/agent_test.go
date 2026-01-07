package cucumber

import (
	"errors"
	"reflect"
	"testing"
)

// TestValidateAgentBatchResponseValid verifies valid batch responses pass.
func TestValidateAgentBatchResponseValid(t *testing.T) {
	output := `{"results":[{"example_id":"alpha:1","implemented":true},{"example_id":"beta:1","implemented":false}]}`
	response, err := ParseAgentBatchResponse(output)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	results, err := ValidateAgentBatchResponse([]string{"alpha:1", "beta:1"}, response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results["alpha:1"].Implemented {
		t.Fatalf("expected alpha:1 implemented")
	}
	if results["beta:1"].Implemented {
		t.Fatalf("expected beta:1 not implemented")
	}
}

// TestValidateAgentBatchResponseMissing verifies missing example IDs are flagged.
func TestValidateAgentBatchResponseMissing(t *testing.T) {
	output := `{"results":[{"example_id":"alpha:1","implemented":true}]}`
	response, err := ParseAgentBatchResponse(output)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	_, err = ValidateAgentBatchResponse([]string{"alpha:1", "beta:1"}, response)
	if err == nil {
		t.Fatalf("expected error")
	}
	var batchErr BatchValidationError
	if !errors.As(err, &batchErr) {
		t.Fatalf("expected batch validation error, got %T", err)
	}
	if !reflect.DeepEqual(batchErr.Missing, []string{"beta:1"}) {
		t.Fatalf("unexpected missing: %#v", batchErr.Missing)
	}
}

// TestValidateAgentBatchResponseExtra verifies extra example IDs are flagged.
func TestValidateAgentBatchResponseExtra(t *testing.T) {
	output := `{"results":[{"example_id":"alpha:1","implemented":true},{"example_id":"beta:1","implemented":false}]}`
	response, err := ParseAgentBatchResponse(output)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	_, err = ValidateAgentBatchResponse([]string{"alpha:1"}, response)
	if err == nil {
		t.Fatalf("expected error")
	}
	var batchErr BatchValidationError
	if !errors.As(err, &batchErr) {
		t.Fatalf("expected batch validation error, got %T", err)
	}
	if !reflect.DeepEqual(batchErr.Extra, []string{"beta:1"}) {
		t.Fatalf("unexpected extra: %#v", batchErr.Extra)
	}
}

// TestValidateAgentBatchResponseDuplicate verifies duplicate IDs are flagged.
func TestValidateAgentBatchResponseDuplicate(t *testing.T) {
	output := `{"results":[{"example_id":"alpha:1","implemented":true},{"example_id":"alpha:1","implemented":false}]}`
	response, err := ParseAgentBatchResponse(output)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	_, err = ValidateAgentBatchResponse([]string{"alpha:1"}, response)
	if err == nil {
		t.Fatalf("expected error")
	}
	var batchErr BatchValidationError
	if !errors.As(err, &batchErr) {
		t.Fatalf("expected batch validation error, got %T", err)
	}
	if !reflect.DeepEqual(batchErr.Duplicate, []string{"alpha:1"}) {
		t.Fatalf("unexpected duplicate: %#v", batchErr.Duplicate)
	}
}

// TestValidateAgentBatchResponseEmptyID verifies empty IDs are rejected.
func TestValidateAgentBatchResponseEmptyID(t *testing.T) {
	output := `{"results":[{"example_id":" ","implemented":true}]}`
	response, err := ParseAgentBatchResponse(output)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	_, err = ValidateAgentBatchResponse([]string{"alpha:1"}, response)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != "example_id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}
