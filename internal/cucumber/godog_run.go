package cucumber

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunGodogJSON executes godog and parses its JSON output.
func RunGodogJSON(ctx context.Context, repoRoot string, featurePaths []string, tags []string) ([]CukeFeatureJSON, error) {
	if len(featurePaths) == 0 {
		return nil, fmt.Errorf("no feature paths provided")
	}
	args := []string{"--format", "cucumber"}
	if tagExpr := tagExpression(tags); tagExpr != "" {
		args = append(args, "--tags", tagExpr)
	}
	args = append(args, featurePaths...)

	cmd := exec.CommandContext(ctx, "godog", args...)
	cmd.Dir = repoRoot
	cmd.Env = withoutEnv(os.Environ(), "GOTOOLDIR")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.Bytes()
	if len(output) == 0 && err != nil {
		return nil, fmt.Errorf("godog failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	features, parseErr := ParseGodogJSON(output)
	if parseErr != nil {
		return nil, fmt.Errorf("parse godog output: %w (%s)", parseErr, strings.TrimSpace(stderr.String()))
	}
	return features, nil
}
