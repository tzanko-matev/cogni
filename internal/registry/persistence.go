package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cogni/pkg/ratelimiter"
)

// Load reads registry state from a JSON file if it exists.
func (r *Registry) Load(path string) error {
	if path == "" {
		return fmt.Errorf("registry path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var states []ratelimiter.LimitState
	if err := json.Unmarshal(data, &states); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = map[ratelimiter.LimitKey]ratelimiter.LimitState{}
	for _, state := range states {
		r.states[state.Definition.Key] = state
	}
	return nil
}

// Save persists registry state to a JSON file using an atomic rename.
func (r *Registry) Save(path string) error {
	if path == "" {
		return fmt.Errorf("registry path is required")
	}
	states := r.List()
	payload, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(payload)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}
	if syncErr != nil {
		_ = os.Remove(tmpPath)
		return syncErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
