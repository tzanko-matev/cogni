package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolCallArgs holds decoded JSON arguments for a tool call.
type ToolCallArgs map[string]json.RawMessage

// RequiredString returns a required string argument.
func (args ToolCallArgs) RequiredString(key string) (string, error) {
	value, ok, err := args.OptionalString(key)
	if err != nil {
		return "", err
	}
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

// OptionalString returns an optional string argument with a presence flag.
func (args ToolCallArgs) OptionalString(key string) (string, bool, error) {
	raw, ok := args[key]
	if !ok {
		return "", false, nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", false, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false, fmt.Errorf("%s must be a string", key)
	}
	return strings.TrimSpace(value), true, nil
}

// OptionalStringSlice returns an optional string slice argument.
func (args ToolCallArgs) OptionalStringSlice(key string) ([]string, error) {
	raw, ok := args[key]
	if !ok {
		return nil, nil
	}
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a list of strings", key)
	}
	return value, nil
}

// OptionalInt returns an optional integer argument.
func (args ToolCallArgs) OptionalInt(key string) (*int, error) {
	raw, ok := args[key]
	if !ok {
		return nil, nil
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be an integer", key)
	}
	return &value, nil
}
