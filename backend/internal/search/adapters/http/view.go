package searchhttp

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/clientip"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
)

// WithProfileView enables POST /businesses/{slug}/view, the beacon the public page
// sends after it loads.
func (h *Handlers) WithProfileView(view *commands.RecordProfileView) *Handlers {
	h.view = view
	return h
}

// WithServiceView enables POST /branches/{branch_id}/services/{service_id}/view, the beacon
// the public page sends when a service is on screen.
func (h *Handlers) WithServiceView(view *commands.RecordServiceView) *Handlers {
	h.serviceView = view
	return h
}

// handleServiceView counts a view and answers 204 with no body, whatever the ids are and
// whether or not it counted: the answer must not reveal which branches or services exist.
func (h *Handlers) handleServiceView(w http.ResponseWriter, r *http.Request) {
	branchID, okBranch := positiveID(chi.URLParam(r, "branch_id"))
	serviceID, okService := positiveID(chi.URLParam(r, "service_id"))
	if okBranch && okService {
		ctx := search.WithClient(r.Context(), clientip.From(r))
		if err := h.serviceView.Handle(ctx, branchID, serviceID, r.UserAgent()); err != nil {
			h.log.Error("service view failed", "request_id", httpx.RequestID(r.Context()), "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func positiveID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

// handleProfileView counts a view and answers 204 with no body and no cookies, whether
// or not it counted: that answer must not reveal which businesses exist or are public.
func (h *Handlers) handleProfileView(w http.ResponseWriter, r *http.Request) {
	ctx := search.WithClient(r.Context(), clientip.From(r))
	if err := h.view.Handle(ctx, chi.URLParam(r, "slug"), r.UserAgent()); err != nil {
		h.log.Error("profile view failed", "request_id", httpx.RequestID(r.Context()), "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
