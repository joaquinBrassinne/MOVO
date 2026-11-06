package config_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

func TestLoadCouponsKeepsTenAttemptsByDefault(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "  "} {
		got, err := config.LoadCoupons(func(string) string { return raw })
		if err != nil || got.MaxAttempts != 10 {
			t.Errorf("LoadCoupons(%q) = %+v, %v; want 10 attempts", raw, got, err)
		}
	}
}

func TestLoadCouponsReadsTheLimitFromTheEnvironment(t *testing.T) {
	t.Parallel()

	got, err := config.LoadCoupons(func(key string) string {
		if key == "COUPON_RATE_LIMIT_ATTEMPTS" {
			return " 250 "
		}
		return ""
	})
	if err != nil || got.MaxAttempts != 250 {
		t.Errorf("LoadCoupons = %+v, %v; want 250 attempts", got, err)
	}
}

func TestLoadCouponsRefusesWhatIsNotAPositiveWholeNumber(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"0", "-5", "ten", "1.5"} {
		if _, err := config.LoadCoupons(func(string) string { return raw }); err == nil {
			t.Errorf("LoadCoupons(%q) must fail", raw)
		}
	}
	if _, err := config.LoadCoupons(nil); err == nil {
		t.Error("a nil lookup must fail")
	}
}
