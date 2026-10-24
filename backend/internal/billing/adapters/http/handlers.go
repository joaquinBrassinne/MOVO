// Package billinghttp exposes the public plan catalog.
package billinghttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// Handlers serves GET /plans.
type Handlers struct {
	catalog billing.PlanCatalog
}

// New wires the handlers.
func New(catalog billing.PlanCatalog) *Handlers {
	return &Handlers{catalog: catalog}
}

// Mount registers GET /plans. It is public: it only shows what is for sale.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/plans", h.handlePlans)
}

type planJSON struct {
	Plan  string `json:"plan"`
	Price int    `json:"price"`
	// The limits are null when the plan is unlimited.
	MaxBranches       *int `json:"max_branches"`
	MaxActiveServices *int `json:"max_active_services"`
}

func (h *Handlers) handlePlans(w http.ResponseWriter, _ *http.Request) {
	plans := h.catalog.Plans()
	out := make([]planJSON, 0, len(plans))
	for _, p := range plans {
		out = append(out, planJSON{
			Plan:              string(p.Plan),
			Price:             p.Price,
			MaxBranches:       limit(p.Entitlements.MaxBranches()),
			MaxActiveServices: limit(p.Entitlements.MaxActiveServices()),
		})
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
