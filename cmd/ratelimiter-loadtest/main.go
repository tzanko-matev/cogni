// Command ratelimiter-loadtest runs a synthetic load test against the rate limiter.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
	"cogni/pkg/ratelimiter/local"
)

// config captures command-line configuration for the load test.
type config struct {
	Mode           string
	Backend        string
	Duration       time.Duration
	Concurrency    int
	Providers      []string
	BaseURL        string
	LimitsPath     string
	MaxTokens      int
	BatchSize      int
	FlushInterval  time.Duration
	RequestTimeout time.Duration
}

// loadtestStats aggregates counters and latency samples.
type loadtestStats struct {
	reserveCount  uint64
	completeCount uint64
	allowedCount  uint64
	deniedCount   uint64
	errorCount    uint64

	mu               sync.Mutex
	reserveLatencies []int64
	completeLatency  []int64
}

// batchStats tracks batch size distribution for reserve/complete.
type batchStats struct {
	mu              sync.Mutex
	reserveBatches  []int
	completeBatches []int
}

// statsLimiter wraps a Limiter and records batch sizes.
type statsLimiter struct {
	inner ratelimiter.Limiter
	stats *batchStats
}

// Reserve forwards reserve calls to the wrapped limiter.
func (s *statsLimiter) Reserve(ctx context.Context, req ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, error) {
	return s.inner.Reserve(ctx, req)
}

// Complete forwards completion calls to the wrapped limiter.
func (s *statsLimiter) Complete(ctx context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	return s.inner.Complete(ctx, req)
}

// BatchReserve records batch sizes before forwarding.
func (s *statsLimiter) BatchReserve(ctx context.Context, req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, error) {
	s.stats.recordReserve(len(req.Requests))
	return s.inner.BatchReserve(ctx, req)
}

// BatchComplete records batch sizes before forwarding.
func (s *statsLimiter) BatchComplete(ctx context.Context, req ratelimiter.BatchCompleteRequest) (ratelimiter.BatchCompleteResponse, error) {
	s.stats.recordComplete(len(req.Requests))
	return s.inner.BatchComplete(ctx, req)
}

