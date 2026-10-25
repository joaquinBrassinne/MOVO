package businesshttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type serviceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// BranchID is null (or absent) for a service of the whole business.
	BranchID *int64 `json:"branch_id"`
	// Price is the estimated price in whole pesos; null or absent means not set.
	Price *int `json:"price"`
}

func (s serviceRequest) input() domain.ServiceInput {
	in := domain.ServiceInput{Name: s.Name, Description: s.Description}
	if s.BranchID != nil {
		in.BranchID = *s.BranchID
	}
	if s.Price != nil {
		in.Price = *s.Price
	}
	return in
}

type activeRequest struct {
	Active bool `json:"active"`
}

// mountServices registers the service routes inside /me/business.
func (h *Handlers) mountServices(me chi.Router) {
	me.Get("/services", h.handleListServices)
	me.Post("/services", h.handleAddService)
	me.Put("/services/{id}", h.handleEditService)
	me.Delete("/services/{id}", h.handleRemoveService)
	me.Put("/services/{id}/active", h.handleSetServiceActive)
	me.Put("/services/{id}/schedule", h.handleSetServiceSchedule)
}

func (h *Handlers) handleListServices(w http.ResponseWriter, r *http.Request) {
	views, err := h.ListServices.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := make([]serviceJSON, 0, len(views))
	for _, v := range views {
		out = append(out, toServiceJSON(v.Service, v.Schedule))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"services": out})
}

func (h *Handlers) handleAddService(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	created, err := h.AddService.Handle(r.Context(), commands.AddServiceInput{OwnerID: callerID(r), Service: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toServiceBaseJSON(created))
}

func (h *Handlers) handleEditService(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrServiceNotFound)
	if !ok {
		return
	}
	var req serviceRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.EditService.Handle(r.Context(), commands.EditServiceInput{OwnerID: callerID(r), ServiceID: id, Service: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toServiceBaseJSON(updated))
}

func (h *Handlers) handleRemoveService(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrServiceNotFound)
	if !ok {
		return
	}

	if err := h.RemoveService.Handle(r.Context(), commands.RemoveServiceInput{OwnerID: callerID(r), ServiceID: id}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) handleSetServiceActive(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrServiceNotFound)
	if !ok {
		return
	}
	var req activeRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.SetServiceActive.Handle(r.Context(), commands.SetServiceActiveInput{OwnerID: callerID(r), ServiceID: id, Active: req.Active})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toServiceBaseJSON(updated))
}

func (h *Handlers) handleSetServiceSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrServiceNotFound)
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

	if err := h.SetServiceSchedule.Handle(r.Context(), commands.SetServiceScheduleInput{OwnerID: callerID(r), ServiceID: id, Slots: slots}); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"schedule": toSlotsJSON(slots)})
}
