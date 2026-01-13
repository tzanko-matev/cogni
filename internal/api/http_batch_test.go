package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cogni/internal/backend/memory"
	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func TestHTTP_BatchReserve_OrderAndPerItemErrors(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		reg := registry.New()
		backend := memory.New(clock)
		def := ratelimiter.LimitDefinition{
			Key:           "k1",
			Kind:          ratelimiter.KindRolling,
			Capacity:      1,
			WindowSeconds: 60,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		}
		applyCtx := testutil.Context(t, time.Second)
		if err := backend.ApplyDefinition(applyCtx, def); err != nil {
			t.Fatalf("apply definition: %v", err)
		}
		reg.Put(reg.NextState(def))

		srv := httptest.NewServer(NewHandler(Config{Registry: reg, Backend: backend, Now: clock.Now}))
		defer srv.Close()

		batch := ratelimiter.BatchReserveRequest{Requests: []ratelimiter.ReserveRequest{
			{LeaseID: "A", Requirements: []ratelimiter.Requirement{{Key: "k1", Amount: 1}}},
			{LeaseID: "", Requirements: []ratelimiter.Requirement{{Key: "k1", Amount: 1}}},
			{LeaseID: "B", Requirements: []ratelimiter.Requirement{{Key: "k1", Amount: 1}}},
		}}
		payload, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("marshal batch: %v", err)
		}
		resp, body := doRequestJSON(t, http.MethodPost, srv.URL+"/v1/reserve/batch", payload)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var parsed ratelimiter.BatchReserveResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if len(parsed.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(parsed.Results))
		}
		if !parsed.Results[0].Allowed {
			t.Fatalf("expected first allowed, got %+v", parsed.Results[0])
		}
		if parsed.Results[1].Error != invalidRequestError {
			t.Fatalf("expected invalid_request for second, got %+v", parsed.Results[1])
		}
		if parsed.Results[2].Allowed {
			t.Fatalf("expected third denied, got %+v", parsed.Results[2])
		}
	})
}
