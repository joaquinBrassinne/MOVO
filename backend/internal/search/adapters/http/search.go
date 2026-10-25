// Package searchhttp serves the public search over HTTP.
package searchhttp

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// CodeValidationFailed is the code of a search parameter that is missing or out of range.
const CodeValidationFailed = "validation_failed"

// Handlers serves the search endpoints.
type Handlers struct {
	nearby *queries.NearbyBranches
	log    *slog.Logger
}

// New builds the handlers. A nil logger discards logs.
func New(nearby *queries.NearbyBranches, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Handlers{nearby: nearby, log: log}
}

// Mount registers GET /search. It is public and takes no identity.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/search", h.handleSearch)
}

// branchJSON is the light card the map and the lists need: nothing internal and no
// contact data (the WhatsApp number never leaves the server).
type branchJSON struct {
	BranchID     int64   `json:"branch_id"`
	BusinessName string  `json:"business_name"`
	BusinessSlug string  `json:"business_slug"`
	BranchName   string  `json:"branch_name"`
	Address      string  `json:"address"`
	City         string  `json:"city"`
	Reference    string  `json:"reference"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	DistanceM    int     `json:"distance_m"`
	RatingAvg    float64 `json:"rating_avg"`
	RatingCount  int     `json:"rating_count"`
	Verified     bool    `json:"verified"`
	Featured     bool    `json:"featured"`
}

func (h *Handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	q, field, reason := parseQuery(r)
	if field != "" {
		invalid(w, field, reason)
		return
	}

	page, err := h.nearby.Handle(r.Context(), q)
	if err != nil {
		var qe *search.QueryError
		if errors.As(err, &qe) {
			invalid(w, qe.Field, "out_of_range")
			return
		}
		h.log.Error("search failed", "request_id", httpx.RequestID(r.Context()), "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}

	items := make([]branchJSON, 0, len(page.Items))
	for _, b := range page.Items {
		items = append(items, branchJSON{
			BranchID: b.BranchID, BusinessName: b.BusinessName, BusinessSlug: b.BusinessSlug,
			BranchName: b.BranchName, Address: b.Address, City: b.City, Reference: b.Reference,
			Lat: b.Lat, Lng: b.Lng, DistanceM: int(math.Round(b.DistanceM)),
			RatingAvg: b.RatingAvg, RatingCount: b.RatingCount, Verified: b.Verified, Featured: b.Featured,
		})
	}
	var next *int
	if page.HasMore {
		n := q.Offset + len(page.Items)
		next = &n
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": items, "next_offset": next})
}

// parseQuery reads the query string. lat and lng are required; the rest is optional.
// A failure returns the field and a stable reason.
func parseQuery(r *http.Request) (q search.Query, field, reason string) {
	values := r.URL.Query()

	number := func(name string) (float64, bool) {
		raw := values.Get(name)
		if raw == "" {
			field, reason = name, "required"
			return 0, false
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			field, reason = name, "invalid"
			return 0, false
		}
		return v, true
	}
	integer := func(name string) (int, bool) {
		raw := values.Get(name)
		if raw == "" {
			return 0, true
		}
		v, err := strconv.Atoi(raw)
		if err != nil {
			field, reason = name, "invalid"
			return 0, false
		}
		return v, true
	}

	var ok bool
	if q.Lat, ok = number("lat"); !ok {
		return q, field, reason
	}
	if q.Lng, ok = number("lng"); !ok {
		return q, field, reason
	}
	if q.RadiusM, ok = integer("radius_m"); !ok {
		return q, field, reason
	}
	if q.Limit, ok = integer("limit"); !ok {
		return q, field, reason
	}
	if q.Offset, ok = integer("offset"); !ok {
		return q, field, reason
	}
	q.Category = values.Get("category")
	return q, "", ""
}

func invalid(w http.ResponseWriter, field, reason string) {
	httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+field+" is not valid",
		map[string]string{"field": field, "reason": reason})
}
