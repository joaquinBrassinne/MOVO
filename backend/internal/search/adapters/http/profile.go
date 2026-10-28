package searchhttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

// CodeBusinessNotFound is the code of a business that does not exist or is not public.
const CodeBusinessNotFound = "business_not_found"

// WithProfile enables GET /businesses/{slug}, the data behind the public page.
func (h *Handlers) WithProfile(profile *queries.BusinessProfile) *Handlers {
	h.profile = profile
	return h
}

type slotJSON struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type profileBranchJSON struct {
	BranchID    int64       `json:"branch_id"`
	Name        string      `json:"name"`
	Address     string      `json:"address"`
	City        string      `json:"city"`
	Reference   string      `json:"reference"`
	BookingURL  *string     `json:"external_booking_url"`
	Lat         float64     `json:"lat"`
	Lng         float64     `json:"lng"`
	RatingAvg   float64     `json:"rating_avg"`
	RatingCount int         `json:"rating_count"`
	Hours       []slotJSON  `json:"hours"`
	OpenNow     openNowJSON `json:"open_now"`
}

type profileServiceJSON struct {
	ID          int64      `json:"id"`
	BranchID    *int64     `json:"branch_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Price       *int       `json:"price"`
	Schedule    []slotJSON `json:"schedule"`
}

type categoryJSON struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// profileJSON is the public page of a business. It has no WhatsApp number: the
// button goes through /r/wa/{branch_id}, which counts the click.
type profileJSON struct {
	Name        string               `json:"name"`
	Slug        string               `json:"slug"`
	Description string               `json:"description"`
	Category    categoryJSON         `json:"category"`
	Verified    bool                 `json:"verified"`
	Branches    []profileBranchJSON  `json:"branches"`
	Services    []profileServiceJSON `json:"services"`
	Benefits    []profileBenefitJSON `json:"benefits"`
	Images      []string             `json:"images"`
}

func (h *Handlers) handleProfile(w http.ResponseWriter, r *http.Request) {
	view, err := h.profile.Handle(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, search.ErrBusinessNotFound) {
		httpx.WriteError(w, http.StatusNotFound, CodeBusinessNotFound, "business not found", nil)
		return
	}
	if err != nil {
		h.log.Error("business profile failed", "request_id", httpx.RequestID(r.Context()), "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, profileBody(view))
}

func profileBody(v queries.ProfileView) profileJSON {
	out := profileJSON{
		Name: v.Name, Slug: v.Slug, Description: v.Description, Verified: v.Verified,
		Category: categoryJSON{Slug: v.CategorySlug, Name: v.CategoryName},
		Branches: make([]profileBranchJSON, 0, len(v.Branches)),
		Services: make([]profileServiceJSON, 0, len(v.Services)),
		Benefits: benefitsJSON(v.Benefits),
		Images:   v.ImageURLs,
	}
	for _, b := range v.Branches {
		out.Branches = append(out.Branches, profileBranchJSON{
			BranchID: b.BranchID, Name: b.Name, Address: b.Address, City: b.City, Reference: b.Reference, BookingURL: nilIfEmpty(b.BookingURL),
			Lat: b.Lat, Lng: b.Lng, RatingAvg: b.RatingAvg, RatingCount: b.RatingCount, Hours: slots(b.Hours),
			OpenNow: openNow(v.OpenNow[b.BranchID]),
		})
	}
	for _, s := range v.Services {
		item := profileServiceJSON{ID: s.ID, Name: s.Name, Description: s.Description, Schedule: slots(s.Schedule)}
		if s.BranchID != 0 {
			item.BranchID = &s.BranchID
		}
		if s.Price != 0 {
			item.Price = &s.Price
		}
		out.Services = append(out.Services, item)
	}
	return out
}

func slots(in []search.Slot) []slotJSON {
	out := make([]slotJSON, 0, len(in))
	for _, s := range in {
		out = append(out, slotJSON{Weekday: s.Weekday, Start: s.Start, End: s.End})
	}
	return out
}

// openNowJSON says whether the branch is open at the time of the request, in the
// Córdoba zone. Open: closes_at is when the stretch ends. Closed: opens_at and
// opens_weekday say when it opens next, and are null when there are no slots.
type openNowJSON struct {
	Open         bool    `json:"open"`
	ClosesAt     *string `json:"closes_at"`
	OpensAt      *string `json:"opens_at"`
	OpensWeekday *int    `json:"opens_weekday"`
}

func openNow(s search.OpenStatus) openNowJSON {
	switch {
	case s.Open:
		return openNowJSON{Open: true, ClosesAt: &s.ClosesAt}
	case s.HasNext:
		return openNowJSON{OpensAt: &s.OpensAt, OpensWeekday: &s.OpensWeekday}
	}
	return openNowJSON{}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// profileBenefitJSON is a benefit offered right now. The fields of the other types are
// null, and there is nothing about the business's monthly limit.
type profileBenefitJSON struct {
	ID         int64   `json:"id"`
	Type       string  `json:"type"`
	Percent    *int    `json:"percent"`
	AmountARS  *int    `json:"amount_ars"`
	Title      *string `json:"title"`
	Conditions *string `json:"conditions"`
	ShortLabel string  `json:"short_label"`
}

func benefitsJSON(in []search.ProfileBenefit) []profileBenefitJSON {
	out := make([]profileBenefitJSON, 0, len(in))
	for _, b := range in {
		item := profileBenefitJSON{ID: b.ID, Type: b.Kind, ShortLabel: b.ShortLabel, Conditions: nilIfEmpty(b.Conditions)}
		switch b.Kind {
		case search.BenefitPercent:
			item.Percent = &b.Percent
		case search.BenefitAmount:
			item.AmountARS = &b.AmountARS
		default:
			item.Title = nilIfEmpty(b.Title)
		}
		out = append(out, item)
	}
	return out
}
