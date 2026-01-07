package eval

import (
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// validateSchema compiles and validates JSON against a schema file.
func validateSchema(parsed JSONValue, schemaPath, repoRoot string) (bool, []string, error) {
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
	if err := schema.Validate(parsed.ToInterface()); err != nil {
		return false, []string{err.Error()}, nil
	}
	return true, nil, nil
}
