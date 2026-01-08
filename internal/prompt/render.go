package prompt

import (
	"context"
	"strings"
)

// RenderCucumberPrompt builds the cucumber_eval prompt text from compiled templates.
func RenderCucumberPrompt(ctx context.Context, featurePath, featureText string, exampleIDs []string) (string, error) {
	var builder strings.Builder
	if err := CucumberPrompt(featurePath, featureText, exampleIDs).Render(ctx, &builder); err != nil {
		return "", err
	}
	return builder.String(), nil
}
