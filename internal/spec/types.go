package spec

import "cogni/pkg/ratelimiter"

// Config is the top-level Cogni configuration schema.
type Config struct {
	Version      int               `yaml:"version"`
	Repo         RepoConfig        `yaml:"repo"`
	Agents       []AgentConfig     `yaml:"agents"`
	DefaultAgent string            `yaml:"default_agent"`
	RateLimiter  RateLimiterConfig `yaml:"rate_limiter"`
	Tasks        []TaskConfig      `yaml:"tasks"`
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
	QuestionsFile string         `yaml:"questions_file"`
	Budget        TaskBudget     `yaml:"budget"`
	Compaction    TaskCompaction `yaml:"compaction"`
	Concurrency   int            `yaml:"concurrency"`
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

// RateLimiterConfig configures the rate limiter integration.
type RateLimiterConfig struct {
	Mode             string                   `yaml:"mode"`
	BaseURL          string                   `yaml:"base_url"`
	Limits           []ratelimiter.LimitState `yaml:"limits"`
	LimitsPath       string                   `yaml:"limits_path"`
	Workers          int                      `yaml:"workers"`
	RequestTimeoutMs int                      `yaml:"request_timeout_ms"`
	MaxOutputTokens  uint64                   `yaml:"max_output_tokens"`
	Batch            BatchConfig              `yaml:"batch"`
}

// BatchConfig configures request batching for the limiter client.
type BatchConfig struct {
	Size    int `yaml:"size"`
	FlushMs int `yaml:"flush_ms"`
}
