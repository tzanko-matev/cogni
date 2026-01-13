package ratelimiter

import "fmt"

// buildRPMKey formats the RPM limit key for a provider/model pair.
func buildRPMKey(provider, model string) string {
	return fmt.Sprintf("global:llm:%s:%s:rpm", provider, model)
}

// buildTPMKey formats the TPM limit key for a provider/model pair.
func buildTPMKey(provider, model string) string {
	return fmt.Sprintf("global:llm:%s:%s:tpm", provider, model)
}

// buildConcurrencyKey formats the concurrency limit key for a provider/model pair.
func buildConcurrencyKey(provider, model string) string {
	return fmt.Sprintf("global:llm:%s:%s:concurrency", provider, model)
}

// buildDailyKey formats the tenant daily budget key.
func buildDailyKey(tenantID string) string {
	return fmt.Sprintf("tenant:%s:llm:daily_tokens", tenantID)
}
