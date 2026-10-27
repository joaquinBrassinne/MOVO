package benefitshttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestOwnerPreviewsACouponWithoutRedeemingItOrSeeingThePhone(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	owner := "token-owner@example.com"
	code := s.generate(path, "Ana", "5491112345678")

	got := s.do(http.MethodGet, "/me/business/codes/"+strings.ToLower(code), owner, nil)
	if got.status != http.StatusOK || got.body["code"] != code || got.body["status"] != "pending" || got.body["customer_name"] != "Ana" {
		t.Fatalf("preview = %d %v", got.status, got.body)
	}
	benefit, _ := got.body["benefit"].(map[string]any)
	if benefit["type"] != "percent" || benefit["percent"] != float64(15) || got.body["expires_at"] == nil || got.body["redeemed_at"] != nil {
		t.Errorf("preview body = %v, want the benefit, the expiry and no redemption time", got.body)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got.header.Get("Cache-Control"))
	}

	// The whole answer, not only a field called "phone", must be free of the number.
	raw, _ := json.Marshal(got.body)
	for _, leak := range []string{"5491112345678", "phone", "access_token", "token"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("preview body %s leaks %q", raw, leak)
		}
	}

	// Looking, even repeatedly, redeems nothing: the redemption still works, once.
	s.do(http.MethodGet, "/me/business/codes/"+code, owner, nil)
	var status string
	if err := s.pool.QueryRow(t.Context(), `SELECT status FROM benefit_codes WHERE code = $1`, code).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("status after previews = %q, %v; want pending", status, err)
	}
	if redeem := s.do(http.MethodPost, "/me/business/codes/redeem", owner, map[string]any{"code": code}); redeem.status != http.StatusOK {
		t.Fatalf("redeem after preview = %d %v", redeem.status, redeem.body)
	}
	after := s.do(http.MethodGet, "/me/business/codes/"+code, owner, nil)
	if after.body["status"] != "redeemed" || after.body["redeemed_at"] == nil {
		t.Errorf("preview after redeeming = %v, want redeemed with its time", after.body)
	}
}

func TestPreviewShowsAnOverdueCouponAsExpiredWithoutMarkingIt(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	code := s.generate(path, "Ana", "5491112345678")
	if _, err := s.pool.Exec(t.Context(), `UPDATE benefit_codes SET expires_at = now() - interval '1 minute' WHERE code = $1`, code); err != nil {
		t.Fatalf("age coupon: %v", err)
	}

	got := s.do(http.MethodGet, "/me/business/codes/"+code, "token-owner@example.com", nil)
	if got.status != http.StatusOK || got.body["status"] != "expired" {
		t.Fatalf("preview = %d %v, want expired", got.status, got.body)
	}
	var stored string
	if err := s.pool.QueryRow(t.Context(), `SELECT status FROM benefit_codes WHERE code = $1`, code).Scan(&stored); err != nil || stored != "pending" {
		t.Errorf("stored status = %q, %v; want it left pending: a read has no side effects", stored, err)
	}
}

func TestPreviewDoesNotRevealTheCouponsOfOtherBusinesses(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	s.visibleBenefit("mine@example.com", 5)
	theirs := s.visibleBenefit("other@example.com", 5)
	foreign := s.generate(theirs, "Beto", "5491187654321")
	mine := "token-mine@example.com"

	var bodies []string
	for name, code := range map[string]string{"foreign": foreign, "unknown": "JKLMNPQR", "malformed": "nope", "redeem word": "redeem"} {
		got := s.do(http.MethodGet, "/me/business/codes/"+code, mine, nil)
		if got.status != http.StatusNotFound || got.errorCode() != "coupon_not_found" {
			t.Errorf("%s = %d %q, want 404 coupon_not_found", name, got.status, got.errorCode())
		}
		raw, _ := json.Marshal(got.body)
		bodies = append(bodies, string(raw))
	}
	// A foreign code is indistinguishable from an unknown one.
	for _, b := range bodies[1:] {
		if b != bodies[0] {
			t.Errorf("answers differ: %s vs %s", b, bodies[0])
		}
	}
}

func TestPreviewNeedsAnOwner(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	code := s.generate(path, "Ana", "5491112345678")
	_, admin := s.user("admin@example.com", "super_admin")

	if got := s.do(http.MethodGet, "/me/business/codes/"+code, "", nil); got.status != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", got.status)
	}
	if got := s.do(http.MethodGet, "/me/business/codes/"+code, admin, nil); got.status != http.StatusForbidden {
		t.Errorf("super admin = %d, want 403", got.status)
	}
}
