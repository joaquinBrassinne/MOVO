package benefitshttp_test

import (
	"bytes"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// visibleBenefit creates an active business with an active branch, so it is visible,
// and an active percent benefit; it returns the benefit's path.
func (s *server) visibleBenefit(email string, limit int) string {
	s.t.Helper()

	s.owner(email, "active")
	var business, benefit int64
	if err := s.pool.QueryRow(s.t.Context(),
		`SELECT b.id FROM businesses b JOIN users u ON u.id = b.owner_user_id WHERE u.email = $1`, email).Scan(&business); err != nil {
		s.t.Fatalf("find business: %v", err)
	}
	if _, err := s.pool.Exec(s.t.Context(), `
INSERT INTO branches (business_id, name, address, city, location)
VALUES ($1, 'Centro', 'Calle 1', 'Cordoba', ST_SetSRID(ST_MakePoint(-64.18, -31.42), 4326)::geography)`, business); err != nil {
		s.t.Fatalf("insert branch: %v", err)
	}
	if err := s.pool.QueryRow(s.t.Context(),
		`INSERT INTO benefits (business_id, type, percent) VALUES ($1, 'percent', 15) RETURNING id`, business).Scan(&benefit); err != nil {
		s.t.Fatalf("insert benefit: %v", err)
	}
	if _, err := s.pool.Exec(s.t.Context(), `UPDATE businesses SET monthly_coupon_limit = $2 WHERE id = $1`, business, limit); err != nil {
		s.t.Fatalf("set coupon quota: %v", err)
	}
	return "/benefits/" + strconv.FormatInt(benefit, 10) + "/codes"
}

func customer(name, phone string) map[string]any { return map[string]any{"name": name, "phone": phone} }

func TestCustomerGeneratesACouponWithoutAnAccount(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)

	before := time.Now()
	got := s.do(http.MethodPost, path, "", customer(" Ana ", "+54 9 11 1234-5678"))
	if got.status != http.StatusCreated {
		t.Fatalf("generate = %d %v", got.status, got.body)
	}
	code, _ := got.body["code"].(string)
	token, _ := got.body["access_token"].(string)
	if !regexp.MustCompile(`^[A-HJ-NP-Z2-9]{8}$`).MatchString(code) {
		t.Errorf("code = %q, want 8 symbols without 0, O, 1, I", code)
	}
	if len(token) != 43 {
		t.Errorf("access_token = %q, want 43 URL-safe symbols", token)
	}
	expires, err := time.Parse(time.RFC3339, got.body["expires_at"].(string))
	if err != nil || expires.Before(before.Add(47*time.Hour)) || expires.After(time.Now().Add(49*time.Hour)) {
		t.Errorf("expires_at = %v, %v; want about 48 hours from now", got.body["expires_at"], err)
	}
	benefit, _ := got.body["benefit"].(map[string]any)
	if benefit["type"] != "percent" || benefit["percent"] != float64(15) || benefit["title"] != nil {
		t.Errorf("benefit = %v, want the 15%% it claims", benefit)
	}
	if got.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store: the answer carries the access token", got.header.Get("Cache-Control"))
	}

	// What is stored: the contact normalized, and only the hash of the token.
	var phone, name string
	var hash []byte
	if err := s.pool.QueryRow(t.Context(), `
SELECT cu.phone, cu.name, c.access_token_hash FROM benefit_codes c JOIN customers cu ON cu.id = c.customer_id WHERE c.code = $1`, code).Scan(&phone, &name, &hash); err != nil {
		t.Fatalf("read coupon: %v", err)
	}
	if phone != "5491112345678" || name != "Ana" || !bytes.Equal(hash, domain.HashAccessToken(token)) {
		t.Errorf("stored = %q %q, hash match %v; want the normalized phone, the trimmed name and the token's hash", phone, name, bytes.Equal(hash, domain.HashAccessToken(token)))
	}
	if bytes.Contains(hash, []byte(token)) {
		t.Error("the plain token must not be stored")
	}
}

func TestSecondCouponAtTheSameBusinessIsAConflict(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)

	if first := s.do(http.MethodPost, path, "", customer("Ana", "5491112345678")); first.status != http.StatusCreated {
		t.Fatalf("first = %d %v", first.status, first.body)
	}
	// The same person typing the number another way is still the same customer.
	again := s.do(http.MethodPost, path, "", customer("Ana", "+54 (9) 11 1234 5678"))
	if again.status != http.StatusConflict || again.errorCode() != "pending_coupon" {
		t.Errorf("repeat = %d %q, want 409 pending_coupon", again.status, again.errorCode())
	}
	if other := s.do(http.MethodPost, path, "", customer("Beto", "5491187654321")); other.status != http.StatusCreated {
		t.Errorf("another customer = %d %v, want 201", other.status, other.body)
	}
}

func TestCouponsStopAtTheMonthlyLimitOfTheBusiness(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 1)

	if first := s.do(http.MethodPost, path, "", customer("Ana", "5491112345678")); first.status != http.StatusCreated {
		t.Fatalf("first = %d %v", first.status, first.body)
	}
	got := s.do(http.MethodPost, path, "", customer("Beto", "5491187654321"))
	if got.status != http.StatusConflict || got.errorCode() != "quota_reached" {
		t.Errorf("over the limit = %d %q, want 409 quota_reached", got.status, got.errorCode())
	}
}

