package reviewshttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	reviewshttp "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// verifier trusts three fixed tokens: an owner, a super admin and nobody else.
type verifier struct{}

func (verifier) Verify(token string) (httpx.Principal, error) {
	switch token {
	case "owner":
		return httpx.Principal{UserID: 7, Role: "business_admin"}, nil
	case "admin":
		return httpx.Principal{UserID: 1, Role: "super_admin"}, nil
	}
	return httpx.Principal{}, errors.New("unknown token")
}

type owners struct{ err error }

func (o owners) EditableBusinessOf(context.Context, int64) (int64, error) { return 42, o.err }

// moderator records what it was asked and answers like the store would.
type moderator struct {
	err                  error
	gotActor, gotBiz     int64
	gotReview            int64
	gotStatus            domain.Status
	reportRan, statusRan bool
}

func (m *moderator) Report(_ context.Context, actorID, businessID, reviewID int64) (reviews.ModeratedReview, error) {
	m.reportRan = true
	m.gotActor, m.gotBiz, m.gotReview = actorID, businessID, reviewID
	return reviews.ModeratedReview{ID: reviewID, Status: domain.StatusReported}, m.err
}

func (m *moderator) SetStatus(_ context.Context, actorID, reviewID int64, status domain.Status) (reviews.ModeratedReview, error) {
	m.statusRan = true
	m.gotActor, m.gotReview, m.gotStatus = actorID, reviewID, status
	return reviews.ModeratedReview{ID: reviewID, Status: status}, m.err
}

func call(t *testing.T, o owners, m *moderator, method, path, token, body string) (int, http.Header, map[string]any) {
	t.Helper()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{
		Verifier:  verifier{},
		Report:    commands.NewReportReview(o, m),
		SetStatus: commands.NewSetReviewStatus(m),
	}).Mount(router)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, rec.Header(), out
}

// lister answers both moderation lists with one review and records what it was asked.
type lister struct {
	err        error
	gotBiz     int64
	gotStatus  domain.Status
	gotLimit   int
	listRan    bool
	pageOfOne  reviews.ModerationReview
	pageOfMore int
}

func (l *lister) ListOfBusiness(_ context.Context, businessID int64, _ *domain.Cursor, limit int) ([]reviews.ModerationReview, error) {
	l.listRan, l.gotBiz, l.gotLimit = true, businessID, limit
	return l.page(limit), l.err
}

func (l *lister) ListByStatus(_ context.Context, status domain.Status, _ *domain.Cursor, limit int) ([]reviews.ModerationReview, error) {
	l.listRan, l.gotStatus, l.gotLimit = true, status, limit
	return l.page(limit), l.err
}

// page returns pageOfMore reviews, or one when it is not set.
func (l *lister) page(limit int) []reviews.ModerationReview {
	n := max(l.pageOfMore, 1)
	n = min(n, limit)
	out := make([]reviews.ModerationReview, n)
	for i := range out {
		out[i] = l.pageOfOne
		out[i].ID = int64(100 - i)
	}
	return out
}

func listCall(t *testing.T, o owners, l *lister, path, token string) (int, http.Header, map[string]any) {
	t.Helper()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{
		Verifier:  verifier{},
		ListOwn:   queries.NewListOwnReviews(o, l),
		ListQueue: queries.NewListReviewsToModerate(l),
	}).Mount(router)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, rec.Header(), out
}

func TestOwnerListsTheReviewsOfTheirBusinessWithTheIdToReport(t *testing.T) {
	t.Parallel()

	l := &lister{pageOfOne: reviews.ModerationReview{
		BranchID: 3, BranchName: "Centro", BusinessID: 42, BusinessName: "Studio", Rating: 4,
		Comment: "Muy bueno", DisplayName: "Ana G.", Status: domain.StatusReported,
	}}
	status, header, body := listCall(t, owners{}, l, "/me/business/reviews", "owner")
	if status != http.StatusOK || header.Get("Cache-Control") != "no-store" {
		t.Fatalf("answer = %d (Cache-Control %q), want 200 and no-store", status, header.Get("Cache-Control"))
	}
	if l.gotBiz != 42 || l.gotLimit != queries.DefaultLimit+1 {
		t.Errorf("reader got business %d and limit %d, want 42 (the owner's) and %d", l.gotBiz, l.gotLimit, queries.DefaultLimit+1)
	}
	list, _ := body["reviews"].([]any)
	first, _ := list[0].(map[string]any)
	if len(list) != 1 || first["id"] != float64(100) || first["status"] != "reported" || first["branch_name"] != "Centro" || first["comment"] != "Muy bueno" {
		t.Errorf("body = %v, want the review with its id, state, branch and text", body)
	}
	if _, leaked := first["customer_id"]; leaked || body["next_cursor"] != nil {
		t.Errorf("body = %v, want no customer and a null next_cursor on the last page", body)
	}
}

