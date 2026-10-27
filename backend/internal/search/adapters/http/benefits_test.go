package searchhttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProfileListsOnlyTheBenefitsOfferedRightNowBestFirstWithoutLimits(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	branch := addBranch(t, pool, 1, 1_000)
	var business int64
	if err := pool.QueryRow(t.Context(), `SELECT business_id FROM branches WHERE id = $1`, branch).Scan(&business); err != nil {
		t.Fatalf("find business: %v", err)
	}
	exec(t, pool, `UPDATE businesses SET monthly_coupon_limit = 7 WHERE id = $1`, business)
	exec(t, pool, `
INSERT INTO benefits (business_id, type, percent, amount_ars, title, conditions, active, admin_paused) VALUES
  ($1, 'custom',  NULL, NULL, 'Una clase gratis', 'Primera visita', true,  false),
  ($1, 'amount',  NULL, 2000, NULL,               NULL,             true,  false),
  ($1, 'percent', 15,   NULL, NULL,               'Solo lunes',     true,  false),
  ($1, 'percent', 50,   NULL, NULL,               NULL,             true,  true),
  ($1, 'percent', 30,   NULL, NULL,               NULL,             false, false)`, business)

	res := getProfile(h, "studio-1")
	if res.Code != http.StatusOK {
		t.Fatalf("profile = %d %s", res.Code, res.Body)
	}
	var body struct {
		Benefits []map[string]any `json:"benefits"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	labels := []any{}
	for _, b := range body.Benefits {
		labels = append(labels, b["short_label"])
	}
	if len(labels) != 3 || labels[0] != "-15%" || labels[1] != "-$2.000" || labels[2] != "Beneficio" {
		t.Fatalf("labels = %v, want the three offered benefits, best first, without the paused and the switched-off ones", labels)
	}
	first := body.Benefits[0]
	if first["type"] != "percent" || first["percent"] != float64(15) || first["conditions"] != "Solo lunes" || first["title"] != nil || first["id"] == nil {
		t.Errorf("percent benefit = %v", first)
	}
	if last := body.Benefits[2]; last["title"] != "Una clase gratis" || last["percent"] != nil || last["amount_ars"] != nil {
		t.Errorf("custom benefit = %v", last)
	}
	for _, leak := range []string{"monthly", "limit", "quota", "admin_paused", "active"} {
		if strings.Contains(res.Body.String(), `"`+leak) {
			t.Errorf("the profile leaks %q: %s", leak, res.Body)
		}
	}
}

func TestProfileOfABusinessWithoutBenefitsHasAnEmptyList(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	addBranch(t, pool, 1, 1_000)

	var body struct {
		Benefits []any `json:"benefits"`
	}
	res := getProfile(h, "studio-1")
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body.Benefits == nil || len(body.Benefits) != 0 {
		t.Errorf("benefits = %v, %v; want an empty array, not null", body.Benefits, err)
	}
}

func TestSearchCardsHaveABenefitLabelOrNull(t *testing.T) {
	t.Parallel()

	h, pool := newServer(t)
	with := addBranch(t, pool, 1, 1_000)
	addBranch(t, pool, 2, 2_000)
	exec(t, pool, `INSERT INTO benefits (business_id, type, percent)
		SELECT business_id, 'percent', 15 FROM branches WHERE id = $1`, with)

	res := get(h, "lat=-32.17&lng=-64.11&radius_m=20000")
	var body struct {
		Results []struct {
			BranchID     int64   `json:"branch_id"`
			BenefitLabel *string `json:"benefit_label"`
		} `json:"results"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || len(body.Results) != 2 {
		t.Fatalf("search = %d %s", res.Code, res.Body)
	}
	for _, card := range body.Results {
		switch {
		case card.BranchID == with && (card.BenefitLabel == nil || *card.BenefitLabel != "-15%"):
			t.Errorf("card with a benefit = %v, want -15%%", card.BenefitLabel)
		case card.BranchID != with && card.BenefitLabel != nil:
			t.Errorf("card without a benefit = %q, want null", *card.BenefitLabel)
		}
	}
	if !strings.Contains(res.Body.String(), `"benefit_label":null`) {
		t.Errorf("the field must be present as null when there is no benefit: %s", res.Body)
	}
}
