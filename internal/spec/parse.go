package spec

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func ParseConfig(data []byte) (Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf("parse config: multiple YAML documents are not supported")
		}
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
