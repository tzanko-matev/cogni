package memory

import "cogni/pkg/ratelimiter"

const (
	decreaseRetryMs       = 10000
	defaultRollingRetryMs = 100
	defaultConcurrencyMs  = 50
	maxRollingRetryMs     = 5000
	maxConcurrencyRetryMs = 2000
	rollingWindowFraction = 0.1
)

func retryAfter(def ratelimiter.LimitDefinition) int {
	switch def.Kind {
	case ratelimiter.KindConcurrency:
		capMs := def.TimeoutSeconds * 1000
		if capMs <= 0 {
			capMs = maxConcurrencyRetryMs
		}
		if capMs > maxConcurrencyRetryMs {
			capMs = maxConcurrencyRetryMs
		}
		if capMs < defaultConcurrencyMs {
			capMs = defaultConcurrencyMs
		}
		return capMs
	case ratelimiter.KindRolling:
		base := int(float64(def.WindowSeconds*1000) * rollingWindowFraction)
		if base < defaultRollingRetryMs {
			base = defaultRollingRetryMs
		}
		if base > maxRollingRetryMs {
			base = maxRollingRetryMs
		}
		return base
	default:
		return defaultRollingRetryMs
	}
}
