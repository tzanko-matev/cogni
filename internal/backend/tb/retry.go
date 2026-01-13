package tb

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"cogni/pkg/ratelimiter"
)

// RetryPolicy configures retry-after hints for denied reservations.
type RetryPolicy struct {
	Concurrency BackoffPolicy
	Rolling     RollingPolicy
	Decreasing  FixedPolicy
}

// BackoffPolicy applies exponential backoff for concurrency limits.
type BackoffPolicy struct {
	BaseMs   int
	MaxMs    int
	Factor   float64
	JitterMs int
}

// RollingPolicy applies window-based backoff for rolling limits.
type RollingPolicy struct {
	BaseMs         int
	MaxMs          int
	Factor         float64
	JitterMs       int
	WindowFraction float64
}

// FixedPolicy provides a fixed retry delay.
type FixedPolicy struct {
	FixedMs int
}

// DefaultRetryPolicy returns the v1 retry-after defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Concurrency: BackoffPolicy{
			BaseMs:   50,
			MaxMs:    2000,
			Factor:   2.0,
			JitterMs: 25,
		},
		Rolling: RollingPolicy{
			BaseMs:         100,
			MaxMs:          5000,
			Factor:         1.5,
			JitterMs:       50,
			WindowFraction: 0.1,
		},
		Decreasing: FixedPolicy{FixedMs: 10000},
	}
}

// RetryAfterMs computes a retry hint for the provided definition.
func RetryAfterMs(def ratelimiter.LimitDefinition, streak int, cfg RetryPolicy, jitterFn func(int) int) int {
	if streak < 1 {
		streak = 1
	}
	switch def.Kind {
	case ratelimiter.KindConcurrency:
		base := cfg.Concurrency.BaseMs
		raw := int(float64(base) * math.Pow(cfg.Concurrency.Factor, float64(streak)))
		capMs := cfg.Concurrency.MaxMs
		if def.TimeoutSeconds > 0 {
			timeoutMs := def.TimeoutSeconds * 1000
			if timeoutMs < capMs {
				capMs = timeoutMs
			}
		}
		delay := clampInt(raw, base, capMs)
		return applyJitter(delay, cfg.Concurrency.JitterMs, jitterFn)
	case ratelimiter.KindRolling:
		base := maxInt(cfg.Rolling.BaseMs, int(float64(def.WindowSeconds*1000)*cfg.Rolling.WindowFraction))
		raw := int(float64(base) * math.Pow(cfg.Rolling.Factor, float64(streak)))
		delay := clampInt(raw, base, cfg.Rolling.MaxMs)
		return applyJitter(delay, cfg.Rolling.JitterMs, jitterFn)
	default:
		return cfg.Rolling.BaseMs
	}
}

func (p RetryPolicy) isZero() bool {
	return p.Concurrency.BaseMs == 0 && p.Rolling.BaseMs == 0 && p.Decreasing.FixedMs == 0
}

// retryRand provides jitter for retry-after hints.
type retryRand struct {
	mu sync.Mutex
	r  *rand.Rand
}

// newRetryRand builds a retryRand with a seeded RNG.
func newRetryRand(seed int64) *retryRand {
	return &retryRand{r: rand.New(rand.NewSource(seed))}
}

// Jitter returns a random integer in [0, max].
func (r *retryRand) Jitter(max int) int {
	if max <= 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.r.Intn(max + 1)
}

func applyJitter(value, jitter int, jitterFn func(int) int) int {
	if jitter <= 0 {
		return value
	}
	if jitterFn == nil {
		jitterFn = newRetryRand(time.Now().UnixNano()).Jitter
	}
	return value + jitterFn(jitter)
}

func clampInt(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
