package benefitshttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// generateWithToken is like generate but also returns the access token the customer got.
func (s *server) generateWithToken(path, name, phone string) (code, token string) {
	s.t.Helper()

	got := s.do(http.MethodPost, path, "", customer(name, phone))
	if got.status != http.StatusCreated {
		s.t.Fatalf("generate = %d %v", got.status, got.body)
	}
	return got.body["code"].(string), got.body["access_token"].(string)
}

func TestCustomerSeesTheCouponOfTheirTokenWithoutAnAccountOrThePhone(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	code, token := s.generateWithToken(path, "Ana", "5491112345678")

	got := s.do(http.MethodGet, "/access/"+token, "", nil)
	if got.status != http.StatusOK || got.body["code"] != code || got.body["status"] != "pending" || got.body["business_name"] != "Studio" {
		t.Fatalf("coupon = %d %v", got.status, got.body)
	}
	benefit, _ := got.body["benefit"].(map[string]any)
	if benefit["type"] != "percent" || benefit["percent"] != float64(15) || got.body["expires_at"] == nil || got.body["redeemed_at"] != nil {
		t.Errorf("coupon body = %v, want the benefit, the expiry and no redemption time", got.body)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got.header.Get("Cache-Control"))
	}
	raw, _ := json.Marshal(got.body)
	for _, leak := range []string{"5491112345678", "phone", "access_token", token} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("body %s leaks %q", raw, leak)
		}
	}

	// After the business redeems it, the customer sees it redeemed.
	if redeem := s.do(http.MethodPost, "/me/business/codes/redeem", "token-owner@example.com", map[string]any{"code": code}); redeem.status != http.StatusOK {
		t.Fatalf("redeem = %d %v", redeem.status, redeem.body)
	}
	if after := s.do(http.MethodGet, "/access/"+token, "", nil); after.body["status"] != "redeemed" || after.body["redeemed_at"] == nil {
		t.Errorf("after redeeming = %v, want redeemed with its time", after.body)
	}
}

func TestAccessTokenOfOneCouponDoesNotRevealTheOthersOfTheSameCustomer(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	mine := s.visibleBenefit("mine@example.com", 5)
	theirs := s.visibleBenefit("theirs@example.com", 5)
	// The phone is not verified: anyone can type someone else's number.
	_, victimToken := s.generateWithToken(theirs, "Victim", "5491112345678")
	attackerCode, attackerToken := s.generateWithToken(mine, "Attacker", "5491112345678")

	got := s.do(http.MethodGet, "/access/"+attackerToken, "", nil)
	if got.status != http.StatusOK || got.body["code"] != attackerCode {
		t.Fatalf("coupon = %d %v", got.status, got.body)
	}
	raw, _ := json.Marshal(got.body)
	if strings.Contains(string(raw), "Victim") || strings.Contains(string(raw), victimToken) {
		t.Errorf("body %s shows the other coupon of the same phone", raw)
	}
}

func TestUnknownOrMalformedAccessTokensAnswerTheSame(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	_, token := s.generateWithToken(s.visibleBenefit("owner@example.com", 5), "Ana", "5491112345678")

	var bodies []string
	for name, bad := range map[string]string{"unknown": token[:len(token)-1] + "x", "short": "abc", "code instead": "ABCDEFGH"} {
		got := s.do(http.MethodGet, "/access/"+bad, "", nil)
		if got.status != http.StatusNotFound || got.errorCode() != "coupon_not_found" {
			t.Errorf("%s = %d %q, want 404 coupon_not_found", name, got.status, got.errorCode())
		}
		raw, _ := json.Marshal(got.body)
		bodies = append(bodies, string(raw))
	}
	for _, b := range bodies[1:] {
		if b != bodies[0] {
			t.Errorf("answers differ: %s vs %s", b, bodies[0])
		}
	}
}