func TestGenerateRefusesInvalidInputAndUnavailableBenefits(t *testing.T) {
	t.Parallel()

	s := newServer(t)
	path := s.visibleBenefit("owner@example.com", 5)
	hidden := s.owner("draft@example.com", "draft")
	_ = hidden
	var draftBenefit int64
	if err := s.pool.QueryRow(t.Context(), `
INSERT INTO benefits (business_id, type, percent)
SELECT b.id, 'percent', 10 FROM businesses b JOIN users u ON u.id = b.owner_user_id WHERE u.email = 'draft@example.com' RETURNING id`).Scan(&draftBenefit); err != nil {
		t.Fatalf("insert draft benefit: %v", err)
	}

	for _, tc := range []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"blank name", customer(" ", "5491112345678"), "name"},
		{"no country code", customer("Ana", "011 1234 5678"), "phone"},
		{"letters in the phone", customer("Ana", "abc"), "phone"},
	} {
		got := s.do(http.MethodPost, path, "", tc.body)
		if got.status != http.StatusUnprocessableEntity || got.errorCode() != "validation_failed" || got.errorField() != tc.field {
			t.Errorf("%s = %d %q field %q, want 422 validation_failed on %q", tc.name, got.status, got.errorCode(), got.errorField(), tc.field)
		}
	}
	if got := s.do(http.MethodPost, path, "", map[string]any{"name": "Ana", "phone": "5491112345678", "business_id": 1}); got.status != http.StatusBadRequest {
		t.Errorf("unknown field = %d, want 400", got.status)
	}

	for name, p := range map[string]string{
		"unknown benefit":         "/benefits/999999/codes",
		"malformed id":            "/benefits/abc/codes",
		"benefit of a draft shop": "/benefits/" + strconv.FormatInt(draftBenefit, 10) + "/codes",
	} {
		if got := s.do(http.MethodPost, p, "", customer("Ana", "5491112345678")); got.status != http.StatusNotFound || got.errorCode() != "benefit_not_found" {
			t.Errorf("%s = %d %q, want 404 benefit_not_found", name, got.status, got.errorCode())
		}
	}

	if _, err := s.pool.Exec(t.Context(), `UPDATE benefits SET active = false WHERE id = ($1::text)::bigint`, path[len("/benefits/"):len(path)-len("/codes")]); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if got := s.do(http.MethodPost, path, "", customer("Ana", "5491112345678")); got.status != http.StatusNotFound {
		t.Errorf("inactive benefit = %d, want 404", got.status)
	}

	var customers int
	if err := s.pool.QueryRow(t.Context(), `SELECT count(*) FROM customers`).Scan(&customers); err != nil || customers != 0 {
		t.Errorf("customers after refused requests = %d, %v; want none filed", customers, err)
	}
}

func forwarded(ip string) http.Header { return http.Header{"X-Forwarded-For": {ip}} }

func TestClientsBehindATrustedProxyHaveTheirOwnBudget(t *testing.T) {
	t.Parallel()

	s := newServerBehind(t, 1, []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")})
	path := s.visibleBenefit("owner@example.com", 50)

	first := s.doWith(forwarded("198.51.100.1"), http.MethodPost, path, "", customer("Ana", "5491112345678"))
	again := s.doWith(forwarded("198.51.100.1"), http.MethodPost, path, "", customer("Ana", "5491112345678"))
	other := s.doWith(forwarded("198.51.100.2"), http.MethodPost, path, "", customer("Beto", "5491187654321"))

	if first.status != http.StatusCreated || other.status != http.StatusCreated {
		t.Errorf("each client's first request = %d and %d, want 201 for both", first.status, other.status)
	}
	if again.status != http.StatusTooManyRequests {
		t.Errorf("the first client's second request = %d, want 429", again.status)
	}
}

func TestAForgedForwardedForDoesNotBuyANewBudget(t *testing.T) {
	t.Parallel()

	// Nobody is trusted: the header is a client claim, so changing it changes nothing.
	s := newServerBehind(t, 1, nil)
	path := s.visibleBenefit("owner@example.com", 50)

	first := s.doWith(forwarded("198.51.100.1"), http.MethodPost, path, "", customer("Ana", "5491112345678"))
	forged := s.doWith(forwarded("198.51.100.2"), http.MethodPost, path, "", customer("Beto", "5491187654321"))

	if first.status != http.StatusCreated {
		t.Fatalf("first request = %d, want 201", first.status)
	}
	if forged.status != http.StatusTooManyRequests {
		t.Errorf("forged address = %d, want 429: the budget belongs to the connection", forged.status)
	}
}

func TestAnOriginThatAsksTooOftenIsToldWhenToRetry(t *testing.T) {
	t.Parallel()

	s := newServerWithLimit(t, 3)
	path := s.visibleBenefit("owner@example.com", 50)

	// Every attempt counts, including the ones that fail validation.
	s.do(http.MethodPost, path, "", customer(" ", "5491112345678"))
	s.do(http.MethodPost, path, "", customer("Ana", "5491112345678"))
	s.do(http.MethodPost, path, "", customer("Beto", "5491187654321"))

	got := s.do(http.MethodPost, path, "", customer("Carla", "5491100000003"))
	if got.status != http.StatusTooManyRequests || got.errorCode() != "rate_limited" {
		t.Fatalf("over the budget = %d %q, want 429 rate_limited", got.status, got.errorCode())
	}
	seconds, err := strconv.Atoi(got.header.Get("Retry-After"))
	if err != nil || seconds < 3500 || seconds > 3600 {
		t.Errorf("Retry-After = %q, want about an hour in seconds", got.header.Get("Retry-After"))
	}
	var customers int
	if err := s.pool.QueryRow(t.Context(), `SELECT count(*) FROM customers WHERE phone = '5491100000003'`).Scan(&customers); err != nil || customers != 0 {
		t.Errorf("a refused origin must not file a customer: %d, %v", customers, err)
	}
}
