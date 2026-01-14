package ratelimiter

import (
	"math/rand"
	"sync"
	"time"
)

const (
	defaultErrorRetryDelay = 100 * time.Millisecond
	defaultIdleInterval    = 5 * time.Millisecond
	defaultJitterMax       = 25 * time.Millisecond
)

// schedulerConfig overrides scheduler behavior for tests or tuning.
type schedulerConfig struct {
	now             func() time.Time
	newLeaseID      func() string
	jitter          func(time.Duration) time.Duration
	errorRetryDelay time.Duration
	idleInterval    time.Duration
	observer        SchedulerObserver
}

// defaultSchedulerConfig returns the production scheduler defaults.
func defaultSchedulerConfig() schedulerConfig {
	jitterSource := newLockedRand(time.Now().UnixNano())
	return schedulerConfig{
		now:             time.Now,
		newLeaseID:      NewULID,
		jitter:          jitterSource.Jitter,
		errorRetryDelay: defaultErrorRetryDelay,
		idleInterval:    defaultIdleInterval,
		observer:        nil,
	}
}

// lockedRand provides a concurrency-safe jitter source.
type lockedRand struct {
	mu sync.Mutex
	r  *rand.Rand
}

// newLockedRand initializes a lockedRand with the given seed.
func newLockedRand(seed int64) *lockedRand {
	return &lockedRand{r: rand.New(rand.NewSource(seed))}
}

// Jitter returns a random duration up to a maximum bound.
func (l *lockedRand) Jitter(base time.Duration) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	max := defaultJitterMax
	if base > 0 && base < defaultJitterMax {
		max = base
	}
	n := l.r.Int63n(int64(max) + 1)
	return time.Duration(n)
}
