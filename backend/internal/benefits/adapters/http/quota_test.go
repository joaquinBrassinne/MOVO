package benefitshttp_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestOwnerReadsAndSetsTheCouponQuotaOfTheBusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	s.owner("owner@example.com", "active")
	owner := "token-owner@example.com"
	resets := domain.DateInArgentina(domain.QuotaResetsOn(time.Now()))

	// A new business starts without a limit.
	got := s.do(http.MethodGet, "/me/business/coupon-quota", owner, nil)
	if got.status != http.StatusOK || got.body["limit"] != nil || got.body["generated_this_month"] != float64(0) || got.body["resets_on"] != resets {
		t.Fatalf("initial quota = %d %v, want no limit, 0 generated and resets_on %s", got.status, got.body, resets)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got.header.Get("Cache-Control"))
	}

	set := s.do(http.MethodPut, "/me/business/coupon-quota", owner, map[string]any{"limit": 30})
	if set.status != http.StatusOK || set.body["limit"] != float64(30) {
		t.Fatalf("set = %d %v", set.status, set.body)
	}
	if again := s.do(http.MethodGet, "/me/business/coupon-quota", owner, nil); again.body["limit"] != float64(30) {
		t.Errorf("read back = %v, want 30", again.body)
	}
	if none := s.do(http.MethodPut, "/me/business/coupon-quota", owner, map[string]any{"limit": nil}); none.status != http.StatusOK || none.body["limit"] != nil {
		t.Errorf("remove the limit = %d %v, want null", none.status, none.body)
	}
}

func TestSettingTheQuotaRefusesWhatIsNotAllowed(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 10)
	owner := "token-owner@example.com"
	for i, phone := range []string{"5491100000001", "5491100000002", "5491100000003"} {
		if got := s.do(http.MethodPost, path, "", customer("Cliente", phone)); got.status != http.StatusCreated {
			t.Fatalf("generate %d = %d %v", i, got.status, got.body)
		}
	}

	below := s.do(http.MethodPut, "/me/business/coupon-quota", owner, map[string]any{"limit": 2})
	details, _ := below.body["error"].(map[string]any)["details"].(map[string]any)
	if below.status != http.StatusUnprocessableEntity || details["field"] != "limit" || details["reason"] != "below_generated" || details["generated_this_month"] != float64(3) {
		t.Errorf("limit below the generated = %d %v, want 422 naming the 3 already generated", below.status, below.body)
	}
	if ok := s.do(http.MethodPut, "/me/business/coupon-quota", owner, map[string]any{"limit": 3}); ok.status != http.StatusOK {
		t.Errorf("limit equal to the generated = %d %v, want 200", ok.status, ok.body)
	}

	for name, body := range map[string]map[string]any{
		"zero":     {"limit": 0},
		"negative": {"limit": -5},
		"decimal":  {"limit": 1.5},
		"text":     {"limit": "mucho"},
		"absent":   {},
	} {
		got := s.do(http.MethodPut, "/me/business/coupon-quota", owner, body)
		if got.status != http.StatusUnprocessableEntity || got.errorCode() != "validation_failed" || got.errorField() != "limit" {
			t.Errorf("%s = %d %q field %q, want 422 validation_failed on limit", name, got.status, got.errorCode(), got.errorField())
		}
	}
	if got := s.do(http.MethodPut, "/me/business/coupon-quota", owner, map[string]any{"limit": 9, "business_id": 1}); got.status != http.StatusBadRequest {
		t.Errorf("forged business_id = %d, want 400 (unknown field)", got.status)
	}
	if after := s.do(http.MethodGet, "/me/business/coupon-quota", owner, nil); after.body["limit"] != float64(3) {
		t.Errorf("after the refusals = %v, want the limit untouched at 3", after.body)
	}
}

func TestCouponQuotaNeedsAnOwnerWithABusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	_, admin := s.user("admin@example.com", "super_admin")
	_, bare := s.user("bare@example.com", "business_admin")

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		if got := s.do(method, "/me/business/coupon-quota", "", map[string]any{"limit": 5}); got.status != http.StatusUnauthorized {
			t.Errorf("%s without a token = %d, want 401", method, got.status)
		}
		if got := s.do(method, "/me/business/coupon-quota", admin, map[string]any{"limit": 5}); got.status != http.StatusForbidden {
			t.Errorf("%s as super admin = %d, want 403", method, got.status)
		}
		if got := s.do(method, "/me/business/coupon-quota", bare, map[string]any{"limit": 5}); got.status != http.StatusNotFound || got.errorCode() != "business_not_found" {
			t.Errorf("%s without a business = %d %q, want 404 business_not_found", method, got.status, got.errorCode())
		}
	}
}

func TestOnlyOneOfTwoSimultaneousGenerationsGetsTheLastCoupon(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 1)

	statuses := make([]int, 2)
	codes := make([]string, 2)
	var wg sync.WaitGroup
	for i, phone := range []string{"5491100000001", "5491100000002"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := s.do(http.MethodPost, path, "", customer("Cliente", phone))
			statuses[i], codes[i] = got.status, got.errorCode()
		}()
	}
	wg.Wait()

	created, refused := 0, 0
	for i, status := range statuses {
		switch {
		case status == http.StatusCreated:
			created++
		case status == http.StatusConflict && codes[i] == "quota_reached":
			refused++
		}
	}
	if created != 1 || refused != 1 {
		t.Errorf("statuses %v codes %v; want exactly one 201 and one 409 quota_reached", statuses, codes)
	}
}
