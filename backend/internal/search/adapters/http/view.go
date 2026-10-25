package searchhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
)

// WithProfileView enables POST /businesses/{slug}/view, the beacon the public page
// sends after it loads.
func (h *Handlers) WithProfileView(view *commands.RecordProfileView) *Handlers {
	h.view = view
	return h
}

// handleProfileView counts a view and answers 204 with no body and no cookies, whether
// or not it counted: that answer must not reveal which businesses exist or are public.
func (h *Handlers) handleProfileView(w http.ResponseWriter, r *http.Request) {
	if err := h.view.Handle(r.Context(), chi.URLParam(r, "slug"), r.UserAgent()); err != nil {
		h.log.Error("profile view failed", "request_id", httpx.RequestID(r.Context()), "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
