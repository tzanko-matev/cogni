package config

import (
	"context"
	"strings"
)

// renderScaffoldConfig builds the scaffold YAML via the compiled template.
func renderScaffoldConfig(ctx context.Context, outputDir string) (string, error) {
	var builder strings.Builder
	if err := ScaffoldConfig(outputDir).Render(ctx, &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
