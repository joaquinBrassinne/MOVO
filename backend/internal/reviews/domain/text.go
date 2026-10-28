package domain

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TextReason says why a free text was refused.
type TextReason string

// Reasons a free text is refused.
const (
	TextInvalid  TextReason = "invalid"        // too long or with control characters
	TextHasLink  TextReason = "contains_link"  // carries a URL or a web address
	TextHasPhone TextReason = "contains_phone" // carries a phone number
)

// TextError is a refused free text (the comment of a review).
type TextError struct {
	Field  string
	Reason TextReason
}

func (e *TextError) Error() string {
	return "reviews: " + e.Field + " is not valid: " + string(e.Reason)
}

// A review is public and carries no way out of MOVO: no URL and no phone number. The
// benefits module applies the same rule to its texts; modules do not share internals,
// so this is its own copy.
var (
	linkPattern = regexp.MustCompile(`(?i)(?:https?:|ftp:|//|www\.|\b[\p{L}\d-]+\.(?:com|net|org|info|biz|edu|gov|gob|io|ly|me|co|app|ar|us|es|cl|uy|br|xyz|site|online|shop|store|link)\b)`)
	// Eight digits or more, allowing one separator between them: the shortest local
	// number. A price such as "$10.000" has five and passes.
	phonePattern = regexp.MustCompile(`\d(?:[\s().-]?\d){7,}`)
)

// plainText trims raw and checks it against the limit of max characters. An empty text
// is valid and comes back empty.
func plainText(field, raw string, max int) (string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", nil
	}
	if utf8.RuneCountInString(text) > max || strings.IndexFunc(text, unicode.IsControl) >= 0 {
		return "", &TextError{Field: field, Reason: TextInvalid}
	}
	if linkPattern.MatchString(text) {
		return "", &TextError{Field: field, Reason: TextHasLink}
	}
	if phonePattern.MatchString(text) {
		return "", &TextError{Field: field, Reason: TextHasPhone}
	}
	return text, nil
}
