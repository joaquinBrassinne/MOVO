package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// ErrPaymentReferenceInvalid means the operation number is longer than any real one.
var ErrPaymentReferenceInvalid = errors.New("business: payment reference is not valid")

const maxPaymentReferenceLen = 100

// SetPaymentReference stores the operation number the owner declares when asking
// for the review. It is optional: a blank value clears it. The super admin is the
// one who confirms the payment, so this is only a hint for them.
func (b Business) SetPaymentReference(ref string) (Business, error) {
	ref = strings.TrimSpace(ref)
	if utf8.RuneCountInString(ref) > maxPaymentReferenceLen {
		return Business{}, ErrPaymentReferenceInvalid
	}
	b.PaymentReference = ref
	return b, nil
}
