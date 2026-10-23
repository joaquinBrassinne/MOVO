package httpx_test

import (
	"encoding/json"
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

	rec := serve(t, httpx.NewRouter(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

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

	rec := serve(t, httpx.NewRouter(), httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

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

	rec := serve(t, httpx.NewRouter(), httptest.NewRequest(http.MethodDelete, "/healthz", nil))

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
