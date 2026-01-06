package cucumber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CukeFeatureJSON struct {
	URI      string        `json:"uri"`
	Elements []CukeElement `json:"elements"`
}

type CukeElement struct {
	Name  string     `json:"name"`
	Line  int        `json:"line"`
	Steps []CukeStep `json:"steps"`
}

type CukeStep struct {
	Result CukeResult `json:"result"`
}

type CukeResult struct {
	Status string `json:"status"`
}

type GodogScenarioResult struct {
	ExampleID    string
	Status       string
	FeaturePath  string
	ScenarioName string
	Line         int
}

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

func ParseGodogJSON(data []byte) ([]CukeFeatureJSON, error) {
	data = cleanGodogOutput(data)
	var features []CukeFeatureJSON
	if err := json.Unmarshal(data, &features); err != nil {
		return nil, err
	}
	return features, nil
}

func NormalizeGodogResults(repoRoot string, features []CukeFeatureJSON, index ExampleIndex) ([]GodogScenarioResult, error) {
	results := make([]GodogScenarioResult, 0)
	for _, feature := range features {
		featurePath := normalizePath(repoRoot, feature.URI)
		for _, element := range feature.Elements {
			example, ok := index.FindByLine(repoRoot, feature.URI, element.Line)
			if !ok {
				return nil, fmt.Errorf("no example matches %s:%d", featurePath, element.Line)
			}
			status := deriveScenarioStatus(element.Steps)
			results = append(results, GodogScenarioResult{
				ExampleID:    example.ID,
				Status:       status,
				FeaturePath:  featurePath,
				ScenarioName: element.Name,
				Line:         element.Line,
			})
		}
	}
	return results, nil
}

func deriveScenarioStatus(steps []CukeStep) string {
	hasPending := false
	hasUndefined := false
	hasSkipped := false
	for _, step := range steps {
		switch strings.ToLower(strings.TrimSpace(step.Result.Status)) {
		case "failed":
			return "failed"
		case "undefined":
			hasUndefined = true
		case "pending":
			hasPending = true
		case "skipped":
			hasSkipped = true
		}
	}
	switch {
	case hasUndefined:
		return "undefined"
	case hasPending:
		return "pending"
	case hasSkipped:
		return "skipped"
	default:
		return "passed"
	}
}

func tagExpression(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !strings.HasPrefix(tag, "@") && !strings.HasPrefix(tag, "~") {
			tag = "@" + tag
		}
		parts = append(parts, tag)
	}
	return strings.Join(parts, " and ")
}

func withoutEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func cleanGodogOutput(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	stripped := stripANSICodes(data)
	stripped = bytes.TrimSpace(stripped)
	if len(stripped) == 0 {
		return stripped
	}
	if stripped[0] == '[' || stripped[0] == '{' {
		return stripped
	}
	for i, b := range stripped {
		if b == '[' || b == '{' {
			return bytes.TrimSpace(stripped[i:])
		}
	}
	return stripped
}

func stripANSICodes(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if data[i] == 0x1b && i+1 < len(data) && data[i+1] == '[' {
			i += 2
			for i < len(data) {
				ch := data[i]
				i++
				if ch >= 0x40 && ch <= 0x7e {
					break
				}
			}
			continue
		}
		out = append(out, data[i])
		i++
	}
	return out
}
