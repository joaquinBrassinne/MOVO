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
	CodeReviewNotFound     = "review_not_found"
	CodeBusinessNotFound   = "business_not_found"
	CodeBusinessDeleted    = "business_deleted"
	CodeCouponNotRedeemed  = "coupon_not_redeemed"
	CodeReviewWindowClosed = "review_window_closed"
	CodeAlreadyReviewed    = "already_reviewed"
)

// roleOwner and roleSuperAdmin are the plain strings stored in the token, so this
// package needs no knowledge of the identity module.
const (
	roleOwner      = "business_admin"
	roleSuperAdmin = "super_admin"
)

// Deps are the use cases and the token verifier the handlers need.
type Deps struct {
	Log      *slog.Logger
	Verifier httpx.TokenVerifier

	// Create serves the customer's review; without it the route is not mounted.
	Create *commands.CreateReview
	// List serves the public reviews of a branch; without it the route is not mounted.
	List *queries.ListReviews
	// Report serves the owner's report of a review; without it the route is not mounted.
	Report *commands.ReportReview
	// SetStatus serves the super admin's hiding of a review; without it the route is not
	// mounted.
	SetStatus *commands.SetReviewStatus
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
	// The business comes from the session, never from the URL or the body.
	if h.Report != nil {
		owner := r.With(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleOwner), noStore)
		owner.Post("/me/business/reviews/{id}/report", h.handleReport)
	}
	// Only a super_admin token gets past this group; an owner has no way to reach it.
	if h.SetStatus != nil {
		admin := r.With(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleSuperAdmin), noStore)
		admin.Put("/admin/reviews/{id}/status", h.handleSetStatus)
	}
}

// moderatedJSON is the moderation state of a review after a change.
type moderatedJSON struct {
	ID     int64         `json:"id"`
	Status domain.Status `json:"status"`
}

type statusRequest struct {
	Status domain.Status `json:"status"`
}

func (h *Handlers) handleReport(w http.ResponseWriter, r *http.Request) {
	id, ok := h.reviewID(w, r)
	if !ok {
		return
	}
	got, err := h.Report.Handle(r.Context(), callerID(r), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, moderatedJSON{ID: got.ID, Status: got.Status})
}

// handleSetStatus is the super admin's hiding of any review. The actor comes from the token.
func (h *Handlers) handleSetStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.reviewID(w, r)
	if !ok {
		return
	}
	var req statusRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	got, err := h.SetStatus.Handle(r.Context(), callerID(r), id, req.Status)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, moderatedJSON{ID: got.ID, Status: got.Status})
}

// reviewID reads {id}; a malformed one answers like a review that does not exist.
func (h *Handlers) reviewID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		h.fail(w, r, reviews.ErrReviewNotFound)
		return 0, false
	}
	return id, true
}

// callerID is the authenticated caller. RequireRole already guaranteed it exists; the id
// always comes from the token, never from the body or the URL.
func callerID(r *http.Request) int64 {
	principal, _ := httpx.PrincipalFrom(r.Context())
	return principal.UserID
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
	case errors.Is(err, reviews.ErrReviewNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeReviewNotFound, "review not found", nil)
		return
	case errors.Is(err, reviews.ErrBusinessNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeBusinessNotFound, "you have not created a business yet", nil)
		return
	case errors.Is(err, reviews.ErrBusinessNotEditable):
		httpx.WriteError(w, http.StatusConflict, CodeBusinessDeleted, "this business was deleted and can no longer change", nil)
		return
	case errors.Is(err, domain.ErrStatusInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the status is not valid",
			map[string]string{"field": "status", "reason": "invalid"})
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
