// Package billinghttp exposes the public plan catalog and the plan of the caller.
package billinghttp

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// Handlers serves GET /plans and, when configured, GET /me/plan.
type Handlers struct {
	catalog  billing.PlanCatalog
	verifier httpx.TokenVerifier
	owners   billing.OwnerPlans
	alerts   billing.OwnerAlerts
	subs     billing.OwnerSubscriptions
}

// New wires the handlers.
func New(catalog billing.PlanCatalog) *Handlers {
	return &Handlers{catalog: catalog}
}

// WithOwnerPlans enables GET /me/plan for business owners.
func (h *Handlers) WithOwnerPlans(verifier httpx.TokenVerifier, owners billing.OwnerPlans) *Handlers {
	h.verifier, h.owners = verifier, owners
	return h
}

// WithOwnerSubscription enables GET /me/subscription for business owners.
func (h *Handlers) WithOwnerSubscription(verifier httpx.TokenVerifier, subs billing.OwnerSubscriptions) *Handlers {
	h.verifier, h.subs = verifier, subs
	return h
}

// WithAlerts enables GET /me/alerts and POST /me/alerts/{id}/read for business owners.
func (h *Handlers) WithAlerts(verifier httpx.TokenVerifier, alerts billing.OwnerAlerts) *Handlers {
	h.verifier, h.alerts = verifier, alerts
	return h
}

// Mount registers GET /plans, which is public because it only shows what is for sale,
// and GET /me/plan, which needs an owner token.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/plans", h.handlePlans)

	if h.owners != nil {
		r.With(httpx.Authenticate(h.verifier), httpx.RequireRole("business_admin")).Get("/me/plan", h.handleMyPlan)
	}
	if h.subs != nil {
		r.With(httpx.Authenticate(h.verifier), httpx.RequireRole("business_admin"), noStore).Get("/me/subscription", h.handleMySubscription)
	}
	if h.alerts != nil {
		owner := r.With(httpx.Authenticate(h.verifier), httpx.RequireRole("business_admin"), noStore)
		owner.Get("/me/alerts", h.handleMyAlerts)
		owner.Post("/me/alerts/{id}/read", h.handleReadAlert)
	}
}

type mySubscriptionJSON struct {
	Pilot bool `json:"pilot"`
	// Subscription is null for a business that has none (a pilot, or one still in review).
	Subscription *ownSubscriptionJSON `json:"subscription"`
}

type ownSubscriptionJSON struct {
	Plan       string `json:"plan"`
	PaidUntil  string `json:"paid_until"`
	GraceUntil string `json:"grace_until"`
	// Status is active, grace or expired; the owner never sees notes or references.
	Status string `json:"status"`
}

// handleMySubscription answers with the payment state of the caller's own business. The
// owner id comes from the token, never from the request.
func (h *Handlers) handleMySubscription(w http.ResponseWriter, r *http.Request) {
	own, err := h.subs.SubscriptionOfOwner(r.Context(), callerID(r))
	switch {
	case errors.Is(err, billing.ErrOwnerHasNoBusiness):
		httpx.WriteError(w, http.StatusNotFound, "business_not_found", "you have not created a business yet", nil)
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}
	out := mySubscriptionJSON{Pilot: own.Pilot}
	if s := own.Subscription; s != nil {
		out.Subscription = &ownSubscriptionJSON{
			Plan: string(s.Plan), PaidUntil: s.PaidUntil.String(), GraceUntil: s.GraceEnd().String(),
			Status: string(s.StatusAt(domain.DayOf(time.Now()))),
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type alertJSON struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

func (h *Handlers) handleMyAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.alerts.Unread(r.Context(), callerID(r))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}
	out := make([]alertJSON, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, alertJSON{ID: a.ID, Type: a.Type, Payload: a.Payload, CreatedAt: a.CreatedAt})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"alerts": out})
}

func (h *Handlers) handleReadAlert(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		httpx.WriteError(w, http.StatusNotFound, "alert_not_found", "alert not found", nil)
		return
	}
	switch err := h.alerts.MarkRead(r.Context(), callerID(r), id); {
	case errors.Is(err, billing.ErrAlertNotFound):
		httpx.WriteError(w, http.StatusNotFound, "alert_not_found", "alert not found", nil)
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

type planJSON struct {
	Plan  string `json:"plan"`
	Price int    `json:"price"`
	// CheckoutURL is the Mercado Pago subscription link; only the public catalog sends it.
	CheckoutURL string `json:"checkout_url,omitempty"`
	// The limits are null when the plan is unlimited.
	MaxBranches       *int `json:"max_branches"`
	MaxActiveServices *int `json:"max_active_services"`
}

func toPlanJSON(p billing.PlanInfo) planJSON {
	return planJSON{
		Plan:              string(p.Plan),
		Price:             p.Price,
		MaxBranches:       limit(p.Entitlements.MaxBranches()),
		MaxActiveServices: limit(p.Entitlements.MaxActiveServices()),
	}
}

func (h *Handlers) handlePlans(w http.ResponseWriter, _ *http.Request) {
	plans := h.catalog.Plans()
	out := make([]planJSON, 0, len(plans))
	for _, p := range plans {
		item := toPlanJSON(p)
		item.CheckoutURL = p.CheckoutURL
		out = append(out, item)
	}

	// The catalog only changes with a restart, so a short shared cache is safe.
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"plans": out})
}

// limit turns the domain's 0 = unlimited into null.
func limit(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

// handleMyPlan answers with the plan of the caller's own business. The owner id comes
// from the token, never from the request.
func (h *Handlers) handleMyPlan(w http.ResponseWriter, r *http.Request) {
	principal, _ := httpx.PrincipalFrom(r.Context())

	info, err := h.owners.PlanOfOwner(r.Context(), principal.UserID)
	switch {
	case errors.Is(err, billing.ErrOwnerHasNoBusiness):
		httpx.WriteError(w, http.StatusNotFound, "business_not_found", "you have not created a business yet", nil)
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, toPlanJSON(info))
}
