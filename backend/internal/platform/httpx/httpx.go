// Package httpx builds the HTTP surface shared by the MOVO services: the chi
// router and the JSON error envelope every endpoint returns on failure.
package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Codes carried by the JSON error envelope.
const (
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
)

// ErrorEnvelope is the JSON body of every failure response.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the machine readable code, a human readable message, and the
// optional per-field context the frontend needs. It never carries internal
// error text: that goes to the logs.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// statusBody is the JSON body of the probes.
type statusBody struct {
	Status string `json:"status"`
}

// NewRouter builds the API router.
func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/healthz", handleHealthz)
	router.NotFound(handleNotFound)
	router.MethodNotAllowed(handleMethodNotAllowed)

	return router
}

// WriteError writes the JSON error envelope. Details is omitted when nil.
func WriteError(w http.ResponseWriter, status int, code, message string, details any) {
	writeJSON(w, status, ErrorEnvelope{Error: ErrorDetail{Code: code, Message: message, Details: details}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// The status line is already sent, so an encoding failure can no longer be
	// reported to the client.
	_ = json.NewEncoder(w).Encode(body)
}

// handleHealthz answers liveness: the process serves HTTP, nothing else.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusBody{Status: "ok"})
}

func handleNotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, CodeNotFound, "resource not found", nil)
}

func handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed", nil)
}
