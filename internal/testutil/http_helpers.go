package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"cogni/pkg/ratelimiter"
)

type adminPutResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

type limitResponse struct {
	Limit ratelimiter.LimitState `json:"limit"`
}

type limitsResponse struct {
	Limits []ratelimiter.LimitState `json:"limits"`
}

// HTTPPutLimit sends a PUT /v1/admin/limits request.
func HTTPPutLimit(t testing.TB, baseURL string, def ratelimiter.LimitDefinition) string {
	t.Helper()
	var resp adminPutResponse
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("marshal limit definition: %v", err)
	}
	body := doRequest(t, http.MethodPut, baseURL+"/v1/admin/limits", data)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode admin put response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("admin put returned ok=false")
	}
	return resp.Status
}

// HTTPGetLimit sends a GET /v1/admin/limits/{key} request.
func HTTPGetLimit(t testing.TB, baseURL string, key string) ratelimiter.LimitState {
	t.Helper()
	var resp limitResponse
	body := doRequest(t, http.MethodGet, baseURL+"/v1/admin/limits/"+key, nil)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode limit response: %v", err)
	}
	return resp.Limit
}

// HTTPListLimits sends a GET /v1/admin/limits request.
func HTTPListLimits(t testing.TB, baseURL string) []ratelimiter.LimitState {
	t.Helper()
	var resp limitsResponse
	body := doRequest(t, http.MethodGet, baseURL+"/v1/admin/limits", nil)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode limits response: %v", err)
	}
	return resp.Limits
}

// HTTPReserve sends a POST /v1/reserve request.
func HTTPReserve(t testing.TB, baseURL string, req ratelimiter.ReserveRequest) ratelimiter.ReserveResponse {
	t.Helper()
	var resp ratelimiter.ReserveResponse
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal reserve request: %v", err)
	}
	body := doRequest(t, http.MethodPost, baseURL+"/v1/reserve", data)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode reserve response: %v", err)
	}
	return resp
}

// HTTPComplete sends a POST /v1/complete request.
func HTTPComplete(t testing.TB, baseURL string, req ratelimiter.CompleteRequest) ratelimiter.CompleteResponse {
	t.Helper()
	var resp ratelimiter.CompleteResponse
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal complete request: %v", err)
	}
	body := doRequest(t, http.MethodPost, baseURL+"/v1/complete", data)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	return resp
}

// HTTPBatchReserve sends a POST /v1/reserve/batch request.
func HTTPBatchReserve(t testing.TB, baseURL string, req ratelimiter.BatchReserveRequest) ratelimiter.BatchReserveResponse {
	t.Helper()
	var resp ratelimiter.BatchReserveResponse
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal batch reserve: %v", err)
	}
	body := doRequest(t, http.MethodPost, baseURL+"/v1/reserve/batch", data)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode batch reserve response: %v", err)
	}
	return resp
}

// HTTPBatchComplete sends a POST /v1/complete/batch request.
func HTTPBatchComplete(t testing.TB, baseURL string, req ratelimiter.BatchCompleteRequest) ratelimiter.BatchCompleteResponse {
	t.Helper()
	var resp ratelimiter.BatchCompleteResponse
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal batch complete: %v", err)
	}
	body := doRequest(t, http.MethodPost, baseURL+"/v1/complete/batch", data)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode batch complete response: %v", err)
	}
	return resp
}

// doRequest executes an HTTP request with a JSON payload and returns the body.
func doRequest(t testing.TB, method, url string, payload []byte) []byte {
	t.Helper()
	ctx := Context(t, 2*time.Second)
	reader := bytes.NewReader(payload)
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("unexpected status %d for %s %s: %s", resp.StatusCode, method, url, string(body))
	}
	return body
}
