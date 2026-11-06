package reviewshttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	reviewshttp "github.com/joaquinBrassinne/MOVO/internal/reviews/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

var (
	redeemedAt = time.Date(2028, 3, 1, 10, 0, 0, 0, time.UTC)
	now        = redeemedAt.Add(24 * time.Hour)
)

// memory is both ports in memory: one coupon per token and the reviews already stored.
type memory struct {
	coupons  map[string]domain.Redemption
	reviewed map[int64]bool
}

func (m *memory) ByTokenHash(_ context.Context, hash []byte) (domain.Redemption, error) {
	for token, redemption := range m.coupons {
		if string(domain.HashAccessToken(token)) == string(hash) {
			return redemption, nil
		}
	}
	return domain.Redemption{}, reviews.ErrCouponNotFound
}

func (m *memory) Create(_ context.Context, review domain.Review) error {
	if m.reviewed[review.CouponID] {
		return reviews.ErrAlreadyReviewed
	}
	m.reviewed[review.CouponID] = true
	return nil
}

type answer struct {
	status int
	header http.Header
	body   map[string]any
}

func post(t *testing.T, m *memory, token, body string) answer {
	t.Helper()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{Create: commands.NewCreateReview(m, m, func() time.Time { return now })}).Mount(router)
	req := httptest.NewRequest(http.MethodPost, "/access/"+token+"/reviews", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	out := answer{status: rec.Code, header: rec.Header()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func newMemory() *memory {
	return &memory{
		reviewed: map[int64]bool{},
		coupons: map[string]domain.Redemption{
			"redeemed-token": {CouponID: 1, BranchID: 3, BusinessID: 2, CustomerID: 9, CustomerName: "Ana García", RedeemedAt: redeemedAt},
			"pending-token":  {CouponID: 2, BranchID: 3, BusinessID: 2, CustomerID: 9, CustomerName: "Ana García"},
			"old-token":      {CouponID: 3, BranchID: 3, BusinessID: 2, CustomerID: 9, CustomerName: "Ana García", RedeemedAt: now.Add(-31 * 24 * time.Hour)},
		},
	}
}

func errorOf(a answer) (code string, details map[string]any) {
	e, _ := a.body["error"].(map[string]any)
	code, _ = e["code"].(string)
	details, _ = e["details"].(map[string]any)
	return code, details
}

func TestCustomerReviewsTheCouponOfTheirTokenWithoutAnAccount(t *testing.T) {
	t.Parallel()

	got := post(t, newMemory(), "redeemed-token", `{"rating":5,"comment":"Muy buena clase","show_name":true}`)
	if got.status != http.StatusCreated {
		t.Fatalf("status = %d %v, want 201", got.status, got.body)
	}
	if got.body["rating"] != float64(5) || got.body["comment"] != "Muy buena clase" || got.body["display_name"] != "Ana G." || got.body["show_name"] != true {
		t.Errorf("body = %v, want the review as it will be shown", got.body)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store: the token in the path is the credential", got.header.Get("Cache-Control"))
	}
	raw, _ := json.Marshal(got.body)
	for _, leak := range []string{"5491187654321", "phone", "customer", "token", "coupon", "García"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("body %s leaks %q", raw, leak)
		}
	}
}

func TestShowNameIsOnUnlessTheCustomerTurnsItOff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, body, want string
	}{
		{"omitted", `{"rating":4}`, "Ana G."},
		{"on", `{"rating":4,"show_name":true}`, "Ana G."},
		{"off", `{"rating":4,"show_name":false}`, "Cliente de MOVO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := post(t, newMemory(), "redeemed-token", tt.body)
			if got.status != http.StatusCreated || got.body["display_name"] != tt.want {
				t.Errorf("answer = %d %v, want 201 shown as %q", got.status, got.body, tt.want)
			}
		})
	}
}

func TestReviewIsRefusedWithAStableCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		token      string
		body       string
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{"unknown token", "nobody", `{"rating":5}`, http.StatusNotFound, "coupon_not_found", ""},
		{"coupon not redeemed", "pending-token", `{"rating":5}`, http.StatusConflict, "coupon_not_redeemed", ""},
		{"window over", "old-token", `{"rating":5}`, http.StatusConflict, "review_window_closed", ""},
		{"rating missing", "redeemed-token", `{}`, http.StatusUnprocessableEntity, "validation_failed", "rating"},
		{"rating too high", "redeemed-token", `{"rating":6}`, http.StatusUnprocessableEntity, "validation_failed", "rating"},
		{"comment with a link", "redeemed-token", `{"rating":5,"comment":"mirá www.ejemplo.com"}`, http.StatusUnprocessableEntity, "validation_failed", "comment"},
		{"unknown field", "redeemed-token", `{"rating":5,"branch_id":99}`, http.StatusBadRequest, "invalid_request", ""},
		{"not json", "redeemed-token", `rating=5`, http.StatusBadRequest, "invalid_request", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := post(t, newMemory(), tt.token, tt.body)
			code, details := errorOf(got)
			if got.status != tt.wantStatus || code != tt.wantCode {
				t.Fatalf("answer = %d %v, want %d %s", got.status, got.body, tt.wantStatus, tt.wantCode)
			}
			if tt.wantField != "" && details["field"] != tt.wantField {
				t.Errorf("details = %v, want field %q", details, tt.wantField)
			}
		})
	}
}

func TestSecondReviewOfTheSameCouponIsRefused(t *testing.T) {
	t.Parallel()

	m := newMemory()
	if first := post(t, m, "redeemed-token", `{"rating":5}`); first.status != http.StatusCreated {
		t.Fatalf("first = %d %v", first.status, first.body)
	}
	got := post(t, m, "redeemed-token", `{"rating":1}`)
	if code, _ := errorOf(got); got.status != http.StatusConflict || code != "already_reviewed" {
		t.Errorf("second = %d %v, want 409 already_reviewed", got.status, got.body)
	}
}

func TestRouteIsNotMountedWithoutTheUseCase(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{}).Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/access/x/reviews", strings.NewReader(`{"rating":5}`)))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
