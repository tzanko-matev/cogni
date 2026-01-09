package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/spec"
)

// validateTasks checks task entries for correctness.
func validateTasks(cfg *spec.Config, baseDir string, agentIDs, adapterIDs map[string]struct{}, add issueAdder) {
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
		} else if taskType != "qa" && taskType != "cucumber_eval" && taskType != "question_eval" {
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", task.Type))
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
		case "qa":
			validateQATask(task, fieldPrefix, baseDir, add)
		case "cucumber_eval":
			validateCucumberTask(task, fieldPrefix, baseDir, adapterIDs, add)
		case "question_eval":
			validateQuestionTask(task, fieldPrefix, baseDir, add)
		}
	}
}

// validateQATask enforces QA task requirements.
func validateQATask(task spec.TaskConfig, fieldPrefix, baseDir string, add issueAdder) {
	if strings.TrimSpace(task.Prompt) == "" {
		add(fieldPrefix+".prompt", "is required")
	}
	if strings.TrimSpace(task.Eval.JSONSchema) != "" {
		schemaPath := task.Eval.JSONSchema
		if !filepath.IsAbs(schemaPath) {
			schemaPath = filepath.Join(baseDir, schemaPath)
		}
		info, err := os.Stat(schemaPath)
		if err != nil {
			add(fieldPrefix+".eval.json_schema", fmt.Sprintf("schema not found at %q", task.Eval.JSONSchema))
		} else if info.IsDir() {
			add(fieldPrefix+".eval.json_schema", fmt.Sprintf("schema path %q is a directory", task.Eval.JSONSchema))
		}
	}
}

// validateCucumberTask enforces cucumber evaluation task requirements.
func validateCucumberTask(task spec.TaskConfig, fieldPrefix, baseDir string, adapterIDs map[string]struct{}, add issueAdder) {
	adapterID := strings.TrimSpace(task.Adapter)
	if adapterID == "" {
		add(fieldPrefix+".adapter", "is required")
	} else if _, ok := adapterIDs[adapterID]; !ok {
		add(fieldPrefix+".adapter", fmt.Sprintf("unknown adapter %q", adapterID))
	}
	if len(task.Features) == 0 {
		add(fieldPrefix+".features", "must include at least one entry")
	}
	for featureIndex, feature := range task.Features {
		feature = strings.TrimSpace(feature)
		if feature == "" {
			add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), "is required")
			continue
		}
		featurePath := feature
		if !filepath.IsAbs(featurePath) {
			featurePath = filepath.Join(baseDir, featurePath)
		}
		if hasGlob(feature) {
			matches, err := filepath.Glob(featurePath)
			if err != nil {
				add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("invalid glob %q", feature))
				continue
			}
			if len(matches) == 0 {
				add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("no matches for %q", feature))
			}
			continue
		}
		info, err := os.Stat(featurePath)
		if err != nil {
			add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("path not found at %q", feature))
		} else if info.IsDir() {
			add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("path %q is a directory", feature))
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
