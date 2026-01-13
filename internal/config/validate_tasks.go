package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/spec"
)

// validateTasks checks task entries for correctness.
func validateTasks(cfg *spec.Config, baseDir string, agentIDs map[string]struct{}, add issueAdder) {
	taskIDs := map[string]struct{}{}
	for i, task := range cfg.Tasks {
		fieldPrefix := fmt.Sprintf("tasks[%d]", i)
		id := strings.TrimSpace(task.ID)
		if id == "" {
			add(fieldPrefix+".id", "is required")
		} else if _, exists := taskIDs[id]; exists {
			add("tasks.id", fmt.Sprintf("duplicate id %q", id))
		} else {
			taskIDs[id] = struct{}{}
		}
		taskType := strings.TrimSpace(task.Type)
		if taskType == "" {
			add(fieldPrefix+".type", "is required")
		} else if taskType != "question_eval" {
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", task.Type))
		}
		if task.Concurrency != 0 {
			if task.Concurrency < 1 {
				add(fieldPrefix+".concurrency", "must be >= 1")
			}
			if taskType != "" && taskType != "question_eval" {
				add(fieldPrefix+".concurrency", "is only valid for question_eval tasks")
			}
		}
		if strings.TrimSpace(task.Agent) == "" {
			add(fieldPrefix+".agent", "is required")
		} else if _, ok := agentIDs[task.Agent]; !ok {
			add(fieldPrefix+".agent", fmt.Sprintf("unknown agent %q", task.Agent))
		}
		if task.Budget.MaxTokens < 0 {
			add(fieldPrefix+".budget.max_tokens", "must be >= 0")
		}
		if task.Budget.MaxSeconds < 0 {
			add(fieldPrefix+".budget.max_seconds", "must be >= 0")
		}
		if task.Budget.MaxSteps < 0 {
			add(fieldPrefix+".budget.max_steps", "must be >= 0")
		}
		if task.Compaction.MaxTokens < 0 {
			add(fieldPrefix+".compaction.max_tokens", "must be >= 0")
		}
		if task.Compaction.RecentUserTokenBudget < 0 {
			add(fieldPrefix+".compaction.recent_user_token_budget", "must be >= 0")
		}
		if task.Compaction.RecentToolOutputLimit < 0 {
			add(fieldPrefix+".compaction.recent_tool_output_limit", "must be >= 0")
		}
		if strings.TrimSpace(task.Compaction.SummaryPrompt) != "" && strings.TrimSpace(task.Compaction.SummaryPromptFile) != "" {
			add(fieldPrefix+".compaction.summary_prompt", "cannot be set with summary_prompt_file")
		}
		if strings.TrimSpace(task.Compaction.SummaryPromptFile) != "" {
			promptPath := task.Compaction.SummaryPromptFile
			if !filepath.IsAbs(promptPath) {
				promptPath = filepath.Join(baseDir, promptPath)
			}
			info, err := os.Stat(promptPath)
			if err != nil {
				add(fieldPrefix+".compaction.summary_prompt_file", fmt.Sprintf("file not found at %q", task.Compaction.SummaryPromptFile))
			} else if info.IsDir() {
				add(fieldPrefix+".compaction.summary_prompt_file", fmt.Sprintf("path %q is a directory", task.Compaction.SummaryPromptFile))
			}
		}
		switch taskType {
		case "question_eval":
			validateQuestionTask(task, fieldPrefix, baseDir, add)
		}
	}
}

// validateQuestionTask enforces question evaluation task requirements.
func validateQuestionTask(task spec.TaskConfig, fieldPrefix, baseDir string, add issueAdder) {
	questionsFile := strings.TrimSpace(task.QuestionsFile)
	if questionsFile == "" {
		add(fieldPrefix+".questions_file", "is required")
		return
	}
	questionsPath := questionsFile
	if !filepath.IsAbs(questionsPath) {
		questionsPath = filepath.Join(baseDir, questionsPath)
	}
	info, err := os.Stat(questionsPath)
	if err != nil {
		add(fieldPrefix+".questions_file", fmt.Sprintf("file not found at %q", questionsFile))
		return
	}
	if info.IsDir() {
		add(fieldPrefix+".questions_file", fmt.Sprintf("path %q is a directory", questionsFile))
	}
}
