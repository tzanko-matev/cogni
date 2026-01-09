package spec

// Config is the top-level Cogni configuration schema.
type Config struct {
	Version      int           `yaml:"version"`
	Repo         RepoConfig    `yaml:"repo"`
	Agents       []AgentConfig `yaml:"agents"`
	DefaultAgent string        `yaml:"default_agent"`
	Tasks        []TaskConfig  `yaml:"tasks"`
}

// RepoConfig describes repository-level settings.
type RepoConfig struct {
	OutputDir     string   `yaml:"output_dir"`
	SetupCommands []string `yaml:"setup_commands"`
}

// AgentConfig configures an LLM agent.
type AgentConfig struct {
	ID          string  `yaml:"id"`
	Type        string  `yaml:"type"`
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	MaxSteps    int     `yaml:"max_steps"`
	Temperature float64 `yaml:"temperature"`
}

// TaskConfig configures a single evaluation task.
type TaskConfig struct {
	ID            string         `yaml:"id"`
	Type          string         `yaml:"type"`
	Agent         string         `yaml:"agent"`
	Model         string         `yaml:"model"`
	Prompt        string         `yaml:"prompt"`
	QuestionsFile string         `yaml:"questions_file"`
	Eval          TaskEval       `yaml:"eval"`
	Budget        TaskBudget     `yaml:"budget"`
	Compaction    TaskCompaction `yaml:"compaction"`
}

// TaskEval configures QA evaluation rules for a task.
type TaskEval struct {
	JSONSchema         string   `yaml:"json_schema"`
	MustContainStrings []string `yaml:"must_contain_strings"`
	ValidateCitations  bool     `yaml:"validate_citations"`
}

// TaskBudget limits resource usage for a task.
type TaskBudget struct {
	MaxTokens  int `yaml:"max_tokens"`
	MaxSeconds int `yaml:"max_seconds"`
	MaxSteps   int `yaml:"max_steps"`
}

// TaskCompaction configures history compaction for a task.
type TaskCompaction struct {
	MaxTokens             int    `yaml:"max_tokens"`
	SummaryPrompt         string `yaml:"summary_prompt"`
	SummaryPromptFile     string `yaml:"summary_prompt_file"`
	RecentUserTokenBudget int    `yaml:"recent_user_token_budget"`
	RecentToolOutputLimit int    `yaml:"recent_tool_output_limit"`
}
