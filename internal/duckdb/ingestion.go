package duckdb

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// CanonicalJSON returns deterministic JSON bytes for hashing and storage.
func CanonicalJSON(value interface{}) ([]byte, error) {
	normalized, err := normalizeJSON(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

// FingerprintJSON returns a SHA-256 hex digest for the canonical JSON.
func FingerprintJSON(value interface{}) (string, error) {
	data, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	return fingerprintBytes(data), nil
}

// CanonicalDims sorts dims into deterministic key=value strings.
func CanonicalDims(dims map[string]string) []string {
	if dims == nil {
		return nil
	}
	keys := make([]string, 0, len(dims))
	for k := range dims {
		keys = append(keys, k)
	}
	sortStrings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+dims[k])
	}
	return out
}

func fingerprintBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func sortStrings(values []string) {
	sort.Strings(values)
}

func normalizeJSON(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case json.RawMessage:
		var decoded interface{}
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil, fmt.Errorf("normalize json raw: %w", err)
		}
		return normalizeJSON(decoded)
	case []byte:
		var decoded interface{}
		if err := json.Unmarshal(v, &decoded); err != nil {
			return nil, fmt.Errorf("normalize json bytes: %w", err)
		}
		return normalizeJSON(decoded)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, inner := range v {
			norm, err := normalizeJSON(inner)
			if err != nil {
				return nil, err
			}
			out[k] = norm
		}
		return out, nil
	case map[string]string:
		out := make(map[string]interface{}, len(v))
		for k, inner := range v {
			out[k] = inner
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(v))
		for i := range v {
			norm, err := normalizeJSON(v[i])
			if err != nil {
				return nil, err
			}
			out[i] = norm
		}
		return out, nil
	case []string:
		out := make([]interface{}, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return out, nil
	default:
		return v, nil
	}
}