func TestTheAdminQueueIsReportedUnlessAskedForHidden(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]domain.Status{
		"/admin/reviews":                 domain.StatusReported,
		"/admin/reviews?status=reported": domain.StatusReported,
		"/admin/reviews?status=hidden":   domain.StatusHidden,
	} {
		l := &lister{}
		if status, _, _ := listCall(t, owners{}, l, path, "admin"); status != http.StatusOK || l.gotStatus != want {
			t.Errorf("%s: %d, queue %q; want 200 and %q", path, status, l.gotStatus, want)
		}
	}
}

func TestAListWithMoreThanOnePageGivesTheCursorOfTheNext(t *testing.T) {
	t.Parallel()

	l := &lister{pageOfMore: 100, pageOfOne: reviews.ModerationReview{CreatedAt: time.Date(2028, 3, 2, 10, 0, 0, 0, time.UTC)}}
	_, _, body := listCall(t, owners{}, l, "/admin/reviews?limit=2", "admin")
	list, _ := body["reviews"].([]any)
	if len(list) != 2 || body["next_cursor"] == nil {
		t.Errorf("body = %v, want 2 reviews and the cursor of the next page", body)
	}
}

func TestModerationListsAreRefusedWithAStableCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, path, token string
		owners            owners
		want              int
		code              string
	}{
		{"owner list without a token", "/me/business/reviews", "", owners{}, http.StatusUnauthorized, ""},
		{"the admin cannot list as an owner", "/me/business/reviews", "admin", owners{}, http.StatusForbidden, ""},
		{"an owner cannot read the queue", "/admin/reviews", "owner", owners{}, http.StatusForbidden, ""},
		{"queue without a token", "/admin/reviews", "", owners{}, http.StatusUnauthorized, ""},
		{"owner without a business", "/me/business/reviews", "owner", owners{err: reviews.ErrBusinessNotFound}, http.StatusNotFound, "business_not_found"},
		{"a queue that does not exist", "/admin/reviews?status=visible", "admin", owners{}, http.StatusUnprocessableEntity, "validation_failed"},
		{"page size above the maximum", "/admin/reviews?limit=51", "admin", owners{}, http.StatusUnprocessableEntity, "validation_failed"},
		{"page size that is not a number", "/me/business/reviews?limit=x", "owner", owners{}, http.StatusUnprocessableEntity, "validation_failed"},
		{"a cursor that is not ours", "/admin/reviews?cursor=junk", "admin", owners{}, http.StatusUnprocessableEntity, "validation_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := &lister{}
			status, _, body := listCall(t, tt.owners, l, tt.path, tt.token)
			if status != tt.want || (tt.code != "" && codeOf(body) != tt.code) {
				t.Errorf("answer = %d %v, want %d %s", status, body, tt.want, tt.code)
			}
			if tt.want == http.StatusUnauthorized || tt.want == http.StatusForbidden {
				if l.listRan {
					t.Error("the reader ran for a caller that must not reach it")
				}
			}
		})
	}
}

func codeOf(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestOwnerReportsAReviewOfTheirBusinessFromTheirSession(t *testing.T) {
	t.Parallel()

	m := &moderator{}
	status, header, body := call(t, owners{}, m, http.MethodPost, "/me/business/reviews/99/report", "owner", "")
	if status != http.StatusOK || body["id"] != float64(99) || body["status"] != "reported" {
		t.Fatalf("answer = %d %v, want 200 with the review reported", status, body)
	}
	if m.gotActor != 7 || m.gotBiz != 42 || m.gotReview != 99 {
		t.Errorf("moderator got actor %d, business %d, review %d; want 7 (the token), 42 (the owner's business), 99", m.gotActor, m.gotBiz, m.gotReview)
	}
	if header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", header.Get("Cache-Control"))
	}
}

func TestOnlyTheRightRoleReachesEachModerationRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, method, path, token, body string
		want                            int
	}{
		{"report without a token", http.MethodPost, "/me/business/reviews/1/report", "", "", http.StatusUnauthorized},
		{"report with a token we do not trust", http.MethodPost, "/me/business/reviews/1/report", "forged", "", http.StatusUnauthorized},
		{"the admin cannot report as an owner", http.MethodPost, "/me/business/reviews/1/report", "admin", "", http.StatusForbidden},
		{"hide without a token", http.MethodPut, "/admin/reviews/1/status", "", `{"status":"hidden"}`, http.StatusUnauthorized},
		{"an owner cannot hide a review", http.MethodPut, "/admin/reviews/1/status", "owner", `{"status":"hidden"}`, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &moderator{}
			status, _, _ := call(t, owners{}, m, tt.method, tt.path, tt.token, tt.body)
			if status != tt.want {
				t.Errorf("status = %d, want %d", status, tt.want)
			}
			if m.reportRan || m.statusRan {
				t.Error("the moderator ran for a caller that must not reach it")
			}
		})
	}
}

func TestReportIsRefusedWithAStableCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		owners     owners
		moderator  *moderator
		path       string
		wantStatus int
		wantCode   string
	}{
		{"review of another business", owners{}, &moderator{err: reviews.ErrReviewNotFound}, "/me/business/reviews/9/report", http.StatusNotFound, "review_not_found"},
		{"id is not a number", owners{}, &moderator{}, "/me/business/reviews/abc/report", http.StatusNotFound, "review_not_found"},
		{"id is zero", owners{}, &moderator{}, "/me/business/reviews/0/report", http.StatusNotFound, "review_not_found"},
		{"owner without a business", owners{err: reviews.ErrBusinessNotFound}, &moderator{}, "/me/business/reviews/9/report", http.StatusNotFound, "business_not_found"},
		{"deleted business", owners{err: reviews.ErrBusinessNotEditable}, &moderator{}, "/me/business/reviews/9/report", http.StatusConflict, "business_deleted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, _, body := call(t, tt.owners, tt.moderator, http.MethodPost, tt.path, "owner", "")
			if status != tt.wantStatus || codeOf(body) != tt.wantCode {
				t.Errorf("answer = %d %v, want %d %s", status, body, tt.wantStatus, tt.wantCode)
			}
		})
	}
}

func TestAdminHidesAndRestoresOnBehalfOfTheTokenUser(t *testing.T) {
	t.Parallel()

	for _, want := range []string{"hidden", "visible"} {
		m := &moderator{}
		status, header, body := call(t, owners{}, m, http.MethodPut, "/admin/reviews/99/status", "admin", `{"status":"`+want+`"}`)
		if status != http.StatusOK || body["id"] != float64(99) || body["status"] != want {
			t.Fatalf("%s: answer = %d %v, want 200 with the status applied", want, status, body)
		}
		if m.gotActor != 1 || m.gotReview != 99 || string(m.gotStatus) != want {
			t.Errorf("%s: moderator got actor %d, review %d, status %q; want actor 1 (the token)", want, m.gotActor, m.gotReview, m.gotStatus)
		}
		if header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", want, header.Get("Cache-Control"))
		}
	}
}

func TestAdminStatusIsRefusedWithAStableCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		moderator  *moderator
		path, body string
		wantStatus int
		wantCode   string
	}{
		{"reported is the owner's move", &moderator{}, "/admin/reviews/9/status", `{"status":"reported"}`, http.StatusUnprocessableEntity, "validation_failed"},
		{"an invented status", &moderator{}, "/admin/reviews/9/status", `{"status":"deleted"}`, http.StatusUnprocessableEntity, "validation_failed"},
		{"no status", &moderator{}, "/admin/reviews/9/status", `{}`, http.StatusUnprocessableEntity, "validation_failed"},
		{"unknown field", &moderator{}, "/admin/reviews/9/status", `{"status":"hidden","by":1}`, http.StatusBadRequest, "invalid_request"},
		{"review that does not exist", &moderator{err: reviews.ErrReviewNotFound}, "/admin/reviews/9/status", `{"status":"hidden"}`, http.StatusNotFound, "review_not_found"},
		{"restoring a purged review", &moderator{err: reviews.ErrReviewPurged}, "/admin/reviews/9/status", `{"status":"visible"}`, http.StatusConflict, "review_purged"},
		{"id is not a number", &moderator{}, "/admin/reviews/abc/status", `{"status":"hidden"}`, http.StatusNotFound, "review_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, _, body := call(t, owners{}, tt.moderator, http.MethodPut, tt.path, "admin", tt.body)
			if status != tt.wantStatus || codeOf(body) != tt.wantCode {
				t.Errorf("answer = %d %v, want %d %s", status, body, tt.wantStatus, tt.wantCode)
			}
		})
	}
}

func TestModerationRoutesAreNotMountedWithoutTheirUseCases(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{Verifier: verifier{}}).Mount(router)
	for _, c := range []struct{ method, path, token string }{
		{http.MethodPost, "/me/business/reviews/1/report", "owner"},
		{http.MethodPut, "/admin/reviews/1/status", "admin"},
	} {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(`{"status":"hidden"}`))
		req.Header.Set("Authorization", "Bearer "+c.token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", c.method, c.path, rec.Code)
		}
	}
}
