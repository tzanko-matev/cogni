package config

import (
	"context"
	"strings"
)

// renderScaffoldConfig builds the scaffold YAML via the compiled template.
func renderScaffoldConfig(outputDir string) (string, error) {
	var builder strings.Builder
	if err := ScaffoldConfig(outputDir).Render(context.Background(), &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
