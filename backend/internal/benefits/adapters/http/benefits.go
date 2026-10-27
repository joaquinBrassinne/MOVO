// Package benefitshttp exposes the benefit use cases: the owner endpoints under
// /me/business/benefits.
package benefitshttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// roleOwner is the plain string stored in the token, so this package needs no
// knowledge of the identity module.
const roleOwner = "business_admin"

// Codes specific to the benefit endpoints.
const (
	CodeValidationFailed = "validation_failed"
	CodeBusinessNotFound = "business_not_found"
	CodeBenefitNotFound  = "benefit_not_found"
	CodeBusinessDeleted  = "business_deleted"
)

// Deps are the use cases and the token verifier the handlers need.
type Deps struct {
	Verifier httpx.TokenVerifier
	Log      *slog.Logger

	List      *queries.ListMyBenefits
	Add       *commands.AddBenefit
	Edit      *commands.EditBenefit
	SetActive *commands.SetBenefitActive
}

// Handlers serves the benefit endpoints.
type Handlers struct {
	Deps
	log *slog.Logger
}

// New wires the handlers. A nil logger discards logs.
func New(deps Deps) *Handlers {
	log := deps.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Handlers{Deps: deps, log: log}
}

// Mount registers the routes. They live under /me/business, which the business module
// also serves; chi picks these static paths over its catch-all. Authentication and the
// role are enforced here, so no handler runs for the wrong kind of caller.
func (h *Handlers) Mount(r chi.Router) {
	owner := r.With(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleOwner), noStore)

	owner.Get("/me/business/benefits", h.handleList)
	owner.Post("/me/business/benefits", h.handleAdd)
	owner.Put("/me/business/benefits/{id}", h.handleEdit)
	owner.Put("/me/business/benefits/{id}/active", h.handleSetActive)
}

type benefitRequest struct {
	Type            domain.Type `json:"type"`
	Percent         int         `json:"percent"`
	GiftDescription string      `json:"gift_description"`
	MonthlyLimit    int         `json:"monthly_limit"`
}

func (b benefitRequest) input() domain.BenefitInput {
	return domain.BenefitInput{Type: b.Type, Percent: b.Percent, GiftDescription: b.GiftDescription, MonthlyLimit: b.MonthlyLimit}
}

type activeRequest struct {
	Active bool `json:"active"`
}

// benefitJSON is a benefit as the panel reads it. The field of the other type is null.
type benefitJSON struct {
	ID              int64       `json:"id"`
	Type            domain.Type `json:"type"`
	Percent         *int        `json:"percent"`
	GiftDescription *string     `json:"gift_description"`
	Active          bool        `json:"active"`
	MonthlyLimit    int         `json:"monthly_limit"`
}

func toBenefitJSON(b domain.Benefit) benefitJSON {
	out := benefitJSON{ID: b.ID, Type: b.Type, Active: b.Active, MonthlyLimit: b.MonthlyLimit}
	if b.Type == domain.TypePercent {
		out.Percent = &b.Percent
	} else {
		out.GiftDescription = &b.GiftDescription
	}
	return out
}

func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	list, err := h.List.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := make([]benefitJSON, 0, len(list))
	for _, b := range list {
		out = append(out, toBenefitJSON(b))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"benefits": out})
}

func (h *Handlers) handleAdd(w http.ResponseWriter, r *http.Request) {
	var req benefitRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	created, err := h.Add.Handle(r.Context(), commands.AddBenefitInput{OwnerID: callerID(r), Benefit: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toBenefitJSON(created))
}

func (h *Handlers) handleEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req benefitRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.Edit.Handle(r.Context(), commands.EditBenefitInput{OwnerID: callerID(r), BenefitID: id, Benefit: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBenefitJSON(updated))
}

func (h *Handlers) handleSetActive(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req activeRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.SetActive.Handle(r.Context(), commands.SetBenefitActiveInput{OwnerID: callerID(r), BenefitID: id, Active: req.Active})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBenefitJSON(updated))
}

// pathID reads {id}; a malformed one answers like a benefit that does not exist.
func (h *Handlers) pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		h.fail(w, r, benefits.ErrBenefitNotFound)
		return 0, false
	}
	return id, true
}

// callerID is the authenticated caller. RequireRole already guaranteed it exists; the
// id always comes from the token, never from the body or the URL.
func callerID(r *http.Request) int64 {
	principal, _ := httpx.PrincipalFrom(r.Context())
	return principal.UserID
}

// noStore keeps owner data out of shared caches.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// validation maps each domain validation error to the field it is about. Anything not
// listed here is not a client mistake.
var validation = map[error]struct{ field, reason string }{
	domain.ErrBenefitTypeInvalid:     {"type", "invalid"},
	domain.ErrPercentInvalid:         {"percent", "invalid"},
	domain.ErrGiftDescriptionInvalid: {"gift_description", "invalid"},
	domain.ErrMonthlyLimitInvalid:    {"monthly_limit", "invalid"},
}

// fail maps a use case error to its HTTP answer. Anything unexpected is logged with
// the request id and reported as a generic 500: the error text may name internals and
// never reaches the client.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, benefits.ErrBenefitNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeBenefitNotFound, "benefit not found", nil)
		return
	case errors.Is(err, benefits.ErrBusinessNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeBusinessNotFound, "you have not created a business yet", nil)
		return
	case errors.Is(err, benefits.ErrBusinessNotEditable):
		httpx.WriteError(w, http.StatusConflict, CodeBusinessDeleted, "this business was deleted and can no longer change", nil)
		return
	}
	for known, field := range validation {
		if errors.Is(err, known) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+field.field+" is not valid",
				map[string]string{"field": field.field, "reason": field.reason})
			return
		}
	}

	h.log.Error("benefits request failed", "request_id", httpx.RequestID(r.Context()), "path", r.URL.Path, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
}
