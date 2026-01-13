//go:build cucumber

package ratelimiter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"

	"cogni/internal/api"
	"cogni/internal/backend/memory"
	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

// TestRateLimiterFeatures executes the rate limiter feature scenarios via godog.
func TestRateLimiterFeatures(t *testing.T) {
	featurePath := filepath.Join("..", "..", "spec", "features", "rate-limiter", "testing.feature")
	suite := godog.TestSuite{
		Name:                "rate-limiter",
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:    "pretty",
			Paths:     []string{featurePath},
			Strict:    true,
			TestingT:  t,
			Randomize: 0,
		},
	}
	if suite.Run() != 0 {
		t.Fatalf("non-zero godog status")
	}
}

// InitializeScenario wires step definitions for the rate limiter feature tests.
func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &rateLimiterState{}
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return ctx, state.reset()
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		state.close()
		return ctx, nil
	})

	ctx.Step(`^a rolling limit "([^"]+)" with capacity (\d+) and window (\d+) seconds$`, state.givenRollingLimit)
	ctx.Step(`^a rolling limit "([^"]+)" with capacity (\d+) and overage "([^"]+)"$`, state.givenRollingLimitWithOverage)
	ctx.Step(`^a concurrency limit "([^"]+)" with capacity (\d+) and timeout (\d+) seconds$`, state.givenConcurrencyLimit)
	ctx.Step(`^limits "([^"]+)" capacity (\d+) and "([^"]+)" capacity (\d+) in the same request$`, state.givenDualLimits)
	ctx.Step(`^I reserve amount (\d+) for lease "([^"]+)"$`, state.reserveAmountForLease)
	ctx.Step(`^I reserve amount (\d+) for both limits in a single request$`, state.reserveAmountForBothLimits)
	ctx.Step(`^I complete lease "([^"]+)"$`, state.completeLease)
	ctx.Step(`^I complete lease "([^"]+)" with actual amount (\d+)$`, state.completeLeaseWithActual)
	ctx.Step(`^I send a batch reserve with leases "([^"]+)" and "([^"]+)" for amount (\d+) each$`, state.batchReserveTwo)
	ctx.Step(`^the third reserve is denied$`, state.thirdReserveDenied)
	ctx.Step(`^the reserve is denied$`, state.lastReserveDenied)
	ctx.Step(`^the "([^"]+)" limit remains at (\d+)$`, state.limitCapacityRemains)
	ctx.Step(`^the response is returned within (\d+) milliseconds$`, state.responseWithin)
	ctx.Step(`^I can reserve amount (\d+) for lease "([^"]+)" within (\d+) milliseconds$`, state.reserveWithin)
	ctx.Step(`^the debt for "([^"]+)" is (\d+)$`, state.debtIs)
	ctx.Step(`^result (\d+) is allowed and result (\d+) is denied$`, state.batchResultOrder)
	ctx.Step(`^the batch response is returned within (\d+) milliseconds$`, state.responseWithin)
	ctx.Step(`^the admin decreases capacity for "([^"]+)" to (\d+)$`, state.adminDecreasesCapacity)
	ctx.Step(`^new reservations for "([^"]+)" are denied with error "([^"]+)"$`, state.newReservationDeniedWithError)
	ctx.Step(`^when the available balance is at least (\d+) the decrease is applied$`, state.applyDecreaseWhenAvailable)
	ctx.Step(`^reservations are accepted again$`, state.reservationsAcceptedAgain)
}

// rateLimiterState holds scenario state for the feature tests.
type rateLimiterState struct {
	server         *httptest.Server
	baseURL        string
	backend        *memory.MemoryBackend
	registry       *registry.Registry
	clock          *testutil.FakeClock
	defs           map[ratelimiter.LimitKey]ratelimiter.LimitDefinition
	keyAliases     map[string]ratelimiter.LimitKey
	singleKey      ratelimiter.LimitKey
	decreaseKey    ratelimiter.LimitKey
	reserveHistory []ratelimiter.ReserveResponse
	lastReserve    ratelimiter.ReserveResponse
	lastBatch      ratelimiter.BatchReserveResponse
	lastComplete   ratelimiter.CompleteResponse
	lastDuration   time.Duration
}

