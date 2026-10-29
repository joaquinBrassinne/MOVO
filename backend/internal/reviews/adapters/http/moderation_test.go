package reviewshttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	reviewshttp "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
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
