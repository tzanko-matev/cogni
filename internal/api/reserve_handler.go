package api

import (
	"encoding/json"
	"net/http"
	"time"

	"cogni/pkg/ratelimiter"
)

func (h *handler) handleReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.backend == nil {
		writeError(w, http.StatusInternalServerError, "backend_error")
		return
	}
	var req ratelimiter.ReserveRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	result := h.validateReserve(req)
	switch result.status {
	case validationInvalid:
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	case validationDenied:
		writeReserveResponse(w, http.StatusOK, result.response)
		return
	}

	res, err := h.backend.Reserve(r.Context(), req, h.now())
	if err != nil {
		writeReserveResponse(w, http.StatusOK, ratelimiter.ReserveResponse{Allowed: false, Error: "backend_error"})
		return
	}
	writeReserveResponse(w, http.StatusOK, res)
}

func (h *handler) handleBatchReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.backend == nil {
		writeError(w, http.StatusInternalServerError, "backend_error")
		return
	}
	var req ratelimiter.BatchReserveRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if len(req.Requests) == 0 || len(req.Requests) > maxBatchRequests {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	results := make([]ratelimiter.BatchReserveResult, 0, len(req.Requests))
	now := h.now()
	for _, item := range req.Requests {
		validation := h.validateReserve(item)
		switch validation.status {
		case validationInvalid:
			results = append(results, ratelimiter.BatchReserveResult{Allowed: false, Error: invalidRequestError})
			continue
		case validationDenied:
			results = append(results, ratelimiter.BatchReserveResult{
				Allowed:        validation.response.Allowed,
				RetryAfterMs:   validation.response.RetryAfterMs,
				ReservedAtUnix: validation.response.ReservedAtUnixMs,
				Error:          validation.response.Error,
			})
			continue
		}

		res, err := h.backend.Reserve(r.Context(), item, now)
		if err != nil {
			results = append(results, ratelimiter.BatchReserveResult{Allowed: false, Error: "backend_error"})
			continue
		}
		results = append(results, ratelimiter.BatchReserveResult{
			Allowed:        res.Allowed,
			RetryAfterMs:   res.RetryAfterMs,
			ReservedAtUnix: res.ReservedAtUnixMs,
			Error:          res.Error,
		})
	}

	writeBatchReserveResponse(w, http.StatusOK, ratelimiter.BatchReserveResponse{Results: results})
}

func (h *handler) now() time.Time {
	if h.nowFn != nil {
		return h.nowFn()
	}
	return time.Now()
}
