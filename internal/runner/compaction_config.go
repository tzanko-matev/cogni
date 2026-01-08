package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/agent"
	"cogni/internal/spec"
)

// buildCompactionConfig resolves compaction settings for a task.
func buildCompactionConfig(task spec.TaskConfig, repoRoot string) (agent.CompactionConfig, error) {
	prompt := strings.TrimSpace(task.Compaction.SummaryPrompt)
	if strings.TrimSpace(task.Compaction.SummaryPromptFile) != "" {
		path := task.Compaction.SummaryPromptFile
		if !filepath.IsAbs(path) {
			path = filepath.Join(repoRoot, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return agent.CompactionConfig{}, fmt.Errorf("read compaction summary prompt: %w", err)
		}
		prompt = strings.TrimSpace(string(data))
	}

	cfg := agent.CompactionConfig{
		SoftLimit:             task.Compaction.MaxTokens,
		HardLimit:             task.Budget.MaxTokens,
		SummaryPrompt:         prompt,
		RecentUserTokenBudget: task.Compaction.RecentUserTokenBudget,
		RecentToolOutputLimit: task.Compaction.RecentToolOutputLimit,
	}
	return agent.NormalizeCompactionConfig(cfg), nil
}
