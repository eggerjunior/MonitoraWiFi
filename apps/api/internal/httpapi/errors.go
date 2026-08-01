package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type apiError struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

func writeError(w http.ResponseWriter, correlationID string, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{
		Error:         code,
		Message:       message,
		CorrelationID: correlationID,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func newCorrelationID() string {
	return uuid.NewString()
}
