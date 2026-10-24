// Package identityhttp exposes the identity use cases as the /auth endpoints.
package identityhttp

import (
	"errors"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// Codes specific to the identity endpoints.
const (
	CodeValidationFailed   = "validation_failed"
	CodeEmailTaken         = "email_taken"
	CodeInvalidCredentials = "invalid_credentials"
	CodeInvalidSession     = "invalid_session"
	CodeTooManyAttempts    = "too_many_attempts"
)

// Handlers serves the /auth endpoints.
type Handlers struct {
	register *commands.Register
	login    *commands.Login
	refresh  *commands.Refresh
	logout   *commands.Logout
	log      *slog.Logger
}

// New wires the handlers. A nil logger discards logs.
func New(
	register *commands.Register,
	login *commands.Login,
	refresh *commands.Refresh,
	logout *commands.Logout,
	log *slog.Logger,
) *Handlers {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	return &Handlers{register: register, login: login, refresh: refresh, logout: logout, log: log}
}

// Mount registers POST /auth/{register,login,refresh,logout}.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/auth", func(auth chi.Router) {
		// Responses carry tokens: no cache or proxy may keep them.
		auth.Use(noStore)
		auth.Post("/register", h.handleRegister)
		auth.Post("/login", h.handleLogin)
		auth.Post("/refresh", h.handleRefresh)
		auth.Post("/logout", h.handleLogout)
	})
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID    int64       `json:"id"`
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

type sessionResponse struct {
	User             userResponse `json:"user"`
	AccessToken      string       `json:"access_token"`
	AccessExpiresAt  time.Time    `json:"access_expires_at"`
	RefreshToken     string       `json:"refresh_token"`
	RefreshExpiresAt time.Time    `json:"refresh_expires_at"`
}

func toResponse(session commands.Session) sessionResponse {
	return sessionResponse{
		User:             userResponse{ID: session.User.ID, Email: session.User.Email, Role: session.User.Role},
		AccessToken:      session.AccessToken,
		AccessExpiresAt:  session.AccessExpiresAt,
		RefreshToken:     session.RefreshToken,
		RefreshExpiresAt: session.RefreshExpiresAt,
	}
}

func (h *Handlers) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	session, err := h.register.Handle(r.Context(), commands.RegisterInput{Email: req.Email, Password: req.Password})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(session))
}

func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	session, err := h.login.Handle(r.Context(), commands.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		ClientIP: clientIP(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(session))
}

func (h *Handlers) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	session, err := h.refresh.Handle(r.Context(), req.RefreshToken)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(session))
}

func (h *Handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if err := h.logout.Handle(r.Context(), req.RefreshToken); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// fail maps a use case error to its HTTP answer. Anything unexpected is logged
// with the request id and reported as a generic 500: the error text may name
// internals and never reaches the client.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var limited *commands.RateLimitedError

	switch {
	case errors.Is(err, domain.ErrEmailInvalid):
		invalid(w, "email", "invalid")
	case errors.Is(err, domain.ErrPasswordTooShort):
		invalid(w, "password", "too_short")
	case errors.Is(err, domain.ErrPasswordTooLong):
		invalid(w, "password", "too_long")
	case errors.Is(err, identity.ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, CodeEmailTaken, "that email is already registered", nil)
	case errors.Is(err, commands.ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, CodeInvalidCredentials, "email or password is incorrect", nil)
	case errors.Is(err, commands.ErrSessionInvalid):
		httpx.WriteError(w, http.StatusUnauthorized, CodeInvalidSession, "the session is no longer valid, sign in again", nil)
	case errors.As(err, &limited):
		seconds := int(math.Ceil(limited.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		httpx.WriteError(w, http.StatusTooManyRequests, CodeTooManyAttempts,
			"too many failed attempts, try again later", map[string]int{"retry_after_seconds": seconds})
	default:
		h.log.Error("identity request failed", "request_id", httpx.RequestID(r.Context()), "path", r.URL.Path, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
	}
}

func invalid(w http.ResponseWriter, field, reason string) {
	httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+field+" is not valid",
		map[string]string{"field": field, "reason": reason})
}

// clientIP is the address of the TCP peer. Forwarding headers are not trusted
// because any client can forge them; behind a reverse proxy this must be
// replaced with a trusted-proxy aware source (see docs/work/phase-1-identity.md).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
