package billinghttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	billinghttp "github.com/joaquinBrassinne/MOVO/internal/billing/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

func TestGetPlansListsPricesAndLimitsWithoutAuthentication(t *testing.T) {
	t.Parallel()

	catalog, err := configcatalog.New(config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800, CheckoutURL: "https://mp.example/starter"},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800, CheckoutURL: "https://mp.example/pro"},
	})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	ts := httptest.NewServer(httpx.NewRouter(nil, nil, billinghttp.New(catalog).Mount))
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/api/v1/plans")
	if err != nil {
		t.Fatalf("GET /plans: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	var body struct {
		Plans []struct {
			Plan              string `json:"plan"`
			Price             int    `json:"price"`
			CheckoutURL       string `json:"checkout_url"`
			MaxBranches       *int   `json:"max_branches"`
			MaxActiveServices *int   `json:"max_active_services"`
		} `json:"plans"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if res.StatusCode != http.StatusOK || len(body.Plans) != 2 {
		t.Fatalf("status %d, plans %+v", res.StatusCode, body.Plans)
	}
	starter, pro := body.Plans[0], body.Plans[1]
	if starter.Plan != "starter" || starter.Price != 19800 || *starter.MaxBranches != 2 || *starter.MaxActiveServices != 3 {
		t.Errorf("starter = %+v", starter)
	}
	if pro.Plan != "pro" || pro.Price != 29800 || *pro.MaxBranches != 4 || pro.MaxActiveServices != nil {
		t.Errorf("pro = %+v, want max_active_services null for unlimited", pro)
	}
	if starter.CheckoutURL != "https://mp.example/starter" || pro.CheckoutURL != "https://mp.example/pro" {
		t.Errorf("checkout urls = %q, %q", starter.CheckoutURL, pro.CheckoutURL)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("Cache-Control = %q", cc)
	}
}
