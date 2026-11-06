package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Validation errors for the contact a customer declares.
var (
	ErrCustomerNameInvalid = errors.New("benefits: customer name is not valid")
	ErrPhoneInvalid        = errors.New("benefits: whatsapp number is not valid")
)

const (
	maxCustomerNameLen = 100
	minPhoneDigits     = 8
	maxPhoneDigits     = 15
)

// Contact is what a customer declares to generate a coupon. There is no account and
// nothing is verified: it is only the name the business will read and the number it
// can write to.
type Contact struct {
	Name  string
	Phone string // digits only, with the country code
}

// NewContact validates what a person typed and returns the stored form.
func NewContact(name, phone string) (Contact, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxCustomerNameLen {
		return Contact{}, ErrCustomerNameInvalid
	}
	number, err := NormalizePhone(phone)
	if err != nil {
		return Contact{}, err
	}
	return Contact{Name: name, Phone: number}, nil
}

// NormalizePhone turns a number typed by a person into the stored form: digits only,
// with the country code and without "+" or the "00" international prefix. The same
// person typing it two ways must give the same value, because that is how the rule
// "one pending coupon per customer and business" recognizes a repeat. A number without
// a country code is rejected rather than guessed. The business module keeps its own
// copy of this rule for its WhatsApp number: modules do not share internals.
func NormalizePhone(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case strings.ContainsRune(" -().+", r):
		default:
			return "", ErrPhoneInvalid
		}
	}

	number := strings.TrimPrefix(digits.String(), "00")
	if len(number) < minPhoneDigits || len(number) > maxPhoneDigits || number[0] == '0' {
		return "", ErrPhoneInvalid
	}
	return number, nil
}
