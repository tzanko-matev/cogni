package ratelimiter

import (
	"testing"
	"time"
)

func TestBuildLLMRequirements_ContainsExpectedKeys(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		input := LLMReserveInput{
			TenantID:        "tenant-a",
			Provider:        "openai",
			Model:           "gpt-4o",
			Prompt:          "hello",
			MaxOutputTokens: 50,
			WantDailyBudget: true,
		}
		reqs := BuildLLMRequirements(input)
		expected := map[LimitKey]uint64{}
		upper := uint64(len([]byte(input.Prompt))) + input.MaxOutputTokens
		expected[LimitKey(buildRPMKey(input.Provider, input.Model))] = 1
		expected[LimitKey(buildTPMKey(input.Provider, input.Model))] = upper
		expected[LimitKey(buildConcurrencyKey(input.Provider, input.Model))] = 1
		expected[LimitKey(buildDailyKey(input.TenantID))] = upper

		if len(reqs) != len(expected) {
			t.Fatalf("expected %d requirements, got %d", len(expected), len(reqs))
		}
		for _, req := range reqs {
			amount, ok := expected[req.Key]
			if !ok {
				t.Fatalf("unexpected key %s", req.Key)
			}
			if req.Amount != amount {
				t.Fatalf("expected %s amount %d, got %d", req.Key, amount, req.Amount)
			}
		}
	})
}
