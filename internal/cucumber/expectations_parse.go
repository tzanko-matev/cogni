package cucumber

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseExpectationNode dispatches to map or list parsing based on node kind.
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

// parseExamplesNode parses the "examples" section of a file.
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

// parseExpectationMapNode parses a mapping of ids to expectations.
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

// parseExpectationListNode parses a list of expectation entries.
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

// parseExpectationValueNode parses a single expectation entry for an id.
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

// mappingValue returns the value node for a key in a mapping node.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// parseImplementedString normalizes implemented/not_implemented values.
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
