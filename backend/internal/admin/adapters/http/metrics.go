// Package adminhttp serves the super admin's figures.
package adminhttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/admin"
	"github.com/joaquinBrassinne/MOVO/internal/admin/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

const roleSuperAdmin = "super_admin"

// Deps are the collaborators of the handlers.
type Deps struct {
	Log      *slog.Logger
	Verifier httpx.TokenVerifier
	Overview *queries.GetOverview
}

// Handlers serves GET /admin/metrics.
type Handlers struct{ Deps }

// New wires the handlers. A nil logger discards logs.
func New(d Deps) *Handlers {
	if d.Log == nil {
		d.Log = slog.New(slog.DiscardHandler)
	}
	return &Handlers{d}
}

// Mount registers the routes, for the super admin only.
func (h *Handlers) Mount(r chi.Router) {
	r.With(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleSuperAdmin), noStore).
		Get("/admin/metrics", h.handleOverview)
}

type countsJSON struct {
	Generated int `json:"generated"`
	Redeemed  int `json:"redeemed"`
}

func toCounts(c admin.CouponCounts) countsJSON {
	return countsJSON{Generated: c.Generated, Redeemed: c.Redeemed}
}

func (h *Handlers) handleOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.Overview.Handle(r.Context())
	if err != nil {
		h.Log.Error("admin metrics failed", "request_id", httpx.RequestID(r.Context()), "path", httpx.Route(r), "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"active_businesses":  overview.ActiveBusinesses,
		"coupons":            toCounts(overview.Coupons),
		"coupons_this_month": toCounts(overview.CouponsThisMonth),
		"month_start":        overview.MonthStart.Format("2006-01-02"),
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
