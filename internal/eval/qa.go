package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type QAConfig struct {
	JSONSchemaPath    string
	MustContain       []string
	ValidateCitations bool
	RepoRoot          string
}

type QAArtifacts struct {
	SchemaErrors       []string
	CitationErrors     []string
	MustContainMissing []string
}

type QAResult struct {
	Status        string
	FailureReason string
	SchemaValid   bool
	CitationValid bool
	Artifacts     QAArtifacts
}

func EvaluateQA(output string, cfg QAConfig) QAResult {
	result := QAResult{
		Status:        "pass",
		SchemaValid:   true,
		CitationValid: true,
	}

	var parsed any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		result.Status = "fail"
		result.FailureReason = "invalid_json"
		result.SchemaValid = false
		result.CitationValid = false
		return result
	}

	if cfg.JSONSchemaPath != "" {
		schemaValid, errors, err := validateSchema(parsed, cfg.JSONSchemaPath, cfg.RepoRoot)
		if err != nil {
			result.Status = "error"
			result.FailureReason = "runtime_error"
			result.SchemaValid = false
			result.Artifacts.SchemaErrors = []string{err.Error()}
			return result
		}
		if !schemaValid {
			result.Status = "fail"
			result.FailureReason = "schema_validation_failed"
			result.SchemaValid = false
			result.Artifacts.SchemaErrors = errors
			return result
		}
	}

	if len(cfg.MustContain) > 0 {
		missing := findMissingMustContain(output, parsed, cfg.MustContain)
		if len(missing) > 0 {
			result.Status = "fail"
			result.FailureReason = "must_contain_failed"
			result.Artifacts.MustContainMissing = missing
			return result
		}
	}

	if cfg.ValidateCitations {
		valid, errors := validateCitations(parsed, cfg.RepoRoot)
		if !valid {
			result.Status = "fail"
			result.FailureReason = "citation_validation_failed"
			result.CitationValid = false
			result.Artifacts.CitationErrors = errors
			return result
		}
	}

	return result
}

func validateSchema(parsed any, schemaPath, repoRoot string) (bool, []string, error) {
	path := schemaPath
	if repoRoot != "" && !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, nil, fmt.Errorf("resolve schema path: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	schemaURL := "file://" + filepath.ToSlash(abs)
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return false, nil, fmt.Errorf("compile schema: %w", err)
	}
	if err := schema.Validate(parsed); err != nil {
		return false, []string{err.Error()}, nil
	}
	return true, nil, nil
}

func findMissingMustContain(raw string, parsed any, required []string) []string {
	missing := make([]string, 0)
	for _, item := range required {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(raw, item) {
			continue
		}
		if containsKey(parsed, item) {
			continue
		}
		missing = append(missing, item)
	}
	return missing
}

func containsKey(value any, key string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if _, ok := typed[key]; ok {
			return true
		}
		for _, v := range typed {
			if containsKey(v, key) {
				return true
			}
		}
	case []any:
		for _, v := range typed {
			if containsKey(v, key) {
				return true
			}
		}
	}
	return false
}

type Citation struct {
	Path  string
	Start int
	End   int
}

func validateCitations(parsed any, repoRoot string) (bool, []string) {
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
		info, err := os.Stat(absPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("citation path not found: %s", citation.Path))
			continue
		}
		if info.IsDir() {
			errors = append(errors, fmt.Sprintf("citation path is directory: %s", citation.Path))
			continue
		}
		lineCount, err := countLines(absPath)
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

func extractCitations(parsed any) ([]Citation, error) {
	root, ok := parsed.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("citations require a JSON object")
	}
	raw, ok := root["citations"]
	if !ok {
		return nil, fmt.Errorf("citations not found")
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("citations must be an array")
	}
	citations := make([]Citation, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("citation entry must be an object")
		}
		path, _ := entry["path"].(string)
		linesRaw, ok := entry["lines"].([]any)
		if !ok || len(linesRaw) != 2 {
			return nil, fmt.Errorf("citation lines must be a two-item array")
		}
		start, ok := linesRaw[0].(float64)
		if !ok {
			return nil, fmt.Errorf("citation start line must be a number")
		}
		end, ok := linesRaw[1].(float64)
		if !ok {
			return nil, fmt.Errorf("citation end line must be a number")
		}
		citations = append(citations, Citation{
			Path:  path,
			Start: int(start),
			End:   int(end),
		})
	}
	return citations, nil
}

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

func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
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
