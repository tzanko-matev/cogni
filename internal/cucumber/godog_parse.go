package cucumber

import (
	"bytes"
	"encoding/json"
)

// ParseGodogJSON parses raw JSON output from godog.
func ParseGodogJSON(data []byte) ([]CukeFeatureJSON, error) {
	data = cleanGodogOutput(data)
	var features []CukeFeatureJSON
	if err := json.Unmarshal(data, &features); err != nil {
		return nil, err
	}
	return features, nil
}

// cleanGodogOutput strips non-JSON noise from godog output.
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

// stripANSICodes removes ANSI escape sequences from output.
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
