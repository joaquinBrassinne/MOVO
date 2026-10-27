package benefitshttp_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// generate has a customer claim the benefit and returns the coupon code.
func (s *server) generate(path, name, phone string) string {
	s.t.Helper()

	got := s.do(http.MethodPost, path, "", customer(name, phone))
	if got.status != http.StatusCreated {
		s.t.Fatalf("generate = %d %v", got.status, got.body)
	}
	return got.body["code"].(string)
}

func TestOwnerRedeemsACouponOnlyOnce(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	owner := "token-owner@example.com"
	code := s.generate(path, "Ana", "5491112345678")

	// The code is accepted as a person types it.
	typed := strings.ToLower(code[:4] + "-" + code[4:])
	got := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": typed})
	if got.status != http.StatusOK || got.body["code"] != code || got.body["customer_name"] != "Ana" {
		t.Fatalf("redeem = %d %v", got.status, got.body)
	}
	benefit, _ := got.body["benefit"].(map[string]any)
	if benefit["type"] != "percent" || benefit["percent"] != float64(15) || got.body["redeemed_at"] == nil || got.body["branch_id"] == nil {
		t.Errorf("redeem body = %v, want the benefit, the time and the branch", got.body)
	}
	if _, leaked := got.body["phone"]; leaked {
		t.Error("the answer must never carry the customer's phone")
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got.header.Get("Cache-Control"))
	}

	again := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": code})
	if again.status != http.StatusConflict || again.errorCode() != "coupon_redeemed" {
		t.Errorf("second redeem = %d %q, want 409 coupon_redeemed", again.status, again.errorCode())
	}
}

func TestRedeemAnswersTheSameForUnknownMalformedAndForeignCodes(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	s.visibleBenefit("mine@example.com", 5)
	theirs := s.visibleBenefit("other@example.com", 5)
	foreign := s.generate(theirs, "Beto", "5491187654321")
	mine := "token-mine@example.com"

	for name, code := range map[string]string{"foreign": foreign, "unknown": "JKLMNPQR", "malformed": "nope", "empty": ""} {
		got := s.do(http.MethodPost, "/me/business/codes/redeem", mine, map[string]any{"code": code})
		if got.status != http.StatusNotFound || got.errorCode() != "coupon_not_found" {
			t.Errorf("%s = %d %q, want 404 coupon_not_found", name, got.status, got.errorCode())
		}
	}

	// The foreign coupon is untouched and its owner can still redeem it.
	if got := s.do(http.MethodPost, "/me/business/codes/redeem", "token-other@example.com", map[string]any{"code": foreign}); got.status != http.StatusOK {
		t.Errorf("owner redeems own coupon = %d %v", got.status, got.body)
	}
}

func TestRedeemNeedsTheBranchWhenThereAreSeveral(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	owner := "token-owner@example.com"
	var business, second int64
	if err := s.pool.QueryRow(t.Context(), `SELECT business_id FROM benefits ORDER BY id LIMIT 1`).Scan(&business); err != nil {
		t.Fatalf("find business: %v", err)
	}
	if err := s.pool.QueryRow(t.Context(), `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Norte', 'Calle 2', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.19, -31.40), 4326)::geography) RETURNING id`, business).Scan(&second); err != nil {
		t.Fatalf("insert branch: %v", err)
	}
	code := s.generate(path, "Ana", "5491112345678")

	got := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": code})
	if got.status != http.StatusUnprocessableEntity || got.errorCode() != "validation_failed" || got.errorField() != "branch_id" {
		t.Errorf("no branch = %d %q field %q, want 422 on branch_id", got.status, got.errorCode(), got.errorField())
	}
	if got := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": code, "branch_id": 999999}); got.status != http.StatusUnprocessableEntity || got.errorField() != "branch_id" {
		t.Errorf("unknown branch = %d field %q, want 422 on branch_id", got.status, got.errorField())
	}
	if got := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": code, "branch_id": second}); got.status != http.StatusOK || got.body["branch_id"] != float64(second) {
		t.Errorf("explicit branch = %d %v, want 200 at branch %d", got.status, got.body, second)
	}
}

func TestRedeemRefusesAnExpiredCoupon(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	code := s.generate(path, "Ana", "5491112345678")
	if _, err := s.pool.Exec(t.Context(), `UPDATE benefit_codes SET expires_at = now() - interval '1 minute' WHERE code = $1`, code); err != nil {
		t.Fatalf("age coupon: %v", err)
	}

	got := s.do(http.MethodPost, "/me/business/codes/redeem", "token-owner@example.com", map[string]any{"code": code})
	if got.status != http.StatusConflict || got.errorCode() != "coupon_expired" {
		t.Errorf("expired = %d %q, want 409 coupon_expired", got.status, got.errorCode())
	}
}

func TestRedeemNeedsAnOwnerAndNeverTakesTheBusinessFromTheBody(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	code := s.generate(path, "Ana", "5491112345678")
	_, admin := s.user("admin@example.com", "super_admin")

	if got := s.do(http.MethodPost, "/me/business/codes/redeem", "", map[string]any{"code": code}); got.status != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodPost, "/me/business/codes/redeem", admin, map[string]any{"code": code}); got.status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", got.status)
	}
	if got := s.do(http.MethodPost, "/me/business/codes/redeem", "token-owner@example.com", map[string]any{"code": code, "business_id": 1}); got.status != http.StatusBadRequest {
		t.Errorf("business_id in the body = %d, want 400", got.status)
	}
	var status string
	if err := s.pool.QueryRow(t.Context(), `SELECT status FROM benefit_codes WHERE code = $1`, code).Scan(&status); err != nil || status != "pending" {
		t.Errorf("status after the refusals = %q, %v; want pending", status, err)
	}
}

func TestRedeemOverHTTPSucceedsExactlyOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	code := s.generate(path, "Ana", "5491112345678")

	const racers = 20
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		statuses = map[int]int{}
		start    = make(chan struct{})
	)
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got := s.do(http.MethodPost, "/me/business/codes/redeem", "token-owner@example.com", map[string]any{"code": code})
			mu.Lock()
			statuses[got.status]++
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	if statuses[http.StatusOK] != 1 || statuses[http.StatusConflict] != racers-1 {
		t.Errorf("statuses = %v, want exactly one 200 and %d 409", statuses, racers-1)
	}
}
