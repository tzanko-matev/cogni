package testutil

import (
	"net/http/httptest"
	"testing"
	"time"

	"cogni/internal/api"
	"cogni/internal/backend"
	"cogni/internal/registry"
)

// ServerConfig wires dependencies for StartServer.
type ServerConfig struct {
	Registry     *registry.Registry
	Backend      backend.Backend
	RegistryPath string
	Now          func() time.Time
}

// ServerInstance represents a running HTTP test server.
type ServerInstance struct {
	BaseURL string
	Close   func()
}

// StartServer launches an in-memory HTTP server for the rate limiter API.
func StartServer(t *testing.T, cfg ServerConfig) *ServerInstance {
	t.Helper()
	if cfg.Registry == nil {
		cfg.Registry = registry.New()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	handler := api.NewHandler(api.Config{
		Registry:     cfg.Registry,
		Backend:      cfg.Backend,
		RegistryPath: cfg.RegistryPath,
		Now:          cfg.Now,
	})
	server := httptest.NewServer(handler)
	return &ServerInstance{
		BaseURL: server.URL,
		Close:   server.Close,
	}
}
