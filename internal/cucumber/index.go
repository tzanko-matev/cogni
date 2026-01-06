package cucumber

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ExampleIndex struct {
	byPath map[string]map[int]Example
}

func BuildExampleIndex(repoRoot string, featurePaths []string) (ExampleIndex, error) {
	resolved, err := ExpandFeaturePaths(repoRoot, featurePaths)
	if err != nil {
		return ExampleIndex{}, err
	}
	index := ExampleIndex{byPath: make(map[string]map[int]Example)}
	for _, path := range resolved {
		examples, err := ParseFeatureFile(path)
		if err != nil {
			return ExampleIndex{}, err
		}
		if len(examples) == 0 {
			continue
		}
		pathKey := normalizePath(repoRoot, path)
		if _, ok := index.byPath[pathKey]; !ok {
			index.byPath[pathKey] = make(map[int]Example)
		}
		for _, example := range examples {
			line := example.ExampleLine
			if line == 0 {
				line = example.ScenarioLine
			}
			if line == 0 {
				return ExampleIndex{}, fmt.Errorf("missing line for example %q in %s", example.ID, pathKey)
			}
			if _, exists := index.byPath[pathKey][line]; exists {
				return ExampleIndex{}, fmt.Errorf("duplicate example line %d in %s", line, pathKey)
			}
			index.byPath[pathKey][line] = example
		}
	}
	return index, nil
}

func (idx ExampleIndex) FindByLine(repoRoot, featurePath string, line int) (Example, bool) {
	if line == 0 {
		return Example{}, false
	}
	pathKey := normalizePath(repoRoot, featurePath)
	rows, ok := idx.byPath[pathKey]
	if !ok {
		return Example{}, false
	}
	example, ok := rows[line]
	return example, ok
}

func (idx ExampleIndex) Examples() []Example {
	examples := make([]Example, 0)
	paths := make([]string, 0, len(idx.byPath))
	for path := range idx.byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		lines := make([]int, 0, len(idx.byPath[path]))
		for line := range idx.byPath[path] {
			lines = append(lines, line)
		}
		sort.Ints(lines)
		for _, line := range lines {
			examples = append(examples, idx.byPath[path][line])
		}
	}
	return examples
}

func ExpandFeaturePaths(repoRoot string, entries []string) ([]string, error) {
	paths := make([]string, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if hasGlob(entry) {
			resolved := resolvePath(repoRoot, entry)
			matches, err := filepath.Glob(resolved)
			if err != nil {
				return nil, fmt.Errorf("expand glob %q: %w", entry, err)
			}
			for _, match := range matches {
				paths = appendUnique(paths, seen, match)
			}
			continue
		}
		resolved := resolvePath(repoRoot, entry)
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, fmt.Errorf("stat feature path %q: %w", entry, err)
		}
		if info.IsDir() {
			dirPaths, err := collectFeatureFiles(resolved)
			if err != nil {
				return nil, err
			}
			for _, path := range dirPaths {
				paths = appendUnique(paths, seen, path)
			}
			continue
		}
		paths = appendUnique(paths, seen, resolved)
	}
	sort.Strings(paths)
	return paths, nil
}

func resolvePath(repoRoot, path string) string {
	if filepath.IsAbs(path) || repoRoot == "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(repoRoot, path))
}

func normalizePath(repoRoot, path string) string {
	abs := resolvePath(repoRoot, path)
	return filepath.Clean(abs)
}

func collectFeatureFiles(root string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".feature") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk feature root %q: %w", root, err)
	}
	return paths, nil
}

func appendUnique(paths []string, seen map[string]struct{}, path string) []string {
	normalized := filepath.Clean(path)
	if _, ok := seen[normalized]; ok {
		return paths
	}
	seen[normalized] = struct{}{}
	return append(paths, normalized)
}

func hasGlob(value string) bool {
	return strings.ContainsAny(value, "*?[]")
}
