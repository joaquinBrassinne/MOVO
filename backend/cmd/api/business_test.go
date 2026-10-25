package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"

	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	identitydomain "github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

type apiHarness struct {
	t      *testing.T
	pool   *pgxpool.Pool
	signer *tokens.Signer
	url    string
}

func newAPIHarness(t *testing.T) *apiHarness {
	t.Helper()

	pool := dbtest.New(t)
	signer, err := tokens.NewSigner([]byte(strings.Repeat("s", 32)), time.Hour, nil)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	plans := config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800},
	}

	businessMount, billingMount, err := newBusinessRoutes(nil, pool, signer, plans, storage.NewMemory(), 6, unusedGeocoder{})
	if err != nil {
		t.Fatalf("newBusinessRoutes: %v", err)
	}
	ts := httptest.NewServer(httpx.NewRouter(nil, pool, businessMount, billingMount))
	t.Cleanup(ts.Close)

	return &apiHarness{t: t, pool: pool, signer: signer, url: ts.URL + "/api/v1"}
}

// user inserts a user and returns a real access token for it.
func (h *apiHarness) user(email string, role identitydomain.Role) string {
	h.t.Helper()

	var id int64
	err := h.pool.QueryRow(h.t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', $2) RETURNING id`, email, string(role)).Scan(&id)
	if err != nil {
		h.t.Fatalf("insert user: %v", err)
	}
	token, _, err := h.signer.Issue(id, role, time.Now())
	if err != nil {
		h.t.Fatalf("issue token: %v", err)
	}
	return token
}

func (h *apiHarness) do(method, path, token string, body any) (int, map[string]any) {
	h.t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(h.t.Context(), method, h.url+path, reader)
	if err != nil {
		h.t.Fatalf("request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()

	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func errorCode(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestRoutesAreWiredAndProtectedWithRealTokens(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	owner := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)
	admin := h.user("root@example.com", identitydomain.RoleSuperAdmin)

	if status, body := h.do(http.MethodGet, "/plans", "", nil); status != http.StatusOK || len(body["plans"].([]any)) != 2 {
		t.Errorf("GET /plans = %d %v, want 200 with two plans and no token", status, body)
	}

	checks := []struct {
		name   string
		path   string
		token  string
		status int
		code   string
	}{
		{"owner route without token", "/me/business", "", http.StatusUnauthorized, "unauthorized"},
		{"owner route with a forged token", "/me/business", "nope", http.StatusUnauthorized, "unauthorized"},
		{"owner route with an owner token", "/me/business", owner, http.StatusNotFound, "business_not_found"},
		{"owner route as super admin", "/me/business", admin, http.StatusForbidden, "forbidden"},
		{"admin route as owner", "/admin/businesses", owner, http.StatusForbidden, "forbidden"},
		{"admin route without token", "/admin/businesses", "", http.StatusUnauthorized, "unauthorized"},
	}
	for _, c := range checks {
		status, body := h.do(http.MethodGet, c.path, c.token, nil)
		if status != c.status || errorCode(body) != c.code {
			t.Errorf("%s = %d %q, want %d %q", c.name, status, errorCode(body), c.status, c.code)
		}
	}
	if status, body := h.do(http.MethodGet, "/admin/businesses", admin, nil); status != http.StatusOK || body["total"] != float64(0) {
		t.Errorf("admin list = %d %v, want 200 and an empty list", status, body)
	}
}

// TestPilotBusinessesGetTheProBranchLimit proves the plan wiring end to end: the
// Starter limit is 2 branches, but a business the admin marked as pilot gets 4.
func TestPilotBusinessesGetTheProBranchLimit(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	var categoryID int64
	if err := h.pool.QueryRow(t.Context(), `SELECT id FROM categories WHERE slug = 'yoga'`).Scan(&categoryID); err != nil {
		t.Fatalf("category: %v", err)
	}

	regular := h.user("regular@example.com", identitydomain.RoleBusinessAdmin)
	pilot := h.user("pilot@example.com", identitydomain.RoleBusinessAdmin)
	for name, token := range map[string]string{"Regular": regular, "Pilot": pilot} {
		status, body := h.do(http.MethodPost, "/me/business", token, map[string]any{
			"category_id": categoryID, "name": name + " Studio", "whatsapp": "5491112345678",
		})
		if status != http.StatusCreated {
			t.Fatalf("create %s = %d %v", name, status, body)
		}
	}
	if _, err := h.pool.Exec(t.Context(), `UPDATE businesses SET pilot = true WHERE name = 'Pilot Studio'`); err != nil {
		t.Fatalf("mark pilot: %v", err)
	}

	branch := map[string]any{"name": "Sede", "address": "Calle 1", "city": "Rosario", "latitude": -32.95, "longitude": -60.65}
	addBranches := func(token string, n int) (lastStatus int, lastCode string) {
		for range n {
			status, body := h.do(http.MethodPost, "/me/business/branches", token, branch)
			lastStatus, lastCode = status, errorCode(body)
		}
		return lastStatus, lastCode
	}

	if status, code := addBranches(regular, 3); status != http.StatusConflict || code != "plan_limit_reached" {
		t.Errorf("regular business, third branch = %d %q, want 409 plan_limit_reached (Starter has 2)", status, code)
	}
	if status, _ := addBranches(pilot, 4); status != http.StatusCreated {
		t.Errorf("pilot business, fourth branch = %d, want 201 (Pro has 4)", status)
	}
	if status, code := addBranches(pilot, 1); status != http.StatusConflict || code != "plan_limit_reached" {
		t.Errorf("pilot business, fifth branch = %d %q, want 409", status, code)
	}
}

// unusedGeocoder stands in where the test never geocodes: reaching it is a bug.
type unusedGeocoder struct{}

func (unusedGeocoder) Geocode(context.Context, string) (business.Place, error) {
	panic("the geocoder must not be called in this test")
}

// TestMyPlanFollowsTheBusinessOfTheCaller proves /me/plan end to end with real tokens:
// no business yet, then a Starter business, then the same business marked as pilot.
func TestMyPlanFollowsTheBusinessOfTheCaller(t *testing.T) {
	t.Parallel()

	h := newAPIHarness(t)
	var categoryID int64
	if err := h.pool.QueryRow(t.Context(), `SELECT id FROM categories WHERE slug = 'yoga'`).Scan(&categoryID); err != nil {
		t.Fatalf("category: %v", err)
	}
	token := h.user("owner@example.com", identitydomain.RoleBusinessAdmin)

	if status, body := h.do(http.MethodGet, "/me/plan", token, nil); status != http.StatusNotFound || errorCode(body) != "business_not_found" {
		t.Fatalf("before creating the business = %d %v, want 404 business_not_found", status, body)
	}

	if status, body := h.do(http.MethodPost, "/me/business", token, map[string]any{
		"category_id": categoryID, "name": "Studio", "whatsapp": "5491112345678",
	}); status != http.StatusCreated {
		t.Fatalf("create = %d %v", status, body)
	}

	status, body := h.do(http.MethodGet, "/me/plan", token, nil)
	if status != http.StatusOK || body["plan"] != "starter" || body["max_branches"] != float64(2) || body["max_active_services"] != float64(3) {
		t.Fatalf("regular business plan = %d %v, want Starter with 2 branches and 3 services", status, body)
	}

	if _, err := h.pool.Exec(t.Context(), `UPDATE businesses SET pilot = true`); err != nil {
		t.Fatalf("mark pilot: %v", err)
	}
	status, body = h.do(http.MethodGet, "/me/plan", token, nil)
	if status != http.StatusOK || body["plan"] != "pro" || body["max_branches"] != float64(4) || body["max_active_services"] != nil {
		t.Errorf("pilot business plan = %d %v, want Pro with 4 branches and unlimited services", status, body)
	}
}
