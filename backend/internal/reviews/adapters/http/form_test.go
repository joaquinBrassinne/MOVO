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
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/queries"
)

// formShelf answers the one coupon it holds, whatever the hash, or a coupon that is not
// found when err is set.
type formShelf struct {
	data reviews.FormData
	err  error
}

func (f formShelf) FormByTokenHash(context.Context, []byte) (reviews.FormData, error) {
	return f.data, f.err
}

func getForm(t *testing.T, f formShelf, now time.Time) (int, http.Header, map[string]any) {
	t.Helper()

	router := chi.NewRouter()
	reviewshttp.New(reviewshttp.Deps{Form: queries.NewGetReviewForm(f, func() time.Time { return now })}).Mount(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/access/some-token/review", nil))

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, rec.Header(), body
}

func TestFormShowsWhereTheCouponWasRedeemedAndTheNameTheReviewWouldShow(t *testing.T) {
	t.Parallel()

	data := reviews.FormData{BusinessName: "Alma Haus", BranchName: "Palermo Soho", City: "CABA", Code: "K7M29XQH", CustomerName: "Lucía Martínez", RedeemedAt: redeemedAt}
	status, header, body := getForm(t, formShelf{data: data}, redeemedAt.Add(24*time.Hour))
	if status != http.StatusOK || header.Get("Cache-Control") != "no-store" {
		t.Fatalf("answer = %d (Cache-Control %q), want 200 and no-store", status, header.Get("Cache-Control"))
	}
	if body["state"] != "open" || body["business_name"] != "Alma Haus" || body["branch_name"] != "Palermo Soho" || body["city"] != "CABA" ||
		body["code"] != "K7M29XQH" || body["display_name"] != "Lucía M." || body["review_until"] != "2028-03-31T10:00:00Z" {
		t.Errorf("body = %v, want it open, with where it was redeemed, the name \"Lucía M.\" and the day the 30 days end", body)
	}
	raw, _ := json.Marshal(body)
	for _, leak := range []string{"phone", "customer", "token", "Martínez"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("body %s leaks %q", raw, leak)
		}
	}
}

func TestFormStatesFollowTheCoupon(t *testing.T) {
	t.Parallel()

	data := reviews.FormData{BusinessName: "Alma Haus", BranchName: "Palermo Soho", City: "CABA", Code: "K7M29XQH", CustomerName: "Lucía", RedeemedAt: redeemedAt}
	reviewed, notRedeemed := data, data
	reviewed.Reviewed = true
	notRedeemed.RedeemedAt, notRedeemed.BranchName, notRedeemed.City = time.Time{}, "", ""

	for name, tt := range map[string]struct {
		data formShelf
		now  time.Time
		want string
	}{
		"open":          {formShelf{data: data}, redeemedAt.Add(time.Hour), "open"},
		"reviewed":      {formShelf{data: reviewed}, redeemedAt.Add(time.Hour), "reviewed"},
		"window closed": {formShelf{data: data}, redeemedAt.Add(31 * 24 * time.Hour), "window_closed"},
		"not redeemed":  {formShelf{data: notRedeemed}, redeemedAt, "not_redeemed"},
	} {
		status, _, body := getForm(t, tt.data, tt.now)
		if status != http.StatusOK || body["state"] != tt.want {
			t.Errorf("%s = %d %v, want state %q", name, status, body, tt.want)
		}
	}

	_, _, body := getForm(t, formShelf{data: notRedeemed}, redeemedAt)
	if body["branch_name"] != nil || body["redeemed_at"] != nil || body["review_until"] != nil {
		t.Errorf("not redeemed body = %v, want no branch and no time", body)
	}
}

func TestFormOfAnUnknownTokenIsNotFound(t *testing.T) {
	t.Parallel()

	status, _, body := getForm(t, formShelf{err: reviews.ErrCouponNotFound}, redeemedAt)
	if status != http.StatusNotFound || body["error"].(map[string]any)["code"] != "coupon_not_found" {
		t.Errorf("answer = %d %v, want 404 coupon_not_found", status, body)
	}
}
