package config

import (
	"fmt"
	"strings"

	"cogni/internal/spec"
)

func OrderedTasks(cfg spec.Config, selectedIDs []string) ([]spec.TaskConfig, error) {
	if len(selectedIDs) == 0 {
		ordered := make([]spec.TaskConfig, len(cfg.Tasks))
		copy(ordered, cfg.Tasks)
		return ordered, nil
	}

	taskIndex := make(map[string]struct{}, len(cfg.Tasks))
	for _, task := range cfg.Tasks {
		taskIndex[task.ID] = struct{}{}
	}

	selected := make(map[string]struct{}, len(selectedIDs))
	unknown := make([]string, 0)
	unknownSet := make(map[string]struct{})
	for _, id := range selectedIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		selected[id] = struct{}{}
		if _, ok := taskIndex[id]; !ok {
			if _, seen := unknownSet[id]; !seen {
				unknown = append(unknown, id)
				unknownSet[id] = struct{}{}
			}
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown task ids: %s", strings.Join(unknown, ", "))
	}

	ordered := make([]spec.TaskConfig, 0, len(selected))
	for _, task := range cfg.Tasks {
		if _, ok := selected[task.ID]; ok {
			ordered = append(ordered, task)
		}
	}
	return ordered, nil
}
