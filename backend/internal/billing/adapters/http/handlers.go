// Package billinghttp exposes the public plan catalog and the plan of the caller.
package billinghttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// Handlers serves GET /plans and, when configured, GET /me/plan.
type Handlers struct {
	catalog  billing.PlanCatalog
	verifier httpx.TokenVerifier
	owners   billing.OwnerPlans
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

// Mount registers GET /plans, which is public because it only shows what is for sale,
// and GET /me/plan, which needs an owner token.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/plans", h.handlePlans)

	if h.owners != nil {
		r.With(httpx.Authenticate(h.verifier), httpx.RequireRole("business_admin")).Get("/me/plan", h.handleMyPlan)
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
