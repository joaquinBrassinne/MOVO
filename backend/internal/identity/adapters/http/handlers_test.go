package identityhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/ratelimit"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

const (
	ownerEmail    = "owner@example.com"
	ownerPassword = "correct horse battery"
)

type env struct {
	router http.Handler
	users  *postgres.Users
}

// newEnv wires the real stack over a fresh database, plus a probe route that
// only a super admin may call.
func newEnv(t *testing.T) env {
	t.Helper()

	pool := dbtest.New(t)
	users := postgres.NewUsers(pool)
	store := postgres.NewRefreshTokens(pool)

	signer, err := tokens.NewSigner([]byte("0123456789abcdef0123456789abcdef"), 15*time.Minute, nil)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	limiter := ratelimit.New(ratelimit.Config{MaxAttempts: 5, Window: 15 * time.Minute, MaxKeys: 1000}, nil)
	sessions := commands.NewSessions(signer, store, users, 30*24*time.Hour, nil, nil)

	handlers := identityhttp.New(
		commands.NewRegister(users, sessions),
		commands.NewLogin(users, limiter, sessions),
		commands.NewRefresh(sessions),
		commands.NewLogout(store, nil),
		nil,
	)

	probe := func(r chi.Router) {
		r.With(httpx.Authenticate(signer)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
			principal, _ := httpx.PrincipalFrom(r.Context())
			httpx.WriteJSON(w, http.StatusOK, principal)
		})
		r.With(httpx.Authenticate(signer), httpx.RequireRole(string(domain.RoleSuperAdmin))).
			Get("/admin/ping", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	}

	return env{router: httpx.NewRouter(nil, pool, handlers.Mount, probe), users: users}
}

func (e env) do(method, path, token string, body any) *httptest.ResponseRecorder {
	var reader bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = *bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, "/api/v1"+path, &reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

type session struct {
	User struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func decodeSession(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) session {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, wantStatus, rec.Body)
	}

	var s session
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return s
}

func errorBody(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorDetail {
	t.Helper()

	var envelope httpx.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v (body %s)", err, rec.Body)
	}
	return envelope.Error
}

func credentials(email, password string) map[string]string {
	return map[string]string{"email": email, "password": password}
}