func main() {
	cfg := parseConfig()
	if err := cfg.validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	limiter, batcher, stats, err := buildLimiter(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if batcher != nil {
		defer func() {
			_ = batcher.Shutdown(context.Background())
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	loadStats := runLoad(ctx, limiter, cfg)
	printSummary(cfg, loadStats, stats)
}

// parseConfig reads flags and builds a config.
func parseConfig() config {
	var cfg config
	var providers string
	flag.StringVar(&cfg.Mode, "mode", "http", "mode: http or local")
	flag.StringVar(&cfg.Backend, "backend", "memory", "backend: memory or tigerbeetle")
	flag.DurationVar(&cfg.Duration, "duration", 30*time.Second, "test duration")
	flag.IntVar(&cfg.Concurrency, "concurrency", 200, "concurrent workers")
	flag.StringVar(&providers, "providers", "openai,anthropic", "comma-separated providers")
	flag.StringVar(&cfg.BaseURL, "base-url", "http://localhost:8080", "ratelimiterd base URL")
	flag.StringVar(&cfg.LimitsPath, "limits", "", "path to limits JSON file")
	flag.IntVar(&cfg.MaxTokens, "max-tokens", 200, "max tokens per request")
	flag.IntVar(&cfg.BatchSize, "batch-size", 25, "batch size for client batching")
	flag.DurationVar(&cfg.FlushInterval, "flush-interval", 2*time.Millisecond, "batch flush interval")
	flag.DurationVar(&cfg.RequestTimeout, "request-timeout", 2*time.Second, "per-request timeout")
	flag.Parse()

	cfg.Providers = splitProviders(providers)
	return cfg
}

// validate ensures the configuration is usable.
func (c config) validate() error {
	if c.Mode != "http" && c.Mode != "local" {
		return fmt.Errorf("unsupported mode: %s", c.Mode)
	}
	if c.Backend != "memory" && c.Backend != "tigerbeetle" {
		return fmt.Errorf("unsupported backend: %s", c.Backend)
	}
	if c.Mode == "local" && c.Backend != "memory" {
		return fmt.Errorf("local mode only supports memory backend")
	}
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("max-tokens must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be positive")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request-timeout must be positive")
	}
	if len(c.Providers) == 0 {
		return fmt.Errorf("at least one provider is required")
	}
	if c.Mode == "local" && c.LimitsPath == "" {
		return fmt.Errorf("limits file is required for local mode")
	}
	return nil
}

// buildLimiter constructs the target limiter and wraps it with batching.
func buildLimiter(cfg config) (ratelimiter.Limiter, *ratelimiter.Batcher, *batchStats, error) {
	var base ratelimiter.Limiter
	switch cfg.Mode {
	case "http":
		client := httpclient.New(cfg.BaseURL)
		if cfg.LimitsPath != "" {
			if err := seedLimits(cfg.BaseURL, cfg.LimitsPath, cfg.RequestTimeout); err != nil {
				return nil, nil, nil, err
			}
		}
		base = client
	case "local":
		client, err := local.NewMemoryLimiterFromFile(cfg.LimitsPath)
		if err != nil {
			return nil, nil, nil, err
		}
		base = client
	}

	stats := &batchStats{}
	wrapped := &statsLimiter{inner: base, stats: stats}
	batcher := ratelimiter.NewBatcher(wrapped, cfg.BatchSize, cfg.FlushInterval)
	return batcher, batcher, stats, nil
}

// seedLimits applies limits from a JSON file to a running server.
func seedLimits(baseURL, path string, timeout time.Duration) error {
	states, err := loadLimitStates(path)
	if err != nil {
		return err
	}
	for _, state := range states {
		if err := putLimit(baseURL, state.Definition, timeout); err != nil {
			return err
		}
	}
	return nil
}

// loadLimitStates loads limit state definitions from disk.
func loadLimitStates(path string) ([]ratelimiter.LimitState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var states []ratelimiter.LimitState
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, err
	}
	return states, nil
}

// putLimit sends an admin PUT request for a single limit definition.
func putLimit(baseURL string, def ratelimiter.LimitDefinition, timeout time.Duration) error {
	payload, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("marshal limit: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(baseURL, "/")+"/v1/admin/limits", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("admin put failed: %s", string(body))
	}
	return nil
}

// runLoad executes the concurrent load until the context expires.
func runLoad(ctx context.Context, limiter ratelimiter.Limiter, cfg config) *loadtestStats {
	loadStats := &loadtestStats{
		reserveLatencies: make([]int64, 0, cfg.Concurrency*16),
		completeLatency:  make([]int64, 0, cfg.Concurrency*16),
	}
	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed))
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				provider := cfg.Providers[rng.Intn(len(cfg.Providers))]
				model := "default"
				leaseID := ratelimiter.NewULID()
				upper := uint64(rng.Intn(cfg.MaxTokens) + 1)
				req := ratelimiter.ReserveRequest{
					LeaseID: leaseID,
					Requirements: []ratelimiter.Requirement{
						{Key: rpmKey(provider, model), Amount: 1},
						{Key: tpmKey(provider, model), Amount: upper},
						{Key: concurrencyKey(provider, model), Amount: 1},
					},
				}
				reserveStart := time.Now()
				reserveCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
				res, err := limiter.Reserve(reserveCtx, req)
				cancel()
				loadStats.recordReserveLatency(time.Since(reserveStart))
				if err != nil {
					atomic.AddUint64(&loadStats.errorCount, 1)
					continue
				}
				atomic.AddUint64(&loadStats.reserveCount, 1)
				if !res.Allowed {
					atomic.AddUint64(&loadStats.deniedCount, 1)
					continue
				}
				atomic.AddUint64(&loadStats.allowedCount, 1)

				time.Sleep(time.Duration(rng.Intn(50)) * time.Millisecond)
				actual := uint64(rng.Intn(int(upper)) + 1)
				completeReq := ratelimiter.CompleteRequest{
					LeaseID: leaseID,
					Actuals: []ratelimiter.Actual{{Key: tpmKey(provider, model), ActualAmount: actual}},
				}
				completeStart := time.Now()
				completeCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
				_, err = limiter.Complete(completeCtx, completeReq)
				cancel()
				loadStats.recordCompleteLatency(time.Since(completeStart))
				if err != nil {
					atomic.AddUint64(&loadStats.errorCount, 1)
					continue
				}
				atomic.AddUint64(&loadStats.completeCount, 1)
			}
		}(int64(i + 1))
	}
	wg.Wait()
	return loadStats
}

