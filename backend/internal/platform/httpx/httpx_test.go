package httpx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// errorEnvelope mirrors the JSON shape every failure response must use.
type errorEnvelope struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func serve(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()

	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode error envelope from %q: %v", body, err)
	}
	return env
}

// TestHealthzReportsLiveness pins that liveness only claims the process serves
// HTTP, never that a dependency is reachable.
func TestHealthzReportsLiveness(t *testing.T) {
	t.Parallel()

	rec := serve(t, httpx.NewRouter(discardLogger()), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json"; !strings.HasPrefix(got, want) {
		t.Errorf("Content-Type = %q, want it to start with %q", got, want)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want %q", body.Status, "ok")
	}
}

// TestUnknownRouteUsesErrorEnvelope pins the uniform format outside the routes.
func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	t.Parallel()

	rec := serve(t, httpx.NewRouter(discardLogger()), httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if env := decodeError(t, rec.Body.Bytes()); env.Error.Code != httpx.CodeNotFound {
		t.Errorf("error code = %q, want %q", env.Error.Code, httpx.CodeNotFound)
	}
}

// TestMethodNotAllowedUsesErrorEnvelope pins the 405 shape.
func TestMethodNotAllowedUsesErrorEnvelope(t *testing.T) {
	t.Parallel()

	rec := serve(t, httpx.NewRouter(discardLogger()), httptest.NewRequest(http.MethodDelete, "/healthz", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /healthz status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if env := decodeError(t, rec.Body.Bytes()); env.Error.Code != httpx.CodeMethodNotAllowed {
		t.Errorf("error code = %q, want %q", env.Error.Code, httpx.CodeMethodNotAllowed)
	}
}

// TestErrorEnvelopeCarriesDetails pins that per-field context travels with the
// error, which is what the frontend needs to mark inputs.
func TestErrorEnvelopeCarriesDetails(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httpx.WriteError(rec, http.StatusBadRequest, "validation_failed", "invalid input",
		map[string]any{"email": "is required"})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if env := decodeError(t, rec.Body.Bytes()); env.Error.Details["email"] != "is required" {
		t.Errorf("details.email = %v, want %q", env.Error.Details["email"], "is required")
	}
}

// TestErrorEnvelopeOmitsEmptyDetails pins that a generic error carries no empty
// details member.
func TestErrorEnvelopeOmitsEmptyDetails(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httpx.WriteError(rec, http.StatusNotFound, httpx.CodeNotFound, "resource not found", nil)

	if body := rec.Body.String(); strings.Contains(body, `"details"`) {
		t.Errorf("body %q carries an empty details member, want it omitted", body)
	}
}

// TestRequestIDIsEchoed pins that an upstream request id is preserved.
func TestRequestIDIsEchoed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(httpx.RequestIDHeader, "abc-123")
	rec := serve(t, httpx.NewRouter(discardLogger()), req)

	if got := rec.Header().Get(httpx.RequestIDHeader); got != "abc-123" {
		t.Errorf("%s = %q, want %q", httpx.RequestIDHeader, got, "abc-123")
	}
}

// TestRequestIDIsGenerated pins that every request gets a fresh id.
func TestRequestIDIsGenerated(t *testing.T) {
	t.Parallel()

	router := httpx.NewRouter(discardLogger())
	first := serve(t, router, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	second := serve(t, router, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	firstID := first.Header().Get(httpx.RequestIDHeader)
	secondID := second.Header().Get(httpx.RequestIDHeader)

	if firstID == "" || secondID == "" {
		t.Fatalf("generated request ids are empty: %q, %q", firstID, secondID)
	}
	if firstID == secondID {
		t.Errorf("two requests share the request id %q, want distinct ids", firstID)
	}
}

// TestUnusableRequestIDIsReplaced pins that a hostile header value never reaches
// the response or the logs.
func TestUnusableRequestIDIsReplaced(t *testing.T) {
	t.Parallel()

	router := httpx.NewRouter(discardLogger())
	rejected := map[string]string{
		"too long":     strings.Repeat("a", 200),
		"control char": "abc\ndef",
	}

	for name, value := range rejected {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.Header.Set(httpx.RequestIDHeader, value)
			rec := serve(t, router, req)

			got := rec.Header().Get(httpx.RequestIDHeader)
			if got == value {
				t.Errorf("%s = the rejected value %q, want a generated one", httpx.RequestIDHeader, got)
			}
			if got == "" {
				t.Errorf("%s is empty, want a generated id", httpx.RequestIDHeader)
			}
		})
	}
}

// TestRequestIsLoggedWithRequestID pins that structured logs carry the id.
func TestRequestIsLoggedWithRequestID(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(httpx.RequestIDHeader, "log-me")
	serve(t, httpx.NewRouter(logger), req)

	entry := logs.String()
	for _, want := range []string{`"request_id":"log-me"`, `"path":"/healthz"`, `"status":200`} {
		if !strings.Contains(entry, want) {
			t.Errorf("log entry %q does not contain %s", entry, want)
		}
	}
}
