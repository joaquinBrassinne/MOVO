package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type payload struct {
	Email string `json:"email"`
}

func decode(body string) (payload, bool, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var got payload
	ok := httpx.DecodeJSON(rec, req, &got)
	return got, ok, rec
}

func TestDecodeJSONReadsAStrictBody(t *testing.T) {
	t.Parallel()

	got, ok, rec := decode(`{"email":"a@b.co"}`)
	if !ok || got.Email != "a@b.co" {
		t.Fatalf("DecodeJSON = (%+v, %v), want the payload and true", got, ok)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("DecodeJSON wrote %q on success", rec.Body.String())
	}
}

func TestDecodeJSONRejectsBadBodiesWith400(t *testing.T) {
	t.Parallel()

	bodies := map[string]string{
		"empty":            ``,
		"not json":         `email=a@b.co`,
		"unknown field":    `{"email":"a@b.co","role":"super_admin"}`,
		"wrong type":       `{"email":5}`,
		"trailing data":    `{"email":"a@b.co"}{"email":"c@d.co"}`,
		"trailing garbage": `{"email":"a@b.co"} x`,
		"oversized":        `{"email":"` + strings.Repeat("a", 70<<10) + `"}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, ok, rec := decode(body)
			if ok || rec.Code != http.StatusBadRequest {
				t.Fatalf("DecodeJSON ok=%v status=%d, want false and 400", ok, rec.Code)
			}
			if got := errorCode(t, rec); got != httpx.CodeInvalidRequest {
				t.Errorf("error code = %q, want %q", got, httpx.CodeInvalidRequest)
			}
		})
	}
}

func TestNewRouterMountsModulesUnderAPIV1(t *testing.T) {
	t.Parallel()

	mount := func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(w, http.StatusOK, map[string]string{"pong": "yes"})
		})
	}
	router := httpx.NewRouter(nil, nil, mount)

	for path, want := range map[string]int{"/api/v1/ping": http.StatusOK, "/ping": http.StatusNotFound, "/healthz": http.StatusOK} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, want)
		}
	}
}

// TestUnmatchedAPIRoutesKeepTheErrorEnvelope pins that the versioned subrouter
// answers like the rest of the API, instead of chi's plain-text defaults.
func TestUnmatchedAPIRoutesKeepTheErrorEnvelope(t *testing.T) {
	t.Parallel()

	mount := func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	}
	router := httpx.NewRouter(nil, nil, mount)

	tests := []struct {
		method, path string
		wantStatus   int
		wantCode     string
	}{
		{http.MethodGet, "/api/v1/missing", http.StatusNotFound, httpx.CodeNotFound},
		{http.MethodPost, "/api/v1/ping", http.StatusMethodNotAllowed, httpx.CodeMethodNotAllowed},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

		if rec.Code != tt.wantStatus {
			t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, rec.Code, tt.wantStatus)
		}
		if got := errorCode(t, rec); got != tt.wantCode {
			t.Errorf("%s %s error code = %q, want %q", tt.method, tt.path, got, tt.wantCode)
		}
	}
}
