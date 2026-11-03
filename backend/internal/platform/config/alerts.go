package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Environment variables understood by LoadAlerts (SDD §7.5).
const (
	envAlertMinCodes      = "ALERT_MIN_CODES"
	envAlertMinRedeemRate = "ALERT_MIN_REDEEM_RATE"
)

// Defaults of the "coupons not redeemed" alert: below one redeemed coupon in five, once at
// least 20 were generated, the owner is told.
const (
	defaultAlertMinCodes      = 20
	defaultAlertMinRedeemRate = 0.2
)

// Alerts holds the thresholds of the "coupons not redeemed" alert (SDD 4.3, metric 15).
type Alerts struct {
	// MinCodes is how many coupons a business must have generated in the last 30 days before
	// the alert may fire: below it a rate says nothing.
	MinCodes int
	// MinRedeemRate is the redemption rate under which the alert fires, in (0, 1].
	MinRedeemRate float64
}

// LoadAlerts builds Alerts from get, normally os.Getenv. An unset or empty variable keeps the
// default; anything that is not a valid value is refused at start-up.
func LoadAlerts(get func(string) string) (Alerts, error) {
	if get == nil {
		return Alerts{}, errors.New("config: environment lookup must not be nil")
	}

	out := Alerts{MinCodes: defaultAlertMinCodes, MinRedeemRate: defaultAlertMinRedeemRate}
	if raw := strings.TrimSpace(get(envAlertMinCodes)); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return Alerts{}, fmt.Errorf("config: %s %q must be a whole number above zero", envAlertMinCodes, raw)
		}
		out.MinCodes = n
	}
	if raw := strings.TrimSpace(get(envAlertMinRedeemRate)); raw != "" {
		rate, err := strconv.ParseFloat(raw, 64)
		if err != nil || rate <= 0 || rate > 1 {
			return Alerts{}, fmt.Errorf("config: %s %q must be a number above 0 and up to 1", envAlertMinRedeemRate, raw)
		}
		out.MinRedeemRate = rate
	}
	return out, nil
}
