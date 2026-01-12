package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"cogni/pkg/ratelimiter"
)

type adminPutResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

func (h *handler) handleAdminPutLimit(w http.ResponseWriter, r *http.Request) {
	def, err := decodeLimitDefinition(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	state := h.registry.NextState(def)
	if h.backend != nil {
		if err := h.backend.ApplyDefinition(r.Context(), def); err != nil {
			writeError(w, http.StatusInternalServerError, "backend_error")
			return
		}
	}
	h.registry.Put(state)
	if h.registryPath != "" {
		if err := h.registry.Save(h.registryPath); err != nil {
			writeError(w, http.StatusInternalServerError, "backend_error")
			return
		}
	}
	writeAdminPutResponse(w, http.StatusOK, adminPutResponse{OK: true, Status: string(state.Status)})
}

func decodeLimitDefinition(r *http.Request) (ratelimiter.LimitDefinition, error) {
	var def ratelimiter.LimitDefinition
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&def); err != nil {
		return ratelimiter.LimitDefinition{}, err
	}
	def.Key = ratelimiter.LimitKey(strings.TrimSpace(string(def.Key)))
	def.Unit = strings.TrimSpace(def.Unit)
	def.Description = strings.TrimSpace(def.Description)
	def.Kind = ratelimiter.LimitKind(strings.TrimSpace(strings.ToLower(string(def.Kind))))
	def.Overage = ratelimiter.OveragePolicy(strings.TrimSpace(strings.ToLower(string(def.Overage))))
	if def.Overage == "" {
		def.Overage = ratelimiter.OverageDebt
	}
	if err := validateLimitDefinition(def); err != nil {
		return ratelimiter.LimitDefinition{}, err
	}
	return def, nil
}

func validateLimitDefinition(def ratelimiter.LimitDefinition) error {
	if strings.TrimSpace(string(def.Key)) == "" {
		return errInvalidDefinition
	}
	if def.Capacity == 0 {
		return errInvalidDefinition
	}
	switch def.Kind {
	case ratelimiter.KindRolling:
		if def.WindowSeconds <= 0 || def.TimeoutSeconds != 0 {
			return errInvalidDefinition
		}
	case ratelimiter.KindConcurrency:
		if def.TimeoutSeconds <= 0 || def.WindowSeconds != 0 {
			return errInvalidDefinition
		}
	default:
		return errInvalidDefinition
	}
	switch def.Overage {
	case ratelimiter.OverageDeny, ratelimiter.OverageDebt:
		return nil
	default:
		return errInvalidDefinition
	}
}
