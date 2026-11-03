package identityhttp

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/clientip"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

const roleSuperAdmin = "super_admin"

// Codes of the password reset endpoints.
const (
	CodeUserNotFound      = "user_not_found"
	CodeInvalidResetToken = "invalid_reset_token"
)

// ResetHandlers serves the assisted password reset: the super admin issues a link and the
// owner redeems it.
type ResetHandlers struct {
	issue    *commands.IssuePasswordReset
	redeem   *commands.ResetPassword
	verifier httpx.TokenVerifier
	errors   *Handlers
}

// NewResetHandlers wires the handlers. A nil logger discards logs.
func NewResetHandlers(
	issue *commands.IssuePasswordReset,
	redeem *commands.ResetPassword,
	verifier httpx.TokenVerifier,
	log *slog.Logger,
) *ResetHandlers {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &ResetHandlers{issue: issue, redeem: redeem, verifier: verifier, errors: &Handlers{log: log}}
}

// Mount registers POST /admin/password-resets (super admin) and POST /auth/reset-password
// (public: the token is the credential).
func (h *ResetHandlers) Mount(r chi.Router) {
	r.With(httpx.Authenticate(h.verifier), httpx.RequireRole(roleSuperAdmin), noStore).
		Post("/admin/password-resets", h.handleIssue)
	r.With(noStore).Post("/auth/reset-password", h.handleRedeem)
}

type issueRequest struct {
	Email string `json:"email"`
}

type issuedResponse struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *ResetHandlers) handleIssue(w http.ResponseWriter, r *http.Request) {
	var req issueRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	principal, _ := httpx.PrincipalFrom(r.Context())

	issued, err := h.issue.Handle(r.Context(), principal.UserID, req.Email)
	if err != nil {
		h.errors.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, issuedResponse{
		UserID: issued.UserID, Email: issued.Email, Token: issued.Token, ExpiresAt: issued.ExpiresAt,
	})
}

type redeemRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *ResetHandlers) handleRedeem(w http.ResponseWriter, r *http.Request) {
	var req redeemRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	err := h.redeem.Handle(r.Context(), commands.ResetPasswordInput{
		Token: req.Token, Password: req.Password, ClientIP: clientip.From(r),
	})
	if err != nil {
		h.errors.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
