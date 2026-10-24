// Package domain holds the business rules: how a business is named, contacted
// and moved through its lifecycle. It imports no infrastructure.
package domain

import (
	"errors"
	"strings"
)

// ErrWhatsAppInvalid means the number cannot be used to build a wa.me link.
var ErrWhatsAppInvalid = errors.New("business: whatsapp number is not valid")

const (
	minWhatsAppDigits = 8
	maxWhatsAppDigits = 15
)

// NormalizeWhatsApp turns a number typed by a person into the stored form:
// digits only, with the country code and without "+" or the "00" international
// prefix (the format wa.me expects). A number without a country code is
// rejected rather than guessed, because a wrong guess sends clients to a
// stranger.
func NormalizeWhatsApp(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case strings.ContainsRune(" -().+", r):
		default:
			return "", ErrWhatsAppInvalid
		}
	}

	number := strings.TrimPrefix(digits.String(), "00")
	if len(number) < minWhatsAppDigits || len(number) > maxWhatsAppDigits || number[0] == '0' {
		return "", ErrWhatsAppInvalid
	}
	return number, nil
}
