package businesshttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type categoryJSON struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// mountCategories registers GET /categories. It is public: the catalog is what the
// sign-up form offers and holds nothing private.
func (h *Handlers) mountCategories(r chi.Router) {
	r.Get("/categories", h.handleCategories)
}

func (h *Handlers) handleCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.Categories.Handle(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := make([]categoryJSON, 0, len(list))
	for _, c := range list {
		out = append(out, categoryJSON{ID: c.ID, Slug: c.Slug, Name: c.Name})
	}

	// The catalog only changes with a migration, so a short shared cache is safe.
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"categories": out})
}
