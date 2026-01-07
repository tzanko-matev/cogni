package eval

import (
	"encoding/json"
	"strings"
)

// EvaluateQA validates a QA response payload against schema and citation rules.
func EvaluateQA(output string, cfg QAConfig) QAResult {
	return EvaluateQAWithDeps(output, cfg, QADeps{})
}

// EvaluateQAWithDeps validates QA output using injectable dependencies.
func EvaluateQAWithDeps(output string, cfg QAConfig, deps QADeps) QAResult {
	result := QAResult{
		Status:        "pass",
		SchemaValid:   true,
		CitationValid: true,
	}

	fs := deps.FS
	if fs == nil {
		fs = osQAFileSystem{}
	}

	var parsed JSONValue
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
		valid, errors := validateCitations(parsed, cfg.RepoRoot, fs)
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

// findMissingMustContain checks raw text and parsed JSON for required tokens.
func findMissingMustContain(raw string, parsed JSONValue, required []string) []string {
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

// containsKey reports whether a JSON object tree contains the given key.
func containsKey(value JSONValue, key string) bool {
	if object, ok := value.ObjectValue(); ok {
		if _, found := object[key]; found {
			return true
		}
		for _, v := range object {
			if containsKey(v, key) {
				return true
			}
		}
		return false
	}
	if array, ok := value.ArrayValue(); ok {
		for _, v := range array {
			if containsKey(v, key) {
				return true
			}
		}
	}
	return false
}