// reset initializes the scenario state and starts the HTTP server.
func (s *rateLimiterState) reset() error {
	s.close()
	s.clock = testutil.NewFakeClock(time.Unix(0, 0))
	s.backend = memory.New(s.clock)
	s.registry = registry.New()
	s.backend.AttachRegistry(s.registry, "")
	handler := api.NewHandler(api.Config{
		Registry: s.registry,
		Backend:  s.backend,
		Now:      s.clock.Now,
	})
	s.server = httptest.NewServer(handler)
	s.baseURL = s.server.URL
	s.defs = map[ratelimiter.LimitKey]ratelimiter.LimitDefinition{}
	s.keyAliases = map[string]ratelimiter.LimitKey{}
	s.reserveHistory = nil
	s.lastDuration = 0
	s.lastReserve = ratelimiter.ReserveResponse{}
	s.lastBatch = ratelimiter.BatchReserveResponse{}
	s.lastComplete = ratelimiter.CompleteResponse{}
	s.singleKey = ""
	s.decreaseKey = ""
	return nil
}

// close shuts down the HTTP server if it is running.
func (s *rateLimiterState) close() {
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
}

// givenRollingLimit defines a rolling limit for the scenario.
func (s *rateLimiterState) givenRollingLimit(key string, capacity, window int) error {
	def := ratelimiter.LimitDefinition{
		Key:           ratelimiter.LimitKey(key),
		Kind:          ratelimiter.KindRolling,
		Capacity:      uint64(capacity),
		WindowSeconds: window,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	}
	if err := s.putLimit(def); err != nil {
		return err
	}
	s.defs[def.Key] = def
	s.singleKey = def.Key
	return nil
}

// givenRollingLimitWithOverage defines a rolling limit with a specific overage policy.
func (s *rateLimiterState) givenRollingLimitWithOverage(key string, capacity int, overage string) error {
	policy := ratelimiter.OveragePolicy(strings.ToLower(strings.TrimSpace(overage)))
	def := ratelimiter.LimitDefinition{
		Key:           ratelimiter.LimitKey(key),
		Kind:          ratelimiter.KindRolling,
		Capacity:      uint64(capacity),
		WindowSeconds: 60,
		Unit:          "tokens",
		Overage:       policy,
	}
	if err := s.putLimit(def); err != nil {
		return err
	}
	s.defs[def.Key] = def
	s.singleKey = def.Key
	return nil
}

// givenConcurrencyLimit defines a concurrency limit for the scenario.
func (s *rateLimiterState) givenConcurrencyLimit(key string, capacity, timeout int) error {
	def := ratelimiter.LimitDefinition{
		Key:            ratelimiter.LimitKey(key),
		Kind:           ratelimiter.KindConcurrency,
		Capacity:       uint64(capacity),
		TimeoutSeconds: timeout,
		Unit:           "requests",
		Overage:        ratelimiter.OverageDebt,
	}
	if err := s.putLimit(def); err != nil {
		return err
	}
	s.defs[def.Key] = def
	s.singleKey = def.Key
	return nil
}

