package cucumber

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadExpectations reads expectation files from a directory.
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

// loadExpectationFile parses a single expectations file into the map.
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
