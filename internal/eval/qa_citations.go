package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateCitations ensures citations point to valid files and ranges.
func validateCitations(parsed JSONValue, repoRoot string, fs QAFileSystem) (bool, []string) {
	citations, err := extractCitations(parsed)
	if err != nil {
		return false, []string{err.Error()}
	}
	errors := make([]string, 0)
	for _, citation := range citations {
		if citation.Path == "" {
			errors = append(errors, "citation path is empty")
			continue
		}
		absPath, err := resolveRepoPath(repoRoot, citation.Path)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		info, err := fs.Stat(absPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("citation path not found: %s", citation.Path))
			continue
		}
		if info.IsDir() {
			errors = append(errors, fmt.Sprintf("citation path is directory: %s", citation.Path))
			continue
		}
		lineCount, err := countLines(absPath, fs)
		if err != nil {
			errors = append(errors, fmt.Sprintf("read citation path: %s", citation.Path))
			continue
		}
		if citation.Start < 1 || citation.End < citation.Start {
			errors = append(errors, fmt.Sprintf("invalid citation range for %s", citation.Path))
			continue
		}
		if citation.End > lineCount {
			errors = append(errors, fmt.Sprintf("citation range out of bounds for %s", citation.Path))
			continue
		}
	}
	return len(errors) == 0, errors
}

// extractCitations extracts citation entries from a QA response.
func extractCitations(parsed JSONValue) ([]Citation, error) {
	root, ok := parsed.ObjectValue()
	if !ok {
		return nil, fmt.Errorf("citations require a JSON object")
	}
	raw, ok := root["citations"]
	if !ok {
		return nil, fmt.Errorf("citations not found")
	}
	items, ok := raw.ArrayValue()
	if !ok {
		return nil, fmt.Errorf("citations must be an array")
	}
	citations := make([]Citation, 0, len(items))
	for _, item := range items {
		entry, ok := item.ObjectValue()
		if !ok {
			return nil, fmt.Errorf("citation entry must be an object")
		}
		pathValue, ok := entry["path"]
		if !ok {
			return nil, fmt.Errorf("citation path is required")
		}
		path, ok := pathValue.StringValue()
		if !ok {
			return nil, fmt.Errorf("citation path must be a string")
		}
		linesValue, ok := entry["lines"]
		if !ok {
			return nil, fmt.Errorf("citation lines must be provided")
		}
		linesRaw, ok := linesValue.ArrayValue()
		if !ok || len(linesRaw) != 2 {
			return nil, fmt.Errorf("citation lines must be a two-item array")
		}
		start, ok := linesRaw[0].NumberValue()
		if !ok {
			return nil, fmt.Errorf("citation start line must be a number")
		}
		end, ok := linesRaw[1].NumberValue()
		if !ok {
			return nil, fmt.Errorf("citation end line must be a number")
		}
		if float64(int(start)) != start || float64(int(end)) != end {
			return nil, fmt.Errorf("citation line numbers must be integers")
		}
		citations = append(citations, Citation{
			Path:  path,
			Start: int(start),
			End:   int(end),
		})
	}
	return citations, nil
}

// resolveRepoPath ensures a citation path stays within the repo root.
func resolveRepoPath(root, path string) (string, error) {
	cleaned := filepath.Clean(path)
	if root == "" {
		return "", fmt.Errorf("repo root is required")
	}
	var abs string
	if filepath.IsAbs(cleaned) {
		abs = cleaned
	} else {
		abs = filepath.Join(root, cleaned)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("resolve citation path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("citation path escapes repo: %s", path)
	}
	return abs, nil
}

// countLines counts the lines in a file on disk.
func countLines(path string, fs QAFileSystem) (int, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	count := 1
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count, nil
}
