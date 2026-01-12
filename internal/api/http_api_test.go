package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

type stubBackend struct{}

func (s stubBackend) ApplyDefinition(_ context.Context, _ ratelimiter.LimitDefinition) error {
	return nil
}

func (s stubBackend) Reserve(_ context.Context, _ ratelimiter.ReserveRequest, _ time.Time) (ratelimiter.ReserveResponse, error) {
	return ratelimiter.ReserveResponse{}, nil
}

func (s stubBackend) Complete(_ context.Context, _ ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	return ratelimiter.CompleteResponse{Ok: true}, nil
}

func TestHTTP_AdminPutValidationErrors(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		reg := registry.New()
		srv := httptest.NewServer(NewHandler(Config{Registry: reg, Backend: stubBackend{}}))
		defer srv.Close()

		cases := []struct {
			name    string
			payload ratelimiter.LimitDefinition
		}{
			{name: "missing_key", payload: ratelimiter.LimitDefinition{Kind: ratelimiter.KindRolling, Capacity: 1, WindowSeconds: 60}},
			{name: "invalid_kind", payload: ratelimiter.LimitDefinition{Key: "k", Kind: "nope", Capacity: 1, WindowSeconds: 60}},
			{name: "zero_capacity", payload: ratelimiter.LimitDefinition{Key: "k", Kind: ratelimiter.KindRolling, Capacity: 0, WindowSeconds: 60}},
			{name: "missing_window", payload: ratelimiter.LimitDefinition{Key: "k", Kind: ratelimiter.KindRolling, Capacity: 1, WindowSeconds: 0}},
			{name: "missing_timeout", payload: ratelimiter.LimitDefinition{Key: "k", Kind: ratelimiter.KindConcurrency, Capacity: 1, TimeoutSeconds: 0}},
			{name: "invalid_overage", payload: ratelimiter.LimitDefinition{Key: "k", Kind: ratelimiter.KindRolling, Capacity: 1, WindowSeconds: 60, Overage: "maybe"}},
		}
		for _, tc := range cases {
			tc := tc
			resp, body := doRequestJSON(t, http.MethodPut, srv.URL+"/v1/admin/limits", mustMarshal(t, tc.payload))
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("%s: expected 400, got %d", tc.name, resp.StatusCode)
			}
			var parsed errorResponse
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("%s: parse response: %v", tc.name, err)
			}
			if parsed.Error != "invalid_request" {
				t.Fatalf("%s: expected invalid_request, got %q", tc.name, parsed.Error)
			}
		}
	})
}

func TestHTTP_AdminGetLimitUnknownReturns404(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		reg := registry.New()
		srv := httptest.NewServer(NewHandler(Config{Registry: reg, Backend: stubBackend{}}))
		defer srv.Close()

		resp, _ := doRequestJSON(t, http.MethodGet, srv.URL+"/v1/admin/limits/missing", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})
}

func TestHTTP_AdminPutDecreaseState(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		reg := registry.New()
		path := filepath.Join(t.TempDir(), "limits.json")
		srv := httptest.NewServer(NewHandler(Config{Registry: reg, Backend: stubBackend{}, RegistryPath: path}))
		defer srv.Close()

		base := ratelimiter.LimitDefinition{
			Key:           "global:llm:test:model:tpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      2,
			WindowSeconds: 60,
			Unit:          "tokens",
			Description:   "test",
			Overage:       ratelimiter.OverageDebt,
		}
		resp, body := doRequestJSON(t, http.MethodPut, srv.URL+"/v1/admin/limits", mustMarshal(t, base))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var putResp adminPutResponse
		if err := json.Unmarshal(body, &putResp); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if putResp.Status != string(ratelimiter.LimitStatusActive) {
			t.Fatalf("expected active status, got %q", putResp.Status)
		}

		base.Capacity = 1
		resp, body = doRequestJSON(t, http.MethodPut, srv.URL+"/v1/admin/limits", mustMarshal(t, base))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if err := json.Unmarshal(body, &putResp); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if putResp.Status != string(ratelimiter.LimitStatusDecreasing) {
			t.Fatalf("expected decreasing status, got %q", putResp.Status)
		}

		state, ok := reg.Get(base.Key)
		if !ok {
			t.Fatalf("expected state to exist")
		}
		if state.Status != ratelimiter.LimitStatusDecreasing {
			t.Fatalf("expected decreasing status, got %q", state.Status)
		}
		if state.PendingDecreaseTo != 1 {
			t.Fatalf("expected pending decrease 1, got %d", state.PendingDecreaseTo)
		}
		if state.Definition.Capacity != 2 {
			t.Fatalf("expected definition capacity to remain 2, got %d", state.Definition.Capacity)
		}
	})
}

func doRequestJSON(t *testing.T, method, url string, payload []byte) (*http.Response, []byte) {
	t.Helper()
	ctx := testutil.Context(t, 2*time.Second)
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, data
}

func mustMarshal(t *testing.T, payload ratelimiter.LimitDefinition) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	ctx := testutil.Context(t, timeout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-ctx.Done():
		t.Fatalf("test timed out")
	case <-done:
	}
}
