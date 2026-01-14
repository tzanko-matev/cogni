package duckdb_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	maxSpecDepth    = 3
	maxMapKeys      = 4
	maxListElements = 4
)

// randomSpec generates a JSON-compatible spec with bounded depth.
func randomSpec(rng *rand.Rand, depth int) interface{} {
	if depth <= 0 {
		return randomScalar(rng)
	}
	switch rng.Intn(3) {
	case 0:
		return randomMap(rng, depth)
	case 1:
		return randomList(rng, depth)
	default:
		return randomScalar(rng)
	}
}

// randomMap builds a JSON map with random keys and values.
func randomMap(rng *rand.Rand, depth int) map[string]interface{} {
	count := rng.Intn(maxMapKeys + 1)
	out := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		out[randomString(rng)] = randomSpec(rng, depth-1)
	}
	return out
}

// randomList builds a JSON list with random elements.
func randomList(rng *rand.Rand, depth int) []interface{} {
	count := rng.Intn(maxListElements + 1)
	out := make([]interface{}, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, randomSpec(rng, depth-1))
	}
	return out
}

// randomScalar produces a JSON scalar value.
func randomScalar(rng *rand.Rand) interface{} {
	switch rng.Intn(5) {
	case 0:
		return randomString(rng)
	case 1:
		return rng.Intn(2) == 0
	case 2:
		return rng.Float64() * 1000
	case 3:
		return rng.Int63n(1_000_000)
	default:
		return nil
	}
}

// randomString creates a short lowercase token for fuzz specs.
func randomString(rng *rand.Rand) string {
	length := rng.Intn(8) + 1
	letters := make([]byte, length)
	for i := 0; i < length; i++ {
		letters[i] = byte('a' + rng.Intn(26))
	}
	return string(letters)
}

// rehydrateSpec round-trips a spec through JSON to scramble map order.
func rehydrateSpec(spec interface{}) (interface{}, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func writeSeedFixture(name string, seed int64, spec interface{}) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(root, "tests", "fixtures", "duckdb", "fuzz")
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir fuzz fixtures: %w", err)
	}
	payload := map[string]interface{}{
		"name": name,
		"seed": seed,
		"spec": spec,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.Join(path, fmt.Sprintf("seed-%d.json", seed))
	return os.WriteFile(file, data, 0o644)
}

// repoRoot searches upward for the repository root containing go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}