// givenDualLimits defines two rolling limits used together in a request.
func (s *rateLimiterState) givenDualLimits(first string, firstCap int, second string, secondCap int) error {
	firstKey := ratelimiter.LimitKey("bdd:" + first)
	secondKey := ratelimiter.LimitKey("bdd:" + second)
	defs := []ratelimiter.LimitDefinition{
		{
			Key:           firstKey,
			Kind:          ratelimiter.KindRolling,
			Capacity:      uint64(firstCap),
			WindowSeconds: 60,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:           secondKey,
			Kind:          ratelimiter.KindRolling,
			Capacity:      uint64(secondCap),
			WindowSeconds: 60,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
	}
	for _, def := range defs {
		if err := s.putLimit(def); err != nil {
			return err
		}
		s.defs[def.Key] = def
	}
	s.keyAliases[first] = firstKey
	s.keyAliases[second] = secondKey
	return nil
}

// reserveAmountForLease reserves against the current single-key limit.
func (s *rateLimiterState) reserveAmountForLease(amount int, leaseID string) error {
	if s.singleKey == "" {
		return fmt.Errorf("no single limit configured")
	}
	req := ratelimiter.ReserveRequest{
		LeaseID: leaseID,
		Requirements: []ratelimiter.Requirement{
			{Key: s.singleKey, Amount: uint64(amount)},
		},
	}
	res, dur, err := s.reserve(req)
	if err != nil {
		return err
	}
	s.lastReserve = res
	s.lastDuration = dur
	s.reserveHistory = append(s.reserveHistory, res)
	return nil
}

// reserveAmountForBothLimits reserves across the paired limits.
func (s *rateLimiterState) reserveAmountForBothLimits(amount int) error {
	firstKey, ok := s.keyAliases["provider"]
	if !ok {
		return fmt.Errorf("provider limit not configured")
	}
	secondKey, ok := s.keyAliases["user"]
	if !ok {
		return fmt.Errorf("user limit not configured")
	}
	req := ratelimiter.ReserveRequest{
		LeaseID: "dual-reserve",
		Requirements: []ratelimiter.Requirement{
			{Key: firstKey, Amount: uint64(amount)},
			{Key: secondKey, Amount: uint64(amount)},
		},
	}
	res, dur, err := s.reserve(req)
	if err != nil {
		return err
	}
	s.lastReserve = res
	s.lastDuration = dur
	return nil
}

// completeLease completes a lease without actuals.
func (s *rateLimiterState) completeLease(leaseID string) error {
	res, dur, err := s.complete(ratelimiter.CompleteRequest{LeaseID: leaseID})
	if err != nil {
		return err
	}
	s.lastComplete = res
	s.lastDuration = dur
	return nil
}

// completeLeaseWithActual completes a lease with actual usage.
func (s *rateLimiterState) completeLeaseWithActual(leaseID string, actual int) error {
	if s.singleKey == "" {
		return fmt.Errorf("no single limit configured")
	}
	res, dur, err := s.complete(ratelimiter.CompleteRequest{
		LeaseID: leaseID,
		Actuals: []ratelimiter.Actual{{Key: s.singleKey, ActualAmount: uint64(actual)}},
	})
	if err != nil {
		return err
	}
	s.lastComplete = res
	s.lastDuration = dur
	return nil
}

// batchReserveTwo submits a batch reserve with two leases.
func (s *rateLimiterState) batchReserveTwo(first, second string, amount int) error {
	if s.singleKey == "" {
		return fmt.Errorf("no single limit configured")
	}
	req := ratelimiter.BatchReserveRequest{
		Requests: []ratelimiter.ReserveRequest{
			{
				LeaseID:      first,
				Requirements: []ratelimiter.Requirement{{Key: s.singleKey, Amount: uint64(amount)}},
			},
			{
				LeaseID:      second,
				Requirements: []ratelimiter.Requirement{{Key: s.singleKey, Amount: uint64(amount)}},
			},
		},
	}
	res, dur, err := s.batchReserve(req)
	if err != nil {
		return err
	}
	s.lastBatch = res
	s.lastDuration = dur
	return nil
}

// thirdReserveDenied asserts the third reserve attempt was denied.
func (s *rateLimiterState) thirdReserveDenied() error {
	if len(s.reserveHistory) < 3 {
		return fmt.Errorf("expected at least 3 reserve attempts")
	}
	if s.reserveHistory[2].Allowed {
		return fmt.Errorf("expected third reserve to be denied")
	}
	return nil
}

// lastReserveDenied asserts the latest reserve attempt was denied.
func (s *rateLimiterState) lastReserveDenied() error {
	if s.lastReserve.Allowed {
		return fmt.Errorf("expected reserve to be denied")
	}
	return nil
}

// limitCapacityRemains asserts a named limit still has the provided capacity.
func (s *rateLimiterState) limitCapacityRemains(alias string, capacity int) error {
	key, ok := s.keyAliases[alias]
	if !ok {
		return fmt.Errorf("unknown alias %s", alias)
	}
	def, ok := s.defs[key]
	if !ok {
		return fmt.Errorf("missing definition for %s", key)
	}
	if def.Capacity != uint64(capacity) {
		return fmt.Errorf("expected capacity %d, got %d", capacity, def.Capacity)
	}
	return nil
}

// responseWithin asserts the last request finished within the threshold.
func (s *rateLimiterState) responseWithin(limitMs int) error {
	if s.lastDuration == 0 {
		return fmt.Errorf("no recorded response time")
	}
	if s.lastDuration > time.Duration(limitMs)*time.Millisecond {
		return fmt.Errorf("response time %s exceeds %dms", s.lastDuration, limitMs)
	}
	return nil
}

// reserveWithin reserves and asserts the response time is within the threshold.
func (s *rateLimiterState) reserveWithin(amount int, leaseID string, limitMs int) error {
	if s.singleKey == "" {
		return fmt.Errorf("no single limit configured")
	}
	req := ratelimiter.ReserveRequest{
		LeaseID: leaseID,
		Requirements: []ratelimiter.Requirement{
			{Key: s.singleKey, Amount: uint64(amount)},
		},
	}
	res, dur, err := s.reserve(req)
	if err != nil {
		return err
	}
	s.lastReserve = res
	s.lastDuration = dur
	if !res.Allowed {
		return fmt.Errorf("expected reserve to be allowed")
	}
	if dur > time.Duration(limitMs)*time.Millisecond {
		return fmt.Errorf("response time %s exceeds %dms", dur, limitMs)
	}
	return nil
}

// debtIs asserts the backend has recorded the expected debt.
func (s *rateLimiterState) debtIs(key string, expected int) error {
	snapshot := s.backend.DebugSnapshot()
	value, ok := snapshot.Debt[ratelimiter.LimitKey(key)]
	if !ok {
		return fmt.Errorf("missing debt entry for %s", key)
	}
	if value != uint64(expected) {
		return fmt.Errorf("expected debt %d, got %d", expected, value)
	}
	return nil
}

// batchResultOrder asserts batch results match the expected allow/deny order.
func (s *rateLimiterState) batchResultOrder(allowedIndex int, deniedIndex int) error {
	if len(s.lastBatch.Results) == 0 {
		return fmt.Errorf("no batch results recorded")
	}
	if allowedIndex < 1 || deniedIndex < 1 || allowedIndex > len(s.lastBatch.Results) || deniedIndex > len(s.lastBatch.Results) {
		return fmt.Errorf("invalid result indices")
	}
	if !s.lastBatch.Results[allowedIndex-1].Allowed {
		return fmt.Errorf("expected result %d allowed", allowedIndex)
	}
	if s.lastBatch.Results[deniedIndex-1].Allowed {
		return fmt.Errorf("expected result %d denied", deniedIndex)
	}
	return nil
}

// adminDecreasesCapacity submits a lower-capacity definition for the key.
func (s *rateLimiterState) adminDecreasesCapacity(key string, newCap int) error {
	def, ok := s.defs[ratelimiter.LimitKey(key)]
	if !ok {
		return fmt.Errorf("missing definition for %s", key)
	}
	def.Capacity = uint64(newCap)
	if err := s.putLimit(def); err != nil {
		return err
	}
	s.defs[def.Key] = def
	s.decreaseKey = def.Key
	return nil
}

// newReservationDeniedWithError asserts a reservation returns the provided error string.
func (s *rateLimiterState) newReservationDeniedWithError(key string, expected string) error {
	req := ratelimiter.ReserveRequest{
		LeaseID: "decrease-check",
		Requirements: []ratelimiter.Requirement{
			{Key: ratelimiter.LimitKey(key), Amount: 1},
		},
	}
	res, dur, err := s.reserve(req)
	if err != nil {
		return err
	}
	s.lastReserve = res
	s.lastDuration = dur
	if res.Allowed {
		return fmt.Errorf("expected reservation to be denied")
	}
	if res.Error != expected {
		return fmt.Errorf("expected error %q, got %q", expected, res.Error)
	}
	return nil
}

// applyDecreaseWhenAvailable advances time and applies a pending decrease.
func (s *rateLimiterState) applyDecreaseWhenAvailable(expected int) error {
	key := s.decreaseKey
	if key == "" {
		key = s.singleKey
	}
	def, ok := s.defs[key]
	if !ok {
		return fmt.Errorf("missing definition for %s", key)
	}
	s.clock.Advance(time.Duration(def.WindowSeconds+1) * time.Second)
	s.backend.TryApplyDecrease(key)
	available := s.backend.DebugSnapshot().Rolling[key].Capacity - s.backend.DebugSnapshot().Rolling[key].Used
	if available < uint64(expected) {
		return fmt.Errorf("expected available >= %d, got %d", expected, available)
	}
	return nil
}

// reservationsAcceptedAgain asserts a new reservation is allowed after decrease.
func (s *rateLimiterState) reservationsAcceptedAgain() error {
	if s.singleKey == "" {
		return fmt.Errorf("no single limit configured")
	}
	req := ratelimiter.ReserveRequest{
		LeaseID: "post-decrease",
		Requirements: []ratelimiter.Requirement{
			{Key: s.singleKey, Amount: 1},
		},
	}
	res, dur, err := s.reserve(req)
	if err != nil {
		return err
	}
	s.lastReserve = res
	s.lastDuration = dur
	if !res.Allowed {
		return fmt.Errorf("expected reservation to be allowed")
	}
	return nil
}

// putLimit sends the admin PUT request.
func (s *rateLimiterState) putLimit(def ratelimiter.LimitDefinition) error {
	payload, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("marshal limit: %w", err)
	}
	body, dur, err := s.doRequest(http.MethodPut, "/v1/admin/limits", payload)
	if err != nil {
		return err
	}
	s.lastDuration = dur
	var resp adminPutResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode admin response: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("admin put returned ok=false")
	}
	return nil
}

