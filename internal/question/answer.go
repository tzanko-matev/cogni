package question

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// ParsedAnswer captures the raw and normalized answer text.
type ParsedAnswer struct {
	Raw        string
	Normalized string
}

// ErrMissingAnswer indicates that no <answer>...</answer> block was found.
var ErrMissingAnswer = errors.New("missing <answer> block")

// ErrTrailingText indicates there was content after the closing </answer> tag.
var ErrTrailingText = errors.New("trailing content after </answer>")

// ErrEmptyAnswer indicates that the parsed answer was empty.
var ErrEmptyAnswer = errors.New("empty answer text")

// ExtractTrailingAnswerXML returns the trailing <answer> XML fragment from output.
func ExtractTrailingAnswerXML(output string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "", ErrMissingAnswer
	}
	end := strings.LastIndex(trimmed, "</answer>")
	if end == -1 {
		return "", ErrMissingAnswer
	}
	if end != len(trimmed)-len("</answer>") {
		return "", ErrTrailingText
	}
	start := strings.LastIndex(trimmed, "<answer>")
	if start == -1 || start > end {
		return "", ErrMissingAnswer
	}
	return trimmed[start:], nil
}

// ParseAnswerXML parses an <answer> XML fragment into a ParsedAnswer.
func ParseAnswerXML(fragment string) (ParsedAnswer, error) {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return ParsedAnswer{}, ErrMissingAnswer
	}
	var payload struct {
		XMLName xml.Name `xml:"answer"`
		Text    string   `xml:",chardata"`
	}
	if err := xml.Unmarshal([]byte(fragment), &payload); err != nil {
		return ParsedAnswer{}, fmt.Errorf("parse answer xml: %w", err)
	}
	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return ParsedAnswer{}, ErrEmptyAnswer
	}
	return ParsedAnswer{Raw: text, Normalized: NormalizeAnswerText(text)}, nil
}

// ParseAnswerFromOutput extracts and parses a trailing <answer> block from output.
func ParseAnswerFromOutput(output string) (ParsedAnswer, error) {
	fragment, err := ExtractTrailingAnswerXML(output)
	if err != nil {
		return ParsedAnswer{}, err
	}
	return ParseAnswerXML(fragment)
}
