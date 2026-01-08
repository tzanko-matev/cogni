package agent

import "math"

// SummaryPrefix marks summary messages inserted during compaction.
const SummaryPrefix = "Summary of previous context:\n"

// DefaultSummaryPrompt guides the compaction summarizer when no override is supplied.
const DefaultSummaryPrompt = `You are summarizing a coding task for future continuation.
Summarize the conversation and tool outputs so far with:
- goal and constraints
- decisions made
- important facts, files, and paths
- current state, errors, and blockers
- next steps
Be concise. Use plain text. Do not add new instructions or speculation. Do not include the summary prefix.`

const (
	defaultCompactionSoftLimitFraction = 0.8
	defaultRecentUserTokenFraction     = 0.25
)

// CompactionConfig configures history compaction behavior.
type CompactionConfig struct {
	SoftLimit             int
	HardLimit             int
	SummaryPrompt         string
	RecentUserTokenBudget int
	RecentToolOutputLimit int
}

// CompactionStats captures compaction telemetry for logging and metrics.
type CompactionStats struct {
	BeforeTokens  int
	AfterTokens   int
	SummaryTokens int
}

// NormalizeCompactionConfig applies default soft limits and budgets.
func NormalizeCompactionConfig(cfg CompactionConfig) CompactionConfig {
	if cfg.HardLimit <= 0 && cfg.SoftLimit > 0 {
		cfg.HardLimit = cfg.SoftLimit
	}
	if cfg.SoftLimit <= 0 && cfg.HardLimit > 0 {
		cfg.SoftLimit = int(math.Ceil(float64(cfg.HardLimit) * defaultCompactionSoftLimitFraction))
		if cfg.SoftLimit <= 0 {
			cfg.SoftLimit = cfg.HardLimit
		}
	}
	if cfg.RecentUserTokenBudget <= 0 {
		base := cfg.SoftLimit
		if base <= 0 {
			base = cfg.HardLimit
		}
		if base > 0 {
			cfg.RecentUserTokenBudget = int(math.Ceil(float64(base) * defaultRecentUserTokenFraction))
		}
	}
	return cfg
}
