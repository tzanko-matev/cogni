package ratelimiter

// LLMReserveInput captures the fields needed to build LLM limit requirements.
type LLMReserveInput struct {
	LeaseID         string
	JobID           string
	TenantID        string
	Provider        string
	Model           string
	Prompt          string
	MaxOutputTokens uint64
	WantDailyBudget bool
}

// EstimatePromptTokens returns a conservative token estimate for a prompt.
func EstimatePromptTokens(prompt string) uint64 {
	return uint64(len([]byte(prompt)))
}

// BuildLLMRequirements builds limit requirements for an LLM request.
func BuildLLMRequirements(in LLMReserveInput) []Requirement {
	upper := EstimatePromptTokens(in.Prompt) + in.MaxOutputTokens
	reqs := []Requirement{
		{Key: LimitKey(buildRPMKey(in.Provider, in.Model)), Amount: 1},
		{Key: LimitKey(buildTPMKey(in.Provider, in.Model)), Amount: upper},
		{Key: LimitKey(buildConcurrencyKey(in.Provider, in.Model)), Amount: 1},
	}
	if in.WantDailyBudget {
		reqs = append(reqs, Requirement{
			Key:    LimitKey(buildDailyKey(in.TenantID)),
			Amount: upper,
		})
	}
	return reqs
}
