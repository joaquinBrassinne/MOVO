package domain

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Errors for what the owner declares about the Mercado Pago payment.
var (
	ErrPayerEmailInvalid = errors.New("business: payer email is not valid")
	ErrPayerNameInvalid  = errors.New("business: payer name is not valid")
)

const (
	maxPayerEmailLen = 254
	minPayerNameLen  = 3
	maxPayerNameLen  = 100

	maxReturnParams        = 10
	maxReturnParamKeyLen   = 40
	maxReturnParamValueLen = 200
)

// PlanHintParam is the return URL parameter read as a hint of the plan paid.
const PlanHintParam = "plan"

// DeclarePayer stores the Mercado Pago email and the full name the owner paid with.
// The super admin crosses them with the "new subscriber" mail from Mercado Pago, so
// both are required; they are a hint, never proof of payment.
func (b Business) DeclarePayer(email, name string) (Business, error) {
	email = strings.TrimSpace(email)
	if !validEmail(email) {
		return Business{}, ErrPayerEmailInvalid
	}
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < minPayerNameLen || n > maxPayerNameLen {
		return Business{}, ErrPayerNameInvalid
	}
	b.MPPayerEmail, b.MPPayerName = email, name
	return b, nil
}

// validEmail accepts a bare address only: "Name <a@b.c>" is parsed by net/mail but is
// not what the owner is asked for.
func validEmail(s string) bool {
	if s == "" || len(s) > maxPayerEmailLen {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	domain := s[strings.LastIndexByte(s, '@')+1:]
	return strings.Contains(domain, ".")
}

// RecordReturnParams keeps the query parameters Mercado Pago added to the return URL,
// only as an aid for the super admin. Malformed or oversized entries are dropped
// instead of failing the submission: they are never trusted and never required.
func (b Business) RecordReturnParams(params map[string]string) Business {
	kept := make(map[string]string, len(params))
	for key, value := range params {
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" || value == "" ||
			utf8.RuneCountInString(key) > maxReturnParamKeyLen ||
			utf8.RuneCountInString(value) > maxReturnParamValueLen ||
			len(kept) >= maxReturnParams {
			continue
		}
		kept[key] = value
	}
	if len(kept) == 0 {
		kept = nil
	}
	b.MPReturnParams = kept
	return b
}

// PlanHint is the plan the return URL suggested, or "". It never activates anything:
// the super admin sets the plan when confirming the payment.
func (b Business) PlanHint() string {
	return b.MPReturnParams[PlanHintParam]
}
