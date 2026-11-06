package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Environment variable understood by LoadCoupons.
const envCouponAttempts = "COUPON_RATE_LIMIT_ATTEMPTS"

// defaultCouponAttempts is how many coupon requests one origin may make per hour: a
// customer needs one coupon, so a handful covers a household or a gym's shared Wi-Fi.
const defaultCouponAttempts = 10

// Coupons holds the settings of the public coupon endpoint.
type Coupons struct {
	// MaxAttempts is the coupon requests per hour one origin may make, successful or not.
	MaxAttempts int
}

// LoadCoupons builds Coupons from get, normally os.Getenv. An unset or empty variable
// keeps the default; anything that is not a whole number above zero is refused at start-up.
func LoadCoupons(get func(string) string) (Coupons, error) {
	if get == nil {
		return Coupons{}, errors.New("config: environment lookup must not be nil")
	}

	raw := strings.TrimSpace(get(envCouponAttempts))
	if raw == "" {
		return Coupons{MaxAttempts: defaultCouponAttempts}, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return Coupons{}, fmt.Errorf("config: %s %q must be a whole number above zero", envCouponAttempts, raw)
	}
	return Coupons{MaxAttempts: n}, nil
}
