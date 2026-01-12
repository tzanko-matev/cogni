package api

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeErrorResponse(w, status, errorResponse{Error: code})
}

func writeErrorResponse(w http.ResponseWriter, status int, payload errorResponse) {
	writeBytes(w, status, mustJSONError(payload))
}

func writeLimitResponse(w http.ResponseWriter, status int, payload limitResponse) {
	writeBytes(w, status, mustJSONLimit(payload))
}

func writeLimitsResponse(w http.ResponseWriter, status int, payload limitsResponse) {
	writeBytes(w, status, mustJSONLimits(payload))
}

func writeAdminPutResponse(w http.ResponseWriter, status int, payload adminPutResponse) {
	writeBytes(w, status, mustJSONAdminPut(payload))
}

func writeBytes(w http.ResponseWriter, status int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func mustJSONError(payload errorResponse) []byte {
	data, _ := json.Marshal(payload)
	return data
}

func mustJSONLimit(payload limitResponse) []byte {
	data, _ := json.Marshal(payload)
	return data
}

func mustJSONLimits(payload limitsResponse) []byte {
	data, _ := json.Marshal(payload)
	return data
}

func mustJSONAdminPut(payload adminPutResponse) []byte {
	data, _ := json.Marshal(payload)
	return data
}
