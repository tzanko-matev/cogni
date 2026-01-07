package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// defaultOpenRouterBaseURL is the default OpenRouter API base URL.
const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

// HTTPDoer abstracts HTTP clients used by providers.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// OpenRouterProvider implements Provider for the OpenRouter API.
type OpenRouterProvider struct {
	APIKey  string
	BaseURL string
	Client  HTTPDoer
	Model   string
}

// ProviderFromEnv builds a provider using environment configuration.
func ProviderFromEnv(provider, model string, client HTTPDoer) (Provider, error) {
	if provider == "" {
		provider = strings.TrimSpace(os.Getenv("LLM_PROVIDER"))
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if provider != "openrouter" {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required")
	}
	return NewOpenRouterProvider(model, apiKey, "", client)
}

// NewOpenRouterProvider constructs an OpenRouter provider with explicit settings.
func NewOpenRouterProvider(model, apiKey, baseURL string, client HTTPDoer) (*OpenRouterProvider, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &OpenRouterProvider{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  client,
		Model:   model,
	}, nil
}

// Stream sends a prompt to OpenRouter and returns a stream of events.
func (p *OpenRouterProvider) Stream(ctx context.Context, prompt Prompt) (Stream, error) {
	messages, err := buildOpenRouterMessages(prompt)
	if err != nil {
		return nil, err
	}
	requestBody := openRouterRequest{
		Model:    p.Model,
		Stream:   true,
		Messages: messages,
	}
	if len(prompt.Tools) > 0 {
		requestBody.Tools = buildOpenRouterTools(prompt.Tools)
		requestBody.ToolChoice = "auto"
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := p.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter error: %s", strings.TrimSpace(string(body)))
	}

	events, err := parseOpenRouterStream(resp.Body)
	if err != nil {
		return nil, err
	}
	return &staticStream{events: events}, nil
}