// reserve submits a reserve request and captures timing.
func (s *rateLimiterState) reserve(req ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, time.Duration, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.ReserveResponse{}, 0, fmt.Errorf("marshal reserve: %w", err)
	}
	body, dur, err := s.doRequest(http.MethodPost, "/v1/reserve", payload)
	if err != nil {
		return ratelimiter.ReserveResponse{}, dur, err
	}
	var resp ratelimiter.ReserveResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ratelimiter.ReserveResponse{}, dur, fmt.Errorf("decode reserve: %w", err)
	}
	return resp, dur, nil
}

// batchReserve submits a batch reserve request and captures timing.
func (s *rateLimiterState) batchReserve(req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, time.Duration, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.BatchReserveResponse{}, 0, fmt.Errorf("marshal batch reserve: %w", err)
	}
	body, dur, err := s.doRequest(http.MethodPost, "/v1/reserve/batch", payload)
	if err != nil {
		return ratelimiter.BatchReserveResponse{}, dur, err
	}
	var resp ratelimiter.BatchReserveResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ratelimiter.BatchReserveResponse{}, dur, fmt.Errorf("decode batch reserve: %w", err)
	}
	return resp, dur, nil
}

// complete submits a completion request and captures timing.
func (s *rateLimiterState) complete(req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, time.Duration, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.CompleteResponse{}, 0, fmt.Errorf("marshal complete: %w", err)
	}
	body, dur, err := s.doRequest(http.MethodPost, "/v1/complete", payload)
	if err != nil {
		return ratelimiter.CompleteResponse{}, dur, err
	}
	var resp ratelimiter.CompleteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ratelimiter.CompleteResponse{}, dur, fmt.Errorf("decode complete: %w", err)
	}
	return resp, dur, nil
}

// doRequest executes an HTTP request with a JSON payload and returns the body.
func (s *rateLimiterState) doRequest(method, path string, payload []byte) ([]byte, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		return nil, duration, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, duration, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return body, duration, nil
}

// adminPutResponse captures the admin PUT response payload.
type adminPutResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}
