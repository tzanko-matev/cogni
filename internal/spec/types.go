package spec

type Config struct {
	Version      int           `yaml:"version"`
	Repo         RepoConfig    `yaml:"repo"`
	Agents       []AgentConfig `yaml:"agents"`
	DefaultAgent string        `yaml:"default_agent"`
	Tasks        []TaskConfig  `yaml:"tasks"`
}

type RepoConfig struct {
	OutputDir     string   `yaml:"output_dir"`
	SetupCommands []string `yaml:"setup_commands"`
}

type AgentConfig struct {
	ID          string  `yaml:"id"`
	Type        string  `yaml:"type"`
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	MaxSteps    int     `yaml:"max_steps"`
	Temperature float64 `yaml:"temperature"`
}

type TaskConfig struct {
	ID     string        `yaml:"id"`
	Type   string        `yaml:"type"`
	Agent  string        `yaml:"agent"`
	Model  string        `yaml:"model"`
	Prompt string        `yaml:"prompt"`
	Eval   TaskEval      `yaml:"eval"`
	Budget TaskBudget    `yaml:"budget"`
}

type TaskEval struct {
	JSONSchema         string   `yaml:"json_schema"`
	MustContainStrings []string `yaml:"must_contain_strings"`
	ValidateCitations  bool     `yaml:"validate_citations"`
}

type TaskBudget struct {
	MaxTokens  int `yaml:"max_tokens"`
	MaxSeconds int `yaml:"max_seconds"`
	MaxSteps   int `yaml:"max_steps"`
}