// printSummary renders load test metrics to stdout.
func printSummary(cfg config, stats *loadtestStats, batches *batchStats) {
	elapsed := cfg.Duration.Seconds()
	reserveCount := atomic.LoadUint64(&stats.reserveCount)
	completeCount := atomic.LoadUint64(&stats.completeCount)
	allowed := atomic.LoadUint64(&stats.allowedCount)
	denied := atomic.LoadUint64(&stats.deniedCount)
	errors := atomic.LoadUint64(&stats.errorCount)

	fmt.Println("ratelimiter load test summary")
	fmt.Printf("mode: %s backend: %s duration: %s concurrency: %d\n", cfg.Mode, cfg.Backend, cfg.Duration, cfg.Concurrency)
	fmt.Printf("reserves/sec: %.2f completes/sec: %.2f\n", float64(reserveCount)/elapsed, float64(completeCount)/elapsed)
	fmt.Printf("allowed: %d denied: %d errors: %d\n", allowed, denied, errors)
	fmt.Printf("reserve latency p50=%s p95=%s p99=%s\n",
		percentileDuration(stats.reserveLatencies, 0.50),
		percentileDuration(stats.reserveLatencies, 0.95),
		percentileDuration(stats.reserveLatencies, 0.99),
	)
	fmt.Printf("complete latency p50=%s p95=%s p99=%s\n",
		percentileDuration(stats.completeLatency, 0.50),
		percentileDuration(stats.completeLatency, 0.95),
		percentileDuration(stats.completeLatency, 0.99),
	)
	reserveSummary, completeSummary := batches.summary()
	fmt.Printf("batch reserve sizes: %s\n", reserveSummary)
	fmt.Printf("batch complete sizes: %s\n", completeSummary)
}

// recordReserveLatency appends a reserve latency sample.
func (s *loadtestStats) recordReserveLatency(d time.Duration) {
	s.mu.Lock()
	s.reserveLatencies = append(s.reserveLatencies, d.Nanoseconds())
	s.mu.Unlock()
}

// recordCompleteLatency appends a complete latency sample.
func (s *loadtestStats) recordCompleteLatency(d time.Duration) {
	s.mu.Lock()
	s.completeLatency = append(s.completeLatency, d.Nanoseconds())
	s.mu.Unlock()
}

// summary formats batch size stats for reserve and complete batches.
func (b *batchStats) summary() (string, string) {
	return b.format(b.reserveBatches), b.format(b.completeBatches)
}

// recordReserve captures a reserve batch size.
func (b *batchStats) recordReserve(size int) {
	b.mu.Lock()
	b.reserveBatches = append(b.reserveBatches, size)
	b.mu.Unlock()
}

// recordComplete captures a complete batch size.
func (b *batchStats) recordComplete(size int) {
	b.mu.Lock()
	b.completeBatches = append(b.completeBatches, size)
	b.mu.Unlock()
}

// format renders average and max batch size metrics.
func (b *batchStats) format(samples []int) string {
	if len(samples) == 0 {
		return "n/a"
	}
	var sum int
	max := samples[0]
	for _, value := range samples {
		sum += value
		if value > max {
			max = value
		}
	}
	avg := float64(sum) / float64(len(samples))
	return fmt.Sprintf("avg=%.2f max=%d count=%d", avg, max, len(samples))
}

// percentileDuration computes a duration percentile for samples in nanoseconds.
func percentileDuration(samples []int64, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	copySamples := append([]int64(nil), samples...)
	sort.Slice(copySamples, func(i, j int) bool { return copySamples[i] < copySamples[j] })
	if p <= 0 {
		return time.Duration(copySamples[0])
	}
	if p >= 1 {
		return time.Duration(copySamples[len(copySamples)-1])
	}
	pos := int(float64(len(copySamples)-1) * p)
	return time.Duration(copySamples[pos])
}

// splitProviders parses the provider list from flags.
func splitProviders(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

// rpmKey constructs the RPM key for a provider/model pair.
func rpmKey(provider, model string) ratelimiter.LimitKey {
	return ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:%s:rpm", provider, model))
}

// tpmKey constructs the TPM key for a provider/model pair.
func tpmKey(provider, model string) ratelimiter.LimitKey {
	return ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:%s:tpm", provider, model))
}

// concurrencyKey constructs the concurrency key for a provider/model pair.
func concurrencyKey(provider, model string) ratelimiter.LimitKey {
	return ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:%s:concurrency", provider, model))
}
