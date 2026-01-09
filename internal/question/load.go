package question

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadSpec reads, parses, and validates a question specification file.
func LoadSpec(path string) (Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read question spec: %w", err)
	}
	spec, err := parseSpec(data, path)
	if err != nil {
		return Spec{}, err
	}
	normalized, err := NormalizeSpec(spec)
	if err != nil {
		return Spec{}, err
	}
	return normalized, nil
}

func parseSpec(data []byte, path string) (Spec, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		return parseJSONSpec(data)
	}
	return parseYAMLSpec(data)
}

func parseJSONSpec(data []byte) (Spec, error) {
	var spec Spec
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("parse json: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Spec{}, fmt.Errorf("parse json: multiple documents are not supported")
		}
		return Spec{}, fmt.Errorf("parse json: %w", err)
	}
	return spec, nil
}

func parseYAMLSpec(data []byte) (Spec, error) {
	var spec Spec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("parse yaml: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Spec{}, fmt.Errorf("parse yaml: multiple documents are not supported")
		}
		return Spec{}, fmt.Errorf("parse yaml: %w", err)
	}
	return spec, nil
}
