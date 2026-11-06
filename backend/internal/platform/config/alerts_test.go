package config_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func alertEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadAlertsKeepsTheSDDDefaults(t *testing.T) {
	t.Parallel()

	got, err := config.LoadAlerts(alertEnv(nil))
	if err != nil || got.MinCodes != 20 || got.MinRedeemRate != 0.2 {
		t.Errorf("LoadAlerts() = %+v, %v; want 20 coupons and a 0.2 rate", got, err)
	}
}

func TestLoadAlertsReadsTheEnvironment(t *testing.T) {
	t.Parallel()

	got, err := config.LoadAlerts(alertEnv(map[string]string{"ALERT_MIN_CODES": " 30 ", "ALERT_MIN_REDEEM_RATE": "0.35"}))
	if err != nil || got.MinCodes != 30 || got.MinRedeemRate != 0.35 {
		t.Errorf("LoadAlerts = %+v, %v; want 30 and 0.35", got, err)
	}
}

func TestLoadAlertsRefusesWhatMakesNoSense(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]map[string]string{
		"zero coupons":     {"ALERT_MIN_CODES": "0"},
		"negative coupons": {"ALERT_MIN_CODES": "-1"},
		"word":             {"ALERT_MIN_CODES": "many"},
		"rate of zero":     {"ALERT_MIN_REDEEM_RATE": "0"},
		"rate above one":   {"ALERT_MIN_REDEEM_RATE": "1.5"},
		"rate word":        {"ALERT_MIN_REDEEM_RATE": "low"},
	} {
		if _, err := config.LoadAlerts(alertEnv(values)); err == nil {
			t.Errorf("%s: LoadAlerts must fail", name)
		}
	}
	if _, err := config.LoadAlerts(nil); err == nil {
		t.Error("a nil lookup must fail")
	}
}
