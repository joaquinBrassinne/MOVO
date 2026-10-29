// Package reviewshttp exposes the review use cases: the public endpoint a customer uses
// with their access token.
package reviewshttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// Codes specific to the review endpoints.
const (
	CodeValidationFailed   = "validation_failed"
	CodeCouponNotFound     = "coupon_not_found"
	CodeBranchNotFound     = "branch_not_found"
	CodeCouponNotRedeemed  = "coupon_not_redeemed"
	CodeReviewWindowClosed = "review_window_closed"
	CodeAlreadyReviewed    = "already_reviewed"
)

// Deps are the use cases the handlers need.
type Deps struct {
	Log *slog.Logger

	// Create serves the customer's review; without it the route is not mounted.
	Create *commands.CreateReview
	// List serves the public reviews of a branch; without it the route is not mounted.
	List *queries.ListReviews
}

// Handlers serves the review endpoints.
type Handlers struct {
	Deps
	log *slog.Logger
}

// New wires the handlers. A nil logger discards logs.
func New(deps Deps) *Handlers {
	log := deps.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Handlers{Deps: deps, log: log}
}

// Mount registers the routes. They are public: the customer has no account. Writing a
// review is authorized by the access token in the path, so that answer is never cached.
func (h *Handlers) Mount(r chi.Router) {
	if h.Create != nil {
		r.With(noStore).Post("/access/{token}/reviews", h.handleCreate)
	}
	if h.List != nil {
		r.Get("/branches/{id}/reviews", h.handleList)
	}
}

// listedJSON is a review as the public reads it: no id, no customer, no coupon.
type listedJSON struct {
	Rating      int       `json:"rating"`
	Comment     string    `json:"comment,omitempty"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// listJSON is a page. NextCursor is null on the last page.
type listJSON struct {
	Reviews    []listedJSON `json:"reviews"`
	NextCursor *string      `json:"next_cursor"`
}

func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	branchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || branchID <= 0 {
		h.fail(w, r, reviews.ErrBranchNotFound)
		return
	}
	in := queries.ListReviewsInput{BranchID: branchID, Cursor: r.URL.Query().Get("cursor")}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			h.fail(w, r, queries.ErrLimitInvalid)
			return
		}
		in.Limit = limit
	}

	page, err := h.List.Handle(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := listJSON{Reviews: make([]listedJSON, len(page.Reviews))}
	for i, review := range page.Reviews {
		out.Reviews[i] = listedJSON{Rating: review.Rating, Comment: review.Comment, DisplayName: review.DisplayName, CreatedAt: review.CreatedAt.UTC()}
	}
	if page.NextCursor != "" {
		out.NextCursor = &page.NextCursor
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// createRequest is the form: a rating, an optional comment and whether to show the name.
// There is no name nor branch: both come from the coupon.
type createRequest struct {
	Rating   int    `json:"rating"`
	Comment  string `json:"comment"`
	ShowName *bool  `json:"show_name"` // on unless the customer turns it off
}

// createdJSON is the review as it will be shown. It carries no id, customer nor coupon.
type createdJSON struct {
	Rating      int       `json:"rating"`
	Comment     string    `json:"comment,omitempty"`
	DisplayName string    `json:"display_name"`
	ShowName    bool      `json:"show_name"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handlers) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	showName := req.ShowName == nil || *req.ShowName

	got, err := h.Create.Handle(r.Context(), commands.CreateReviewInput{
		Token: chi.URLParam(r, "token"), Rating: req.Rating, Comment: req.Comment, ShowName: showName,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, createdJSON{
		Rating: got.Rating, Comment: got.Comment, DisplayName: got.DisplayName, ShowName: got.ShowName, CreatedAt: got.CreatedAt.UTC(),
	})
}

// noStore keeps the answers, which depend on a token in the path, out of shared caches.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// fail maps a use case error to its HTTP answer. Anything unexpected is logged with the
// request id and reported as a generic 500: the error text may name internals and never
// reaches the client.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, reviews.ErrCouponNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeCouponNotFound, "coupon not found", nil)
		return
	case errors.Is(err, reviews.ErrBranchNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeBranchNotFound, "branch not found", nil)
		return
	case errors.Is(err, domain.ErrCursorInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the cursor is not valid",
			map[string]string{"field": "cursor", "reason": "invalid"})
		return
	case errors.Is(err, queries.ErrLimitInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the limit is not valid",
			map[string]string{"field": "limit", "reason": "invalid"})
		return
	case errors.Is(err, domain.ErrNotRedeemed):
		httpx.WriteError(w, http.StatusConflict, CodeCouponNotRedeemed, "the coupon has to be redeemed before reviewing", nil)
		return
	case errors.Is(err, domain.ErrWindowClosed):
		httpx.WriteError(w, http.StatusConflict, CodeReviewWindowClosed, "the 30 days to review this coupon are over", nil)
		return
	case errors.Is(err, reviews.ErrAlreadyReviewed):
		httpx.WriteError(w, http.StatusConflict, CodeAlreadyReviewed, "this coupon was already reviewed", nil)
		return
	case errors.Is(err, domain.ErrRatingInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the rating is not valid",
			map[string]string{"field": "rating", "reason": "out_of_range"})
		return
	}
	var text *domain.TextError
	if errors.As(err, &text) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+text.Field+" is not valid",
			map[string]string{"field": text.Field, "reason": string(text.Reason)})
		return
	}

	h.log.Error("reviews request failed", "request_id", httpx.RequestID(r.Context()), "path", r.URL.Path, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
}
