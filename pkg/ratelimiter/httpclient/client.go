package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cogni/pkg/ratelimiter"
)

// Client implements Limiter against a remote ratelimiterd server.
type Client struct {
	baseURL string
	client  *http.Client
}

// New constructs a client for the given base URL.
func New(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{}}
}

// NewWithTimeout constructs a client for the given base URL with a request timeout.
func NewWithTimeout(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

// Reserve requests a reservation over HTTP.
func (c *Client) Reserve(ctx context.Context, req ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.ReserveResponse{}, err
	}
	body, status, err := c.post(ctx, "/v1/reserve", payload)
	if err != nil {
		return ratelimiter.ReserveResponse{}, err
	}
	if status != http.StatusOK {
		return ratelimiter.ReserveResponse{}, decodeHTTPError(status, body)
	}
	var res ratelimiter.ReserveResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ratelimiter.ReserveResponse{}, err
	}
	return res, nil
}

// Complete reports completion over HTTP.
func (c *Client) Complete(ctx context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.CompleteResponse{}, err
	}
	body, status, err := c.post(ctx, "/v1/complete", payload)
	if err != nil {
		return ratelimiter.CompleteResponse{}, err
	}
	if status != http.StatusOK {
		return ratelimiter.CompleteResponse{}, decodeHTTPError(status, body)
	}
	var res ratelimiter.CompleteResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ratelimiter.CompleteResponse{}, err
	}
	return res, nil
}

// BatchReserve sends a batch reserve request over HTTP.
func (c *Client) BatchReserve(ctx context.Context, req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.BatchReserveResponse{}, err
	}
	body, status, err := c.post(ctx, "/v1/reserve/batch", payload)
	if err != nil {
		return ratelimiter.BatchReserveResponse{}, err
	}
	if status != http.StatusOK {
		return ratelimiter.BatchReserveResponse{}, decodeHTTPError(status, body)
	}
	var res ratelimiter.BatchReserveResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ratelimiter.BatchReserveResponse{}, err
	}
	return res, nil
}

// BatchComplete sends a batch complete request over HTTP.
func (c *Client) BatchComplete(ctx context.Context, req ratelimiter.BatchCompleteRequest) (ratelimiter.BatchCompleteResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ratelimiter.BatchCompleteResponse{}, err
	}
	body, status, err := c.post(ctx, "/v1/complete/batch", payload)
	if err != nil {
		return ratelimiter.BatchCompleteResponse{}, err
	}
	if status != http.StatusOK {
		return ratelimiter.BatchCompleteResponse{}, decodeHTTPError(status, body)
	}
	var res ratelimiter.BatchCompleteResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ratelimiter.BatchCompleteResponse{}, err
	}
	return res, nil
}

func (c *Client) post(ctx context.Context, path string, payload []byte) ([]byte, int, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

type errorResponse struct {
	Error string `json:"error"`
}

func decodeHTTPError(status int, body []byte) error {
	var resp errorResponse
	if err := json.Unmarshal(body, &resp); err == nil && resp.Error != "" {
		return fmt.Errorf("http %d: %s", status, resp.Error)
	}
	return fmt.Errorf("http %d", status)
}
