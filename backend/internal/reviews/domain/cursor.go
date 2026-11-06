package domain

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// ErrCursorInvalid means the cursor is not one this module issued.
var ErrCursorInvalid = errors.New("reviews: the cursor is not valid")

// maxCursorLen bounds what is decoded: a real cursor is about 25 characters.
const maxCursorLen = 64

// Cursor is the position after the last review of a page. Reviews are listed newest
// first, ordered by (CreatedAt, ID) so that two reviews created at the same instant keep
// a stable order and no review is repeated or skipped between pages.
type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

// Encode returns the opaque, URL-safe text the client sends back to get the next page.
func (c Cursor) Encode() string {
	raw := strconv.FormatInt(c.CreatedAt.UnixMicro(), 10) + "." + strconv.FormatInt(c.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// ParseCursor reads what Encode wrote and refuses anything else. The time has the
// microsecond precision the database keeps.
func ParseCursor(s string) (Cursor, error) {
	if s == "" || len(s) > maxCursorLen {
		return Cursor{}, ErrCursorInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrCursorInvalid
	}
	micros, id, ok := strings.Cut(string(raw), ".")
	if !ok {
		return Cursor{}, ErrCursorInvalid
	}
	at, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return Cursor{}, ErrCursorInvalid
	}
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n < 1 {
		return Cursor{}, ErrCursorInvalid
	}
	return Cursor{CreatedAt: time.UnixMicro(at).UTC(), ID: n}, nil
}
