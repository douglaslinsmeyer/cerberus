package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// Meta contains response metadata
type Meta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondSuccess writes a successful JSON response
func respondSuccess(w http.ResponseWriter, data interface{}) {
	resp := SuccessResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	respondJSON(w, http.StatusOK, resp)
}

// respondError writes an error JSON response
func respondError(w http.ResponseWriter, status int, message string) {
	resp := ErrorResponse{
		Success: false,
		Error:   message,
	}
	respondJSON(w, status, resp)
}

// respondCreated writes a 201 Created response
func respondCreated(w http.ResponseWriter, data interface{}) {
	resp := SuccessResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
	respondJSON(w, http.StatusCreated, resp)
}

// respondNoContent writes a 204 No Content response
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
