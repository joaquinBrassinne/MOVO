package domain

import (
	"errors"
	"strconv"
	"strings"
)

// ErrSlugInvalid means a name has no letters or digits to build a URL from.
var ErrSlugInvalid = errors.New("business: slug cannot be built from the name")

var accents = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a", "ã", "a",
	"é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i",
	"ó", "o", "ò", "o", "ö", "o", "ô", "o", "õ", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u",
	"ñ", "n", "ç", "c",
)

// Slugify builds the URL form of a name: lowercase ASCII letters and digits
// joined by single hyphens, matching the database check on businesses.slug.
func Slugify(name string) (string, error) {
	var slug strings.Builder
	pendingHyphen := false
	for _, r := range accents.Replace(strings.ToLower(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingHyphen && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteRune(r)
			pendingHyphen = false
			continue
		}
		pendingHyphen = true
	}

	if slug.Len() == 0 {
		return "", ErrSlugInvalid
	}
	return slug.String(), nil
}

// SlugWithSuffix returns the candidate for the given attempt when the base slug
// is already taken: the base itself first, then base-2, base-3 and so on.
func SlugWithSuffix(base string, attempt int) string {
	if attempt <= 1 {
		return base
	}
	return base + "-" + strconv.Itoa(attempt)
}
