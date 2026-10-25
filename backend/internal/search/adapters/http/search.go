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
	"github.com/joaquinBrassinne/MOVO/internal/search/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// CodeValidationFailed is the code of a search parameter that is missing or out of range.
const CodeValidationFailed = "validation_failed"

// Handlers serves the search endpoints.
type Handlers struct {
	nearby   *queries.NearbyBranches
	featured *queries.FeaturedBranches
	whatsapp *commands.OpenWhatsApp
	profile  *queries.BusinessProfile
	log      *slog.Logger
}

// New builds the handlers. A nil logger discards logs.
func New(nearby *queries.NearbyBranches, featured *queries.FeaturedBranches, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Handlers{nearby: nearby, featured: featured, log: log}
}

// Mount registers GET /search, GET /home/featured and, when enabled, GET /businesses/{slug}. All are public and take no identity.
func (h *Handlers) Mount(r chi.Router) {
	r.Get("/search", h.handleSearch)
	r.Get("/home/featured", h.handleFeatured)
	if h.profile != nil {
		r.Get("/businesses/{slug}", h.handleProfile)
	}
	if h.whatsapp != nil {
		r.Get("/r/wa/{branch_id}", h.handleWhatsApp)
		r.Head("/r/wa/{branch_id}", h.handleWhatsApp)
	}
}

// WithWhatsApp enables GET /r/wa/{branch_id}, the redirect behind the WhatsApp button.
func (h *Handlers) WithWhatsApp(open *commands.OpenWhatsApp) *Handlers {
	h.whatsapp = open
	return h
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
	CategorySlug string  `json:"category_slug"`
	CategoryName string  `json:"category_name"`
}

func (h *Handlers) handleSearch(w http.ResponseWriter, r *http.Request) {
	q, field, reason := parseQuery(r)
	if field != "" {
		invalid(w, field, reason)
		return
	}

	page, err := h.nearby.Handle(r.Context(), q)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	var next *int
	if page.HasMore {
		n := q.Offset + len(page.Items)
		next = &n
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": cards(page.Items), "next_offset": next})
}

// handleFeatured serves "Destacados cerca tuyo". It takes lat, lng, radius_m and
// limit; category and offset do not apply to a rotating section and are ignored.
func (h *Handlers) handleFeatured(w http.ResponseWriter, r *http.Request) {
	q, field, reason := parseQuery(r)
	if field != "" {
		invalid(w, field, reason)
		return
	}
	q.Category, q.Offset = "", 0

	items, err := h.featured.Handle(r.Context(), q)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": cards(items)})
}

func cards(branches []search.Branch) []branchJSON {
	items := make([]branchJSON, 0, len(branches))
	for _, b := range branches {
		items = append(items, branchJSON{
			BranchID: b.BranchID, BusinessName: b.BusinessName, BusinessSlug: b.BusinessSlug,
			BranchName: b.BranchName, Address: b.Address, City: b.City, Reference: b.Reference,
			Lat: b.Lat, Lng: b.Lng, DistanceM: int(math.Round(b.DistanceM)),
			RatingAvg: b.RatingAvg, RatingCount: b.RatingCount, Verified: b.Verified, Featured: b.Featured,
			CategorySlug: b.CategorySlug, CategoryName: b.CategoryName,
		})
	}
	return items
}

// fail answers a use case error: a range error is the client's, anything else is
// logged with the request id and reported as a generic 500.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var qe *search.QueryError
	if errors.As(err, &qe) {
		invalid(w, qe.Field, "out_of_range")
		return
	}
	h.log.Error("search failed", "request_id", httpx.RequestID(r.Context()), "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
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
