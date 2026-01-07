package cucumber

import (
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

func loadExpectationFile(path string, expectations map[string]Expectation) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read expectations: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	if len(doc.Content) == 0 {
		return fmt.Errorf("no expectations found in %s", filepath.Base(path))
	}
	return parseExpectationNode(doc.Content[0], expectations, filepath.Base(path))
}

func parseExpectationNode(node *yaml.Node, expectations map[string]Expectation, source string) error {
	switch node.Kind {
	case yaml.MappingNode:
		examplesNode := mappingValue(node, "examples")
		if examplesNode != nil {
			return parseExamplesNode(examplesNode, expectations, source)
		}
		return parseExpectationMapNode(node, expectations, source)
	case yaml.SequenceNode:
		return parseExpectationListNode(node, expectations, source)
	default:
		return fmt.Errorf("invalid expectations in %s", source)
	}
}

func parseExamplesNode(node *yaml.Node, expectations map[string]Expectation, source string) error {
	switch node.Kind {
	case yaml.MappingNode:
		return parseExpectationMapNode(node, expectations, source)
	case yaml.SequenceNode:
		return parseExpectationListNode(node, expectations, source)
	default:
		return fmt.Errorf("invalid examples in %s", source)
	}
}

func parseExpectationMapNode(node *yaml.Node, expectations map[string]Expectation, source string) error {
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		id := strings.TrimSpace(key.Value)
		if id == "" {
			return fmt.Errorf("empty example id in %s", source)
		}
		entry, err := parseExpectationValueNode(id, value, source)
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

func parseExpectationListNode(node *yaml.Node, expectations map[string]Expectation, source string) error {
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			return fmt.Errorf("invalid expectation entry in %s", source)
		}
		idNode := mappingValue(item, "id")
		if idNode == nil {
			return fmt.Errorf("missing id in %s", source)
		}
		id := strings.TrimSpace(idNode.Value)
		if id == "" {
			return fmt.Errorf("invalid id in %s", source)
		}
		entry, err := parseExpectationValueNode(id, item, source)
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

func parseExpectationValueNode(id string, node *yaml.Node, source string) (Expectation, error) {
	id = strings.TrimSpace(id)
	entry := Expectation{ExampleID: id}
	switch node.Kind {
	case yaml.ScalarNode:
		implemented, err := parseImplementedString(node.Value)
		if err != nil {
			return Expectation{}, fmt.Errorf("invalid expectation for %q in %s: %w", id, source, err)
		}
		entry.Implemented = implemented
		return entry, nil
	case yaml.MappingNode:
		implNode := mappingValue(node, "implemented")
		if implNode == nil {
			return Expectation{}, fmt.Errorf("missing implemented for %q in %s", id, source)
		}
		implemented, err := parseImplementedString(implNode.Value)
		if err != nil {
			return Expectation{}, fmt.Errorf("invalid implemented for %q in %s: %w", id, source, err)
		}
		entry.Implemented = implemented
		if notesNode := mappingValue(node, "notes"); notesNode != nil {
			entry.Notes = notesNode.Value
		}
		return entry, nil
	default:
		return Expectation{}, fmt.Errorf("invalid expectation for %q in %s", id, source)
	}
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
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
