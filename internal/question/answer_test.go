package question

import (
	"errors"
	"testing"
)

// TestParseAnswerFromOutput verifies trailing answer extraction and parsing.
func TestParseAnswerFromOutput(t *testing.T) {
	output := "Reasoning goes here.\n<answer>Blue</answer>"
	answer, err := ParseAnswerFromOutput(output)
	if err != nil {
		t.Fatalf("parse answer: %v", err)
	}
	if answer.Raw != "Blue" {
		t.Fatalf("expected raw answer Blue, got %q", answer.Raw)
	}
	if answer.Normalized != "blue" {
		t.Fatalf("expected normalized answer blue, got %q", answer.Normalized)
	}
}

// TestExtractTrailingAnswerXMLTrailingText verifies trailing text is rejected.
func TestExtractTrailingAnswerXMLTrailingText(t *testing.T) {
	_, err := ExtractTrailingAnswerXML("before <answer>ok</answer> extra")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrTrailingText) {
		t.Fatalf("expected trailing text error, got %v", err)
	}
}

// TestParseAnswerXMLRejectsWrongRoot verifies invalid root tags are rejected.
func TestParseAnswerXMLRejectsWrongRoot(t *testing.T) {
	_, err := ParseAnswerXML("<answers>nope</answers>")
	if err == nil {
		t.Fatalf("expected error")
	}
}
