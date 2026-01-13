package tb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cogni/internal/registry"
	"cogni/internal/tbutil"
	"cogni/pkg/ratelimiter"
)

const (
	ledgerLimits uint32 = 1
	codeLimit    uint16 = 1
)

// Config defines dependencies and tuning for the TB backend.
type Config struct {
	ClusterID      uint32
	Addresses      []string
	Sessions       int
	MaxBatchEvents int
	FlushInterval  time.Duration
	Registry       *registry.Registry
	RegistryPath   string
	RetryPolicy    RetryPolicy
	Now            func() time.Time
}

// Backend implements the TigerBeetle-backed rate limiter.
type Backend struct {
	pool         *tbutil.ClientPool
	submitter    *tbutil.Submitter
	cancel       context.CancelFunc
	registry     *registry.Registry
	registryPath string
	retryPolicy  RetryPolicy
	denyTracker  *denyTracker
	retryRand    *retryRand
	nowFn        func() time.Time

	mu     sync.Mutex
	states map[ratelimiter.LimitKey]ratelimiter.LimitState
	leases map[string]LeaseState
}

// New creates a TB backend and starts background workers.
func New(cfg Config) (*Backend, error) {
	if cfg.Registry == nil {
		return nil, fmt.Errorf("registry required")
	}
	if cfg.MaxBatchEvents <= 0 {
		cfg.MaxBatchEvents = 8000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 200 * time.Microsecond
	}
	if cfg.Sessions <= 0 {
		cfg.Sessions = 1
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.RetryPolicy.isZero() {
		cfg.RetryPolicy = DefaultRetryPolicy()
	}

	pool, err := tbutil.NewClientPool(cfg.ClusterID, cfg.Addresses, cfg.Sessions)
	if err != nil {
		return nil, err
	}
	submitter := &tbutil.Submitter{
		In:         make(chan tbutil.WorkItem, cfg.MaxBatchEvents),
		FlushEvery: cfg.FlushInterval,
		MaxEvents:  cfg.MaxBatchEvents,
		Pool:       pool,
	}
	ctx, cancel := context.WithCancel(context.Background())
	backend := &Backend{
		pool:         pool,
		submitter:    submitter,
		cancel:       cancel,
		registry:     cfg.Registry,
		registryPath: cfg.RegistryPath,
		retryPolicy:  cfg.RetryPolicy,
		denyTracker:  newDenyTracker(),
		retryRand:    newRetryRand(time.Now().UnixNano()),
		nowFn:        cfg.Now,
		states:       map[ratelimiter.LimitKey]ratelimiter.LimitState{},
		leases:       map[string]LeaseState{},
	}
	go submitter.Run(ctx)
	go backend.decreaseLoop(ctx)
	if err := backend.loadStates(cfg.Registry.List()); err != nil {
		backend.cancel()
		_ = pool.Close()
		return nil, err
	}
	return backend, nil
}

// Close stops background workers and closes TB clients.
func (b *Backend) Close() error {
	b.cancel()
	return b.pool.Close()
}

// loadStates applies persisted states to the backend.
func (b *Backend) loadStates(states []ratelimiter.LimitState) error {
	for _, state := range states {
		if err := b.ApplyState(state); err != nil {
			return err
		}
	}
	return nil
}
