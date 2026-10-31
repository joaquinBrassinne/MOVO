package billinghttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

const roleSuperAdmin = "super_admin"

// Codes of the subscription endpoints.
const (
	CodeValidationFailed       = "validation_failed"
	CodeBusinessNotFound       = "business_not_found"
	CodeSubscriptionNotFound   = "subscription_not_found"
	CodeBusinessNotConfirmable = "business_not_confirmable"
	CodeSubscriptionExists     = "subscription_exists"
	CodeNoChange               = "no_change"
	CodeConflict               = "conflict"
)

// AdminDeps are the use cases of the super admin subscription endpoints.
type AdminDeps struct {
	Verifier      httpx.TokenVerifier
	Subscriptions billing.SubscriptionRepository
	Confirm       *commands.ConfirmPayment
	Register      *commands.RegisterPayment
	ChangePlan    *commands.ChangePlan
}

// Admin serves the super admin subscription endpoints under /admin/subscriptions.
type Admin struct {
	deps AdminDeps
	log  *slog.Logger
	now  func() time.Time
}

// NewAdmin wires the handlers.
func NewAdmin(log *slog.Logger, deps AdminDeps) *Admin {
	return &Admin{deps: deps, log: log, now: time.Now}
}

// Mount registers the routes. Only a super_admin token gets past the group middleware;
// the use cases do not check the role. {id} is the id of the business.
func (a *Admin) Mount(r chi.Router) {
	r.Route("/admin/subscriptions", func(admin chi.Router) {
		admin.Use(httpx.Authenticate(a.deps.Verifier), httpx.RequireRole(roleSuperAdmin), noStore)

		admin.Get("/{id}", a.handleGet)
		admin.Post("/{id}", a.handleConfirm)
		admin.Post("/{id}/payments", a.handleRegisterPayment)
		admin.Put("/{id}/plan", a.handleChangePlan)
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type subscriptionJSON struct {
	BusinessID        int64  `json:"business_id"`
	Plan              string `json:"plan"`
	PaidUntil         string `json:"paid_until"`
	GraceUntil        string `json:"grace_until"`
	Status            string `json:"status"`
	MPSubscriptionRef string `json:"mp_subscription_ref,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

func (a *Admin) toJSON(s domain.Subscription) subscriptionJSON {
	return subscriptionJSON{
		BusinessID: s.BusinessID, Plan: string(s.Plan),
		PaidUntil: s.PaidUntil.String(), GraceUntil: s.GraceEnd().String(),
		Status:            string(s.StatusAt(domain.DayOf(a.now()))),
		MPSubscriptionRef: s.MPSubscriptionRef, Notes: s.Notes,
	}
}

func callerID(r *http.Request) int64 {
	principal, _ := httpx.PrincipalFrom(r.Context())
	return principal.UserID
}

func (a *Admin) businessID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		httpx.WriteError(w, http.StatusNotFound, CodeBusinessNotFound, "business not found", nil)
		return 0, false
	}
	return id, true
}

func (a *Admin) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := a.businessID(w, r)
	if !ok {
		return
	}
	sub, err := a.deps.Subscriptions.Find(r.Context(), id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.toJSON(sub))
}

type confirmRequest struct {
	Plan              string `json:"plan"`
	PaidUntil         string `json:"paid_until"`
	MPSubscriptionRef string `json:"mp_subscription_ref"`
	Notes             string `json:"notes"`
}

func (a *Admin) handleConfirm(w http.ResponseWriter, r *http.Request) {
	id, ok := a.businessID(w, r)
	if !ok {
		return
	}
	var req confirmRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	parsed, err := time.Parse(time.DateOnly, req.PaidUntil)
	if err != nil {
		invalid(w, "paid_until", "invalid")
		return
	}

	sub, err := a.deps.Confirm.Handle(r.Context(), commands.ConfirmPaymentInput{
		ActorID: callerID(r), BusinessID: id, Plan: req.Plan, PaidUntil: domain.DayFromDate(parsed),
		MPSubscriptionRef: req.MPSubscriptionRef, Notes: req.Notes,
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, a.toJSON(sub))
}

func (a *Admin) handleRegisterPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := a.businessID(w, r)
	if !ok {
		return
	}
	sub, err := a.deps.Register.Handle(r.Context(), callerID(r), id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.toJSON(sub))
}

type changePlanRequest struct {
	Plan string `json:"plan"`
}

func (a *Admin) handleChangePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := a.businessID(w, r)
	if !ok {
		return
	}
	var req changePlanRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	sub, err := a.deps.ChangePlan.Handle(r.Context(), callerID(r), id, req.Plan)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.toJSON(sub))
}

var validation = map[error]fieldError{
	domain.ErrPlanInvalid:          {"plan", "invalid"},
	domain.ErrPaidUntilInvalid:     {"paid_until", "invalid"},
	commands.ErrPaidUntilInThePast: {"paid_until", "in_the_past"},
	domain.ErrRefInvalid:           {"mp_subscription_ref", "too_long"},
	domain.ErrNotesInvalid:         {"notes", "too_long"},
}

type fieldError struct{ field, reason string }

var conflicts = []struct {
	err     error
	status  int
	code    string
	message string
}{
	{billing.ErrBusinessNotFound, http.StatusNotFound, CodeBusinessNotFound, "business not found"},
	{billing.ErrSubscriptionNotFound, http.StatusNotFound, CodeSubscriptionNotFound, "this business has no subscription yet"},
	{billing.ErrBusinessNotConfirmable, http.StatusConflict, CodeBusinessNotConfirmable, "only a business pending review or active can get a payment confirmed"},
	{billing.ErrSubscriptionExists, http.StatusConflict, CodeSubscriptionExists, "this business already has a subscription: register a payment or change the plan"},
	{commands.ErrNoChange, http.StatusConflict, CodeNoChange, "the subscription already has that plan"},
	{billing.ErrSubscriptionConflict, http.StatusConflict, CodeConflict, "the subscription was changed by someone else, reload and try again"},
}

func (a *Admin) fail(w http.ResponseWriter, r *http.Request, err error) {
	for _, c := range conflicts {
		if errors.Is(err, c.err) {
			httpx.WriteError(w, c.status, c.code, c.message, nil)
			return
		}
	}
	for known, field := range validation {
		if errors.Is(err, known) {
			invalid(w, field.field, field.reason)
			return
		}
	}
	a.log.Error("subscription request failed", "request_id", httpx.RequestID(r.Context()), "path", httpx.Route(r), "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
}

func invalid(w http.ResponseWriter, field, reason string) {
	httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+field+" is not valid",
		map[string]string{"field": field, "reason": reason})
}
