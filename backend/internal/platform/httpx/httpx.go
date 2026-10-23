// Package httpx builds the HTTP surface shared by the MOVO services: the chi
// router, request-scoped structured logging with a correlation id, and the JSON
// error envelope every endpoint returns on failure.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// RequestIDHeader carries the correlation id of a request: the value is echoed
// when the caller supplies a usable one and generated otherwise.
const RequestIDHeader = "X-Request-Id"

// Codes carried by the JSON error envelope.
const (
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
)

// maxRequestIDLen caps an incoming correlation id so a hostile header value
// cannot flood the logs.
const maxRequestIDLen = 128

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

type requestIDKey struct{}

// RequestID returns the correlation id of the request in ctx, or "" when the
// request did not go through NewRouter.
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// NewRouter builds the API router. A nil logger discards request logs.
func NewRouter(log *slog.Logger) http.Handler {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	router := chi.NewRouter()
	router.Use(withRequestID)
	router.Use(withRequestLogging(log))
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
	// reported to the client; request logging has already captured the status.
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

// withRequestID resolves the correlation id, exposes it in the context and the
// response headers, and keeps it safe for headers and logs.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := usableRequestID(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = newRequestID()
		}

		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// usableRequestID reports the value as-is when it is short, printable ASCII, and
// empty otherwise so the caller generates a fresh id.
func usableRequestID(raw string) string {
	id := strings.TrimSpace(raw)
	if id == "" || len(id) > maxRequestIDLen {
		return ""
	}

	for _, char := range id {
		if char < ' ' || char > '~' {
			return ""
		}
	}

	return id
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand does not fail in practice; the timestamp keeps the id
		// unique enough to correlate a request instead of dropping it.
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	return hex.EncodeToString(buf[:])
}

// statusRecorder remembers the status code so the log line can report it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}

	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	return r.ResponseWriter.Write(body)
}

// withRequestLogging emits one structured line per request.
func withRequestLogging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			log.Info("http request",
				"request_id", RequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
