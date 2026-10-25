package searchhttp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// CodeBranchNotFound is the code of a branch that does not exist or is not public.
const CodeBranchNotFound = "branch_not_found"

// handleWhatsApp counts the click and redirects to the conversation. The number is
// only ever inside the Location header, never in a body.
func (h *Handlers) handleWhatsApp(w http.ResponseWriter, r *http.Request) {
	branchID, err := strconv.ParseInt(chi.URLParam(r, "branch_id"), 10, 64)
	if err != nil || branchID <= 0 {
		httpx.WriteError(w, http.StatusNotFound, CodeBranchNotFound, "branch not found", nil)
		return
	}

	var serviceID int64
	if raw := r.URL.Query().Get("service"); raw != "" {
		serviceID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || serviceID <= 0 {
			invalid(w, "service", "invalid")
			return
		}
	}

	// A HEAD gets the same answer without a count: previewers use it, people do not.
	var link string
	if r.Method == http.MethodHead {
		link, err = h.whatsapp.Preview(r.Context(), branchID, serviceID)
	} else {
		link, err = h.whatsapp.Handle(r.Context(), branchID, serviceID, r.UserAgent())
	}
	if errors.Is(err, search.ErrBranchNotFound) {
		httpx.WriteError(w, http.StatusNotFound, CodeBranchNotFound, "branch not found", nil)
		return
	}
	if err != nil {
		h.log.Error("whatsapp redirect failed", "request_id", httpx.RequestID(r.Context()), "branch_id", branchID, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}

	// A cached redirect would skip the next click, and the page that sent the visitor
	// has nothing to tell WhatsApp.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Location", link)
	w.WriteHeader(http.StatusFound)
}
