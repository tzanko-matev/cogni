package api

import (
	"net/http"
	"strings"
	"time"

	"cogni/internal/backend"
	"cogni/internal/registry"
)

// Config wires dependencies for the HTTP handler.
type Config struct {
	Registry     *registry.Registry
	Backend      backend.Backend
	RegistryPath string
	Now          func() time.Time
}

// NewHandler builds an HTTP handler for the rate limiter API.
func NewHandler(cfg Config) http.Handler {
	h := &handler{
		registry:     cfg.Registry,
		backend:      cfg.Backend,
		registryPath: cfg.RegistryPath,
		nowFn:        cfg.Now,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/admin/limits", h.handleAdminLimits)
	mux.HandleFunc("/v1/admin/limits/", h.handleAdminLimitByKey)
	mux.HandleFunc("/v1/reserve", h.handleReserve)
	mux.HandleFunc("/v1/reserve/batch", h.handleBatchReserve)
	mux.HandleFunc("/v1/complete", h.handleComplete)
	mux.HandleFunc("/v1/complete/batch", h.handleBatchComplete)
	return mux
}

type handler struct {
	registry     *registry.Registry
	backend      backend.Backend
	registryPath string
	nowFn        func() time.Time
}

func (h *handler) handleAdminLimits(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil {
		writeError(w, http.StatusInternalServerError, "backend_error")
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.handleAdminPutLimit(w, r)
	case http.MethodGet:
		h.handleAdminListLimits(w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *handler) handleAdminLimitByKey(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil {
		writeError(w, http.StatusInternalServerError, "backend_error")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/v1/admin/limits/")
	key = strings.TrimSpace(key)
	if key == "" {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	state, ok := h.registry.Get(ratelimiterKey(key))
	if !ok {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	writeLimitResponse(w, http.StatusOK, limitResponse{Limit: state})
}
