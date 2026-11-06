package businesshttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type registerImageRequest struct {
	ObjectKey string `json:"object_key"`
}

type reorderImagesRequest struct {
	IDs []int64 `json:"ids"`
}

type registeredImageJSON struct {
	ID        int64  `json:"id"`
	Position  int    `json:"position"`
	ObjectKey string `json:"object_key"`
}

// mountPhotos registers the photo routes inside /me/business. The upload itself
// goes straight to the object server through the URL of /images/presign; these
// routes only register, order and remove what was uploaded.
func (h *Handlers) mountPhotos(me chi.Router) {
	me.Post("/images", h.handleRegisterImage)
	me.Put("/images/order", h.handleReorderImages)
	me.Delete("/images/{id}", h.handleRemoveImage)
}

func (h *Handlers) handleRegisterImage(w http.ResponseWriter, r *http.Request) {
	var req registerImageRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	image, err := h.RegisterImage.Handle(r.Context(), commands.RegisterImageInput{OwnerID: callerID(r), ObjectKey: req.ObjectKey})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, registeredImageJSON{ID: image.ID, Position: image.Position, ObjectKey: image.ObjectKey})
}

func (h *Handlers) handleReorderImages(w http.ResponseWriter, r *http.Request) {
	var req reorderImagesRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if err := h.ReorderImages.Handle(r.Context(), commands.ReorderImagesInput{OwnerID: callerID(r), IDs: req.IDs}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) handleRemoveImage(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrImageNotFound)
	if !ok {
		return
	}

	if err := h.RemoveImage.Handle(r.Context(), commands.RemoveImageInput{OwnerID: callerID(r), ImageID: id}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
