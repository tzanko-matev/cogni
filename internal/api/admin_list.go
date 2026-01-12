package api

import "net/http"

func (h *handler) handleAdminListLimits(w http.ResponseWriter) {
	states := h.registry.List()
	writeLimitsResponse(w, http.StatusOK, limitsResponse{Limits: states})
}
