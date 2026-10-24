package businesshttp

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type branchRequest struct {
	Name       string  `json:"name"`
	Address    string  `json:"address"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	BookingURL string  `json:"booking_url"`
}

func (b branchRequest) input() domain.NewBranchInput {
	return domain.NewBranchInput{
		Name: b.Name, Address: b.Address, City: b.City, Latitude: b.Latitude, Longitude: b.Longitude, BookingURL: b.BookingURL,
	}
}

type hoursRequest struct {
	Slots []slotJSON `json:"slots"`
}

// mountBranches registers the branch routes inside /me/business.
func (h *Handlers) mountBranches(me chi.Router) {
	me.Post("/branches", h.handleAddBranch)
	me.Put("/branches/{id}", h.handleEditBranch)
	me.Delete("/branches/{id}", h.handleRemoveBranch)
	me.Put("/branches/{id}/hours", h.handleSetHours)
}

func (h *Handlers) handleAddBranch(w http.ResponseWriter, r *http.Request) {
	var req branchRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	created, err := h.AddBranch.Handle(r.Context(), commands.AddBranchInput{OwnerID: callerID(r), Branch: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toBranchJSON(created, nil))
}

func (h *Handlers) handleEditBranch(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrBranchNotFound)
	if !ok {
		return
	}
	var req branchRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.EditBranch.Handle(r.Context(), commands.EditBranchInput{OwnerID: callerID(r), BranchID: id, Branch: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBranchJSON(updated, nil))
}

func (h *Handlers) handleRemoveBranch(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrBranchNotFound)
	if !ok {
		return
	}

	if err := h.RemoveBranch.Handle(r.Context(), commands.RemoveBranchInput{OwnerID: callerID(r), BranchID: id}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) handleSetHours(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrBranchNotFound)
	if !ok {
		return
	}
	var req hoursRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	slots, err := toSlots(req.Slots)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	if err := h.SetBranchHours.Handle(r.Context(), commands.SetBranchHoursInput{OwnerID: callerID(r), BranchID: id, Slots: slots}); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"hours": toSlotsJSON(slots)})
}

// pathID reads the {id} of the route. A value that is not a number cannot name an
// existing resource, so it is answered like a missing one with notFoundErr.
func (h *Handlers) pathID(w http.ResponseWriter, r *http.Request, notFoundErr error) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		h.fail(w, r, notFoundErr)
		return 0, false
	}
	return id, true
}