func TestRegisterSignsTheOwnerInAndTheAccessTokenWorks(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/auth/register", "", credentials("  Owner@Example.com", ownerPassword))
	s := decodeSession(t, rec, http.StatusCreated)

	if s.User.Email != ownerEmail || s.User.Role != "business_admin" || s.User.ID == 0 {
		t.Errorf("user = %+v, want the new business admin", s.User)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store on a response that carries tokens", got)
	}
	if strings.Contains(rec.Body.String(), "argon2") || strings.Contains(strings.ToLower(rec.Body.String()), "password") {
		t.Errorf("response leaks credential material: %s", rec.Body)
	}

	me := e.do(http.MethodGet, "/me", s.AccessToken, nil)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"role":"business_admin"`) {
		t.Errorf("GET /me = %d %s, want the owner's principal", me.Code, me.Body)
	}
}

func TestRegisterCannotCreateAnythingButABusinessAdmin(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/auth/register", "", map[string]string{
		"email": ownerEmail, "password": ownerPassword, "role": "super_admin",
	})
	if rec.Code != http.StatusBadRequest || errorBody(t, rec).Code != httpx.CodeInvalidRequest {
		t.Fatalf("register with a role = %d %s, want 400 invalid_request", rec.Code, rec.Body)
	}

	if _, err := e.users.FindByEmail(t.Context(), ownerEmail); err == nil {
		t.Error("a rejected registration still created the user")
	}
}

func TestRegisterReportsValidationAndDuplicates(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	decodeSession(t, e.do(http.MethodPost, "/auth/register", "", credentials(ownerEmail, ownerPassword)), http.StatusCreated)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
		wantCode   string
		wantField  string
		wantReason string
	}{
		{"bad email", credentials("nope", ownerPassword), 422, identityhttp.CodeValidationFailed, "email", "invalid"},
		{"short password", credentials("new@example.com", "short"), 422, identityhttp.CodeValidationFailed, "password", "too_short"},
		{"long password", credentials("new@example.com", strings.Repeat("a", 129)), 422, identityhttp.CodeValidationFailed, "password", "too_long"},
		{"duplicate email", credentials("OWNER@example.com", ownerPassword), 409, identityhttp.CodeEmailTaken, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := e.do(http.MethodPost, "/auth/register", "", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}

			detail := errorBody(t, rec)
			if detail.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", detail.Code, tt.wantCode)
			}
			if tt.wantField != "" {
				got, _ := detail.Details.(map[string]any)
				if got["field"] != tt.wantField || got["reason"] != tt.wantReason {
					t.Errorf("details = %v, want field %q reason %q", detail.Details, tt.wantField, tt.wantReason)
				}
			}
		})
	}
}

func TestLoginWithValidAndInvalidCredentials(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	decodeSession(t, e.do(http.MethodPost, "/auth/register", "", credentials(ownerEmail, ownerPassword)), http.StatusCreated)

	s := decodeSession(t, e.do(http.MethodPost, "/auth/login", "", credentials(" OWNER@example.com", ownerPassword)), http.StatusOK)
	if s.User.Email != ownerEmail || s.AccessToken == "" || s.RefreshToken == "" {
		t.Errorf("login session = %+v, want the owner with both tokens", s)
	}

	wrong := e.do(http.MethodPost, "/auth/login", "", credentials(ownerEmail, "not the password"))
	ghost := e.do(http.MethodPost, "/auth/login", "", credentials("ghost@example.com", ownerPassword))

	for name, rec := range map[string]*httptest.ResponseRecorder{"wrong password": wrong, "unknown email": ghost} {
		if rec.Code != http.StatusUnauthorized || errorBody(t, rec).Code != identityhttp.CodeInvalidCredentials {
			t.Errorf("%s = %d %s, want 401 invalid_credentials", name, rec.Code, rec.Body)
		}
	}
	if wrong.Body.String() != ghost.Body.String() {
		t.Errorf("the two failures differ and reveal which emails exist:\n%s\n%s", wrong.Body, ghost.Body)
	}
}

func TestLoginIsThrottledAfterRepeatedFailures(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	decodeSession(t, e.do(http.MethodPost, "/auth/register", "", credentials(ownerEmail, ownerPassword)), http.StatusCreated)

	for range 5 {
		if rec := e.do(http.MethodPost, "/auth/login", "", credentials(ownerEmail, "wrong password!")); rec.Code != http.StatusUnauthorized {
			t.Fatalf("failed login status = %d, want 401", rec.Code)
		}
	}

	// The right password no longer helps while the account is blocked.
	rec := e.do(http.MethodPost, "/auth/login", "", credentials(ownerEmail, ownerPassword))
	if rec.Code != http.StatusTooManyRequests || errorBody(t, rec).Code != identityhttp.CodeTooManyAttempts {
		t.Fatalf("throttled login = %d %s, want 429 too_many_attempts", rec.Code, rec.Body)
	}

	retry, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || retry <= 0 || retry > 15*60 {
		t.Errorf("Retry-After = %q, want 1..900 seconds", rec.Header().Get("Retry-After"))
	}
}

func TestRefreshRotatesAndReuseKillsTheSession(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	first := decodeSession(t, e.do(http.MethodPost, "/auth/register", "", credentials(ownerEmail, ownerPassword)), http.StatusCreated)

	second := decodeSession(t, e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": first.RefreshToken}), http.StatusOK)
	if second.RefreshToken == first.RefreshToken || second.AccessToken == "" {
		t.Errorf("refresh returned %+v, want a rotated pair", second)
	}

	replay := e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": first.RefreshToken})
	if replay.Code != http.StatusUnauthorized || errorBody(t, replay).Code != identityhttp.CodeInvalidSession {
		t.Fatalf("replayed token = %d %s, want 401 invalid_session", replay.Code, replay.Body)
	}

	successor := e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": second.RefreshToken})
	if successor.Code != http.StatusUnauthorized {
		t.Errorf("successor after a replay = %d, want 401: the family must be revoked", successor.Code)
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := decodeSession(t, e.do(http.MethodPost, "/auth/register", "", credentials(ownerEmail, ownerPassword)), http.StatusCreated)

	body := map[string]string{"refresh_token": s.RefreshToken}
	if rec := e.do(http.MethodPost, "/auth/logout", "", body); rec.Code != http.StatusNoContent {
		t.Fatalf("logout = %d, want 204", rec.Code)
	}
	if rec := e.do(http.MethodPost, "/auth/logout", "", body); rec.Code != http.StatusNoContent {
		t.Errorf("second logout = %d, want 204 (idempotent)", rec.Code)
	}
	if rec := e.do(http.MethodPost, "/auth/refresh", "", body); rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout = %d, want 401", rec.Code)
	}
}

func TestOwnersCannotReachSuperAdminRoutes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := decodeSession(t, e.do(http.MethodPost, "/auth/register", "", credentials(ownerEmail, ownerPassword)), http.StatusCreated)

	admin, err := domain.NewUser("root@example.com", ownerPassword, domain.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if _, err := e.users.Create(t.Context(), admin); err != nil {
		t.Fatalf("Create super admin: %v", err)
	}
	super := decodeSession(t, e.do(http.MethodPost, "/auth/login", "", credentials("root@example.com", ownerPassword)), http.StatusOK)

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"forged token", owner.AccessToken + "x", http.StatusUnauthorized},
		{"business owner", owner.AccessToken, http.StatusForbidden},
		{"super admin", super.AccessToken, http.StatusNoContent},
	}
	for _, tt := range tests {
		if rec := e.do(http.MethodGet, "/admin/ping", tt.token, nil); rec.Code != tt.want {
			t.Errorf("%s: GET /admin/ping = %d, want %d", tt.name, rec.Code, tt.want)
		}
	}
}

func TestMalformedBodiesAreRejected(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	for _, path := range []string{"/auth/register", "/auth/login", "/auth/refresh", "/auth/logout"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1"+path, strings.NewReader("{not json"))
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("POST %s with a broken body = %d, want 400", path, rec.Code)
		}
	}
}
