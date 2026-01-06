package cucumber

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Expectation struct {
	ExampleID   string
	Implemented bool
	Notes       string
}

func LoadExpectations(dir string) (map[string]Expectation, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("expectations directory is required")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read expectations dir: %w", err)
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".json") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no expectations files found in %s", dir)
	}

	expectations := make(map[string]Expectation)
	for _, file := range files {
		if err := loadExpectationFile(file, expectations); err != nil {
			return nil, err
		}
	}
	return expectations, nil
}

func ValidateExpectations(expectations map[string]Expectation, examples []Example) error {
	byID := make(map[string]Example, len(examples))
	for _, example := range examples {
		byID[example.ID] = example
	}
	missing := make([]string, 0)
	for _, example := range examples {
		if _, ok := expectations[example.ID]; !ok {
			missing = append(missing, example.ID)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing expectations for examples: %s", strings.Join(missing, ", "))
	}
	for id := range expectations {
		if _, ok := byID[id]; !ok {
			return fmt.Errorf("expectation references unknown example %q", id)
		}
	}
	return nil
}

type expectationFile struct {
	Examples any `json:"examples" yaml:"examples"`
}

func loadExpectationFile(path string, expectations map[string]Expectation) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read expectations: %w", err)
	}
	var payload expectationFile
	switch {
	case strings.HasSuffix(path, ".json"):
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
	default:
		if err := yaml.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
	}
	if payload.Examples == nil {
		var fallback map[string]any
		if err := yaml.Unmarshal(data, &fallback); err == nil && len(fallback) > 0 {
			return parseExpectationMap(fallback, expectations, filepath.Base(path))
		}
		return fmt.Errorf("no examples found in %s", filepath.Base(path))
	}
	switch examples := payload.Examples.(type) {
	case map[string]any:
		return parseExpectationMap(examples, expectations, filepath.Base(path))
	case []any:
		return parseExpectationList(examples, expectations, filepath.Base(path))
	default:
		return fmt.Errorf("invalid examples in %s", filepath.Base(path))
	}
}

func parseExpectationMap(values map[string]any, expectations map[string]Expectation, source string) error {
	for id, raw := range values {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("empty example id in %s", source)
		}
		entry, err := parseExpectationValue(id, raw, source)
		if err != nil {
			return err
		}
		if _, exists := expectations[id]; exists {
			return fmt.Errorf("duplicate expectation for %q in %s", id, source)
		}
		expectations[id] = entry
	}
	return nil
}

func parseExpectationList(values []any, expectations map[string]Expectation, source string) error {
	for _, raw := range values {
		entryMap, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid expectation entry in %s", source)
		}
		rawID, ok := entryMap["id"]
		if !ok {
			return fmt.Errorf("missing id in %s", source)
		}
		id, ok := rawID.(string)
		if !ok {
			return fmt.Errorf("invalid id in %s", source)
		}
		entry, err := parseExpectationValue(id, entryMap, source)
		if err != nil {
			return err
		}
		if _, exists := expectations[id]; exists {
			return fmt.Errorf("duplicate expectation for %q in %s", id, source)
		}
		expectations[id] = entry
	}
	return nil
}

func parseExpectationValue(id string, raw any, source string) (Expectation, error) {
	id = strings.TrimSpace(id)
	entry := Expectation{ExampleID: id}
	switch typed := raw.(type) {
	case bool:
		entry.Implemented = typed
		return entry, nil
	case string:
		implemented, err := parseImplementedString(typed)
		if err != nil {
			return Expectation{}, fmt.Errorf("invalid expectation for %q in %s: %w", id, source, err)
		}
		entry.Implemented = implemented
		return entry, nil
	case map[string]any:
		if rawImplemented, ok := typed["implemented"]; ok {
			switch value := rawImplemented.(type) {
			case bool:
				entry.Implemented = value
			case string:
				implemented, err := parseImplementedString(value)
				if err != nil {
					return Expectation{}, fmt.Errorf("invalid implemented for %q in %s: %w", id, source, err)
				}
				entry.Implemented = implemented
			default:
				return Expectation{}, fmt.Errorf("invalid implemented for %q in %s", id, source)
			}
		} else {
			return Expectation{}, fmt.Errorf("missing implemented for %q in %s", id, source)
		}
		if rawNotes, ok := typed["notes"].(string); ok {
			entry.Notes = rawNotes
		}
		return entry, nil
	default:
		return Expectation{}, fmt.Errorf("invalid expectation for %q in %s", id, source)
	}
}

func parseImplementedString(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "implemented", "yes":
		return true, nil
	case "false", "not_implemented", "not-implemented", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected implemented or not_implemented")
	}
}
