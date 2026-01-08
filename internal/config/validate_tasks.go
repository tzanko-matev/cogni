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
		} else if taskType != "qa" && taskType != "cucumber_eval" {
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
		switch taskType {
		case "qa":
			validateQATask(task, fieldPrefix, baseDir, add)
		case "cucumber_eval":
			validateCucumberTask(task, fieldPrefix, baseDir, adapterIDs, add)
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
