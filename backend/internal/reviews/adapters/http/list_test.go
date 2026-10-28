package reviewshttp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	reviewshttp "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

// shelf serves the reviews of branch 1 newest first and knows no other branch, like the
// store, which answers ErrBranchNotFound for a branch the public cannot see.
type shelf struct{ all []reviews.ListedReview }

func (s shelf) ListPublic(_ context.Context, branchID int64, after *domain.Cursor, limit int) ([]reviews.ListedReview, error) {
	if branchID != 1 {
		return nil, reviews.ErrBranchNotFound
	}
	var out []reviews.ListedReview
	for _, r := range s.all {
		if after != nil && (r.CreatedAt.After(after.CreatedAt) || (r.CreatedAt.Equal(after.CreatedAt) && r.ID >= after.ID)) {
			continue
		}
		if len(out) == limit {
			break
		}
		out = append(out, r)
	}
	return out, nil
}

func newShelf(n int) shelf {
	all := make([]reviews.ListedReview, n)
	for i := range all {
		all[i] = reviews.ListedReview{ID: int64(n - i), Rating: 5, DisplayName: fmt.Sprintf("Ana %d", i), CreatedAt: redeemedAt.Add(-time.Duration(i) * time.Hour)}
	}
	return shelf{all: all}
}

func get(t *testing.T, s shelf, path string) (int, http.Header, map[string]any) {
	t.Helper()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{List: queries.NewListReviews(s)}).Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, rec.Header(), body
}

func TestPublicListsTheReviewsOfABranchWithoutIdsOrCustomers(t *testing.T) {
	t.Parallel()

	s := newShelf(1)
	s.all[0].Comment = "Muy buena clase"
	status, _, body := get(t, s, "/branches/1/reviews")
	if status != http.StatusOK {
		t.Fatalf("status = %d %v, want 200", status, body)
	}
	list, _ := body["reviews"].([]any)
	if len(list) != 1 {
		t.Fatalf("reviews = %v, want one", body["reviews"])
	}
	review, _ := list[0].(map[string]any)
	if review["rating"] != float64(5) || review["comment"] != "Muy buena clase" || review["display_name"] != "Ana 0" || review["created_at"] == nil {
		t.Errorf("review = %v, want its rating, comment, name and time", review)
	}
	raw, _ := json.Marshal(body)
	for _, leak := range []string{`"id"`, "customer", "phone", "coupon", "branch_id", "business_id", "status"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("body %s leaks %q", raw, leak)
		}
	}
}

func TestPublicListsAnEmptyBranchAsAnEmptyArray(t *testing.T) {
	t.Parallel()

	status, _, body := get(t, newShelf(0), "/branches/1/reviews")
	list, isArray := body["reviews"].([]any)
	if status != http.StatusOK || !isArray || len(list) != 0 || body["next_cursor"] != nil {
		t.Errorf("answer = %d %v, want 200, [] and a null next_cursor", status, body)
	}
}

func TestPublicPagesWithTheCursorTheServerGave(t *testing.T) {
	t.Parallel()

	s := newShelf(5)
	var seen []string
	next := ""
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("the pages never end")
		}
		path := "/branches/1/reviews?limit=2"
		if next != "" {
			path += "&cursor=" + url.QueryEscape(next)
		}
		status, _, body := get(t, s, path)
		if status != http.StatusOK {
			t.Fatalf("page %d = %d %v", pages, status, body)
		}
		for _, item := range body["reviews"].([]any) {
			seen = append(seen, item.(map[string]any)["display_name"].(string))
		}
		cursor, _ := body["next_cursor"].(string)
		if cursor == "" {
			break
		}
		next = cursor
	}
	if want := "[Ana 0 Ana 1 Ana 2 Ana 3 Ana 4]"; fmt.Sprint(seen) != want {
		t.Errorf("reviews seen = %v, want %s, each once and newest first", seen, want)
	}
}

func TestPublicListIsRefusedWithAStableCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{"unknown branch", "/branches/99/reviews", http.StatusNotFound, "branch_not_found", ""},
		{"id is not a number", "/branches/abc/reviews", http.StatusNotFound, "branch_not_found", ""},
		{"id is zero", "/branches/0/reviews", http.StatusNotFound, "branch_not_found", ""},
		{"cursor we did not issue", "/branches/1/reviews?cursor=garbage", http.StatusUnprocessableEntity, "validation_failed", "cursor"},
		{"limit over the maximum", "/branches/1/reviews?limit=51", http.StatusUnprocessableEntity, "validation_failed", "limit"},
		{"limit negative", "/branches/1/reviews?limit=-1", http.StatusUnprocessableEntity, "validation_failed", "limit"},
		{"limit not a number", "/branches/1/reviews?limit=many", http.StatusUnprocessableEntity, "validation_failed", "limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, _, body := get(t, newShelf(3), tt.path)
			e, _ := body["error"].(map[string]any)
			details, _ := e["details"].(map[string]any)
			if status != tt.wantStatus || e["code"] != tt.wantCode {
				t.Fatalf("answer = %d %v, want %d %s", status, body, tt.wantStatus, tt.wantCode)
			}
			if tt.wantField != "" && details["field"] != tt.wantField {
				t.Errorf("details = %v, want field %q", details, tt.wantField)
			}
		})
	}
}

func TestPublicListRouteIsNotMountedWithoutTheQuery(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{}).Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/branches/1/reviews", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
