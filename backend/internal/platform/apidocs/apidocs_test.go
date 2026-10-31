package apidocs_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/apidocs"
)

func TestMountServesTheDocsAndPassesEverythingElseThrough(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := apidocs.Mount([]byte("openapi: 3.0.3\n"), next)

	for _, c := range []struct {
		name, method, path string
		status             int
		contentType, body  string
	}{
		{"page", http.MethodGet, apidocs.PagePath, http.StatusOK, "text/html", "SwaggerUIBundle"},
		{"spec", http.MethodGet, apidocs.SpecPath, http.StatusOK, "application/yaml", "openapi: 3.0.3"},
		{"api route", http.MethodGet, "/api/v1/plans", http.StatusTeapot, "", ""},
		{"probe", http.MethodGet, "/healthz", http.StatusTeapot, "", ""},
		{"docs are read only", http.MethodPost, apidocs.PagePath, http.StatusTeapot, "", ""},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), c.method, c.path, nil))

		if rec.Code != c.status {
			t.Errorf("%s: status %d, want %d", c.name, rec.Code, c.status)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, c.contentType) {
			t.Errorf("%s: content type %q, want prefix %q", c.name, got, c.contentType)
		}
		if !strings.Contains(rec.Body.String(), c.body) {
			t.Errorf("%s: body %q does not contain %q", c.name, rec.Body.String(), c.body)
		}
	}
}
