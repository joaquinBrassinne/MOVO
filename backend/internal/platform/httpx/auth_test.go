package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type stubVerifier map[string]httpx.Principal

func (s stubVerifier) Verify(token string) (httpx.Principal, error) {
	if principal, ok := s[token]; ok {
		return principal, nil
	}
	return httpx.Principal{}, errors.New("bad token")
}

var verifier = stubVerifier{
	"owner-token": {UserID: 7, Role: "business_admin"},
	"admin-token": {UserID: 1, Role: "super_admin"},
}

// protected answers 200 and reports who the handler saw.
func protected(middleware ...func(http.Handler) http.Handler) (http.Handler, *int) {
	calls := 0
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		principal, _ := httpx.PrincipalFrom(r.Context())
		_ = json.NewEncoder(w).Encode(principal)
	})
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler, &calls
}

func callWith(handler http.Handler, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var envelope httpx.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("response is not the error envelope: %v", err)
	}
	return envelope.Error.Code
}

func TestAuthenticateAcceptsAValidBearerToken(t *testing.T) {
	t.Parallel()

	for _, header := range []string{"Bearer owner-token", "bearer owner-token", "BEARER   owner-token"} {
		handler, calls := protected(httpx.Authenticate(verifier))

		rec := callWith(handler, header)
		if rec.Code != http.StatusOK || *calls != 1 {
			t.Fatalf("%q: status %d, handler calls %d, want 200 and 1", header, rec.Code, *calls)
		}

		var principal httpx.Principal
		if err := json.NewDecoder(rec.Body).Decode(&principal); err != nil {
			t.Fatalf("decode principal: %v", err)
		}
		if principal != (httpx.Principal{UserID: 7, Role: "business_admin"}) {
			t.Errorf("%q: principal = %+v, want the owner", header, principal)
		}
	}
}

func TestAuthenticateRejectsEverythingElseWith401(t *testing.T) {
	t.Parallel()

	headers := map[string]string{
		"no header":      "",
		"basic scheme":   "Basic owner-token",
		"no scheme":      "owner-token",
		"empty token":    "Bearer ",
		"unknown token":  "Bearer forged",
		"extra segments": "Bearer owner-token extra",
	}
	for name, header := range headers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			handler, calls := protected(httpx.Authenticate(verifier))

			rec := callWith(handler, header)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if *calls != 0 {
				t.Error("the protected handler ran for an unauthenticated request")
			}
			if got := errorCode(t, rec); got != httpx.CodeUnauthorized {
				t.Errorf("error code = %q, want %q", got, httpx.CodeUnauthorized)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", got)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		allow      []string
		header     string
		wantStatus int
		wantCode   string
	}{
		{"allowed role", []string{"business_admin"}, "Bearer owner-token", http.StatusOK, ""},
		{"one of several roles", []string{"business_admin", "super_admin"}, "Bearer admin-token", http.StatusOK, ""},
		{"owner on an admin route", []string{"super_admin"}, "Bearer owner-token", http.StatusForbidden, httpx.CodeForbidden},
		{"admin on an owner route", []string{"business_admin"}, "Bearer admin-token", http.StatusForbidden, httpx.CodeForbidden},
		{"anonymous", []string{"super_admin"}, "", http.StatusUnauthorized, httpx.CodeUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, calls := protected(httpx.Authenticate(verifier), httpx.RequireRole(tt.allow...))

			rec := callWith(handler, tt.header)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				if *calls != 0 {
					t.Error("the protected handler ran for a refused request")
				}
				if got := errorCode(t, rec); got != tt.wantCode {
					t.Errorf("error code = %q, want %q", got, tt.wantCode)
				}
			}
		})
	}
}

func TestRequireRoleWithoutAuthenticateFailsClosed(t *testing.T) {
	t.Parallel()
	handler, calls := protected(httpx.RequireRole("super_admin"))

	rec := callWith(handler, "Bearer admin-token")
	if rec.Code != http.StatusUnauthorized || *calls != 0 {
		t.Errorf("status %d, handler calls %d, want 401 and 0 when Authenticate is missing", rec.Code, *calls)
	}
}

func TestPrincipalFromIsEmptyOutsideAuthenticatedRequests(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := httpx.PrincipalFrom(req.Context()); ok {
		t.Error("PrincipalFrom reported a principal on an unauthenticated context")
	}
	if _, ok := httpx.PrincipalFrom(nil); ok { //nolint:staticcheck // nil is part of the contract being pinned
		t.Error("PrincipalFrom(nil) reported a principal")
	}
}
