package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// Job represents a single LLM call attempt managed by the Scheduler.
type Job struct {
	JobID   string
	LeaseID string

	TenantID, Provider, Model string
	Prompt                    string
	MaxOutputTokens           uint64
	WantDailyBudget           bool

	Execute func(ctx context.Context) (actualTokens uint64, err error)
}

// Scheduler coordinates Reserve/Complete attempts across per-provider queues.
type Scheduler struct {
	limiter  Limiter
	workers  int
	observer SchedulerObserver

	submitCh  chan Job
	requeueCh chan requeueRequest
	workCh    chan Job
	stopCh    chan struct{}
	doneCh    chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc

	state *schedulerState

	now             func() time.Time
	newLeaseID      func() string
	jitter          func(time.Duration) time.Duration
	errorRetryDelay time.Duration
	idleInterval    time.Duration
}

// NewScheduler creates a Scheduler with the default configuration.
func NewScheduler(limiter Limiter, workers int) *Scheduler {
	return newScheduler(limiter, workers, defaultSchedulerConfig())
}

// NewSchedulerWithObserver creates a Scheduler with an observer.
func NewSchedulerWithObserver(limiter Limiter, workers int, observer SchedulerObserver) *Scheduler {
	cfg := defaultSchedulerConfig()
	cfg.observer = observer
	return newScheduler(limiter, workers, cfg)
}

// Submit enqueues a job for scheduling.
func (s *Scheduler) Submit(job Job) {
	select {
	case <-s.doneCh:
		return
	case s.submitCh <- job:
	}
}

// Shutdown stops the scheduler and waits for workers to finish.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.cancel()
	})
	wait := make(chan struct{})
	go func() {
		<-s.doneCh
		s.wg.Wait()
		close(wait)
	}()
	select {
	case <-wait:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// newScheduler builds a Scheduler with custom configuration, primarily for tests.
func newScheduler(limiter Limiter, workers int, cfg schedulerConfig) *Scheduler {
	if workers <= 0 {
		workers = 1
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}
	if cfg.newLeaseID == nil {
		cfg.newLeaseID = NewULID
	}
	if cfg.jitter == nil {
		cfg.jitter = func(time.Duration) time.Duration { return 0 }
	}
	if cfg.errorRetryDelay <= 0 {
		cfg.errorRetryDelay = defaultErrorRetryDelay
	}
	if cfg.idleInterval <= 0 {
		cfg.idleInterval = defaultIdleInterval
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		limiter:         limiter,
		workers:         workers,
		observer:        cfg.observer,
		submitCh:        make(chan Job, workers*4),
		requeueCh:       make(chan requeueRequest, workers*4),
		workCh:          make(chan Job, workers),
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
		ctx:             ctx,
		cancel:          cancel,
		state:           newSchedulerState(),
		now:             cfg.now,
		newLeaseID:      cfg.newLeaseID,
		jitter:          cfg.jitter,
		errorRetryDelay: cfg.errorRetryDelay,
		idleInterval:    cfg.idleInterval,
	}
	go s.run()
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	return s
}
