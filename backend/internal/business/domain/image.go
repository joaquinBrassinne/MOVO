package domain

import (
	"errors"
	"strconv"
	"strings"
)

// ErrImageKeyInvalid means the object key does not belong to the business or is
// not an accepted image name.
var ErrImageKeyInvalid = errors.New("business: image key is not valid")

const maxImageNameLen = 100

// Image is a photo of a business. Position 0 is the cover.
type Image struct {
	ID         int64
	BusinessID int64
	ObjectKey  string
	Position   int
}

// ValidateImageKey checks that key is a direct child of the business folder and
// has an accepted image extension. It is the key shape PresignImageUpload builds,
// so an owner can only register objects the API gave them room to upload.
func ValidateImageKey(businessID int64, key string) error {
	prefix := "businesses/" + strconv.FormatInt(businessID, 10) + "/"
	name, ok := strings.CutPrefix(key, prefix)
	if !ok || name == "" || len(name) > maxImageNameLen {
		return ErrImageKeyInvalid
	}

	dot := strings.LastIndexByte(name, '.')
	if dot <= 0 {
		return ErrImageKeyInvalid
	}
	switch name[dot+1:] {
	case "jpg", "png", "webp":
	default:
		return ErrImageKeyInvalid
	}

	for _, r := range name[:dot] {
		if !isKeyChar(r) {
			return ErrImageKeyInvalid
		}
	}
	return nil
}

// isKeyChar accepts the characters of the names PresignImageUpload generates:
// hex ids, plus letters, digits, "-" and "_" for anything a client may choose.
func isKeyChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return r == '-' || r == '_'
	}
}
