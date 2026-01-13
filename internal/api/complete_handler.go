package api

import (
	"encoding/json"
	"net/http"

	"cogni/pkg/ratelimiter"
)

func (h *handler) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.backend == nil {
		writeError(w, http.StatusInternalServerError, "backend_error")
		return
	}
	var req ratelimiter.CompleteRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if req.LeaseID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	res, err := h.backend.Complete(r.Context(), req)
	if err != nil {
		writeCompleteResponse(w, http.StatusOK, ratelimiter.CompleteResponse{Ok: false, Error: "backend_error"})
		return
	}
	writeCompleteResponse(w, http.StatusOK, res)
}

func (h *handler) handleBatchComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.backend == nil {
		writeError(w, http.StatusInternalServerError, "backend_error")
		return
	}
	var req ratelimiter.BatchCompleteRequest
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

	results := make([]ratelimiter.BatchCompleteResult, 0, len(req.Requests))
	for _, item := range req.Requests {
		if item.LeaseID == "" {
			results = append(results, ratelimiter.BatchCompleteResult{Ok: false, Error: invalidRequestError})
			continue
		}
		res, err := h.backend.Complete(r.Context(), item)
		if err != nil {
			results = append(results, ratelimiter.BatchCompleteResult{Ok: false, Error: "backend_error"})
			continue
		}
		results = append(results, ratelimiter.BatchCompleteResult{Ok: res.Ok, Error: res.Error})
	}

	writeBatchCompleteResponse(w, http.StatusOK, ratelimiter.BatchCompleteResponse{Results: results})
}
