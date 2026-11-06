package billinghttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	billinghttp "github.com/joaquinBrassinne/MOVO/internal/billing/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/billing/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type tokens map[string]httpx.Principal

func (t tokens) Verify(raw string) (httpx.Principal, error) {
	p, ok := t[raw]
	if !ok {
		return httpx.Principal{}, errors.New("unknown token")
	}
	return p, nil
}

type ownerPlans map[int64]billing.PlanInfo

func (o ownerPlans) PlanOfOwner(_ context.Context, ownerID int64) (billing.PlanInfo, error) {
	info, ok := o[ownerID]
	if !ok {
		return billing.PlanInfo{}, billing.ErrOwnerHasNoBusiness
	}
	return info, nil
}

func newMeServer(t *testing.T) (*httptest.Server, billing.PlanCatalog) {
	t.Helper()

	catalog, err := configcatalog.New(config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800},
	})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	plans := catalog.Plans()
	handlers := billinghttp.New(catalog).WithOwnerPlans(
		tokens{"owner": {UserID: 7, Role: "business_admin"}, "fresh": {UserID: 8, Role: "business_admin"}, "root": {UserID: 1, Role: "super_admin"}},
		ownerPlans{7: plans[1]},
	)

	ts := httptest.NewServer(httpx.NewRouter(nil, nil, handlers.Mount))
	t.Cleanup(ts.Close)
	return ts, catalog
}

func get(t *testing.T, url, token string) (int, map[string]any) {
	t.Helper()

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = res.Body.Close() }()

	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	return res.StatusCode, body
}

func TestMyPlanShowsThePlanOfTheCallersBusinessWithItsLimits(t *testing.T) {
	t.Parallel()

	ts, _ := newMeServer(t)
	status, body := get(t, ts.URL+"/api/v1/me/plan", "owner")

	if status != http.StatusOK || body["plan"] != string(domain.PlanPro) || body["price"] != float64(29800) {
		t.Fatalf("plan = %d %v", status, body)
	}
	if body["max_branches"] != float64(4) || body["max_active_services"] != nil {
		t.Errorf("limits = %v / %v, want 4 branches and null (unlimited) services", body["max_branches"], body["max_active_services"])
	}
}

func TestMyPlanIsOnlyForOwnersWithABusiness(t *testing.T) {
	t.Parallel()

	ts, _ := newMeServer(t)
	for name, tc := range map[string]struct {
		token string
		want  int
	}{
		"anonymous":        {"", http.StatusUnauthorized},
		"forged token":     {"nope", http.StatusUnauthorized},
		"super admin":      {"root", http.StatusForbidden},
		"owner without it": {"fresh", http.StatusNotFound},
	} {
		if status, _ := get(t, ts.URL+"/api/v1/me/plan", tc.token); status != tc.want {
			t.Errorf("%s = %d, want %d", name, status, tc.want)
		}
	}
}
