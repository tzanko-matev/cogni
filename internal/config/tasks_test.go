package config

import (
	"strings"
	"testing"

	"cogni/internal/spec"
)

// TestOrderedTasksDefaultsToConfigOrder verifies default ordering behavior.
func TestOrderedTasksDefaultsToConfigOrder(t *testing.T) {
	cfg := spec.Config{
		Tasks: []spec.TaskConfig{
			{ID: "a"},
			{ID: "b"},
			{ID: "c"},
		},
	}
	ordered, err := OrderedTasks(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ordered) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(ordered))
	}
	if ordered[0].ID != "a" || ordered[1].ID != "b" || ordered[2].ID != "c" {
		t.Fatalf("unexpected order: %v", []string{ordered[0].ID, ordered[1].ID, ordered[2].ID})
	}
}

// TestOrderedTasksSelectedUsesConfigOrder verifies selection preserves config order.
func TestOrderedTasksSelectedUsesConfigOrder(t *testing.T) {
	cfg := spec.Config{
		Tasks: []spec.TaskConfig{
			{ID: "task-a"},
			{ID: "task-b"},
			{ID: "task-c"},
		},
	}
	ordered, err := OrderedTasks(cfg, []string{"task-c", "task-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ordered) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(ordered))
	}
	if ordered[0].ID != "task-a" || ordered[1].ID != "task-c" {
		t.Fatalf("unexpected order: %v", []string{ordered[0].ID, ordered[1].ID})
	}
}

// TestOrderedTasksUnknownIDs verifies unknown task IDs return errors.
func TestOrderedTasksUnknownIDs(t *testing.T) {
	cfg := spec.Config{
		Tasks: []spec.TaskConfig{
			{ID: "alpha"},
			{ID: "beta"},
		},
	}
	_, err := OrderedTasks(cfg, []string{"alpha", "missing", "missing"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected error to mention missing, got %v", err)
	}
}
