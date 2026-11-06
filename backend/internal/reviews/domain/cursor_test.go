package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

func TestCursorRoundTripsKeepingMicrosecondPrecision(t *testing.T) {
	t.Parallel()

	// The database keeps microseconds, so the cursor must not lose or invent any.
	at := time.Date(2028, 3, 1, 10, 0, 0, 123456000, time.UTC)
	for _, id := range []int64{1, 42, 9_000_000_000} {
		encoded := domain.Cursor{CreatedAt: at, ID: id}.Encode()
		got, err := domain.ParseCursor(encoded)
		if err != nil {
			t.Fatalf("ParseCursor(%q): %v", encoded, err)
		}
		if !got.CreatedAt.Equal(at) || got.ID != id {
			t.Errorf("round trip = %+v, want %v and %d", got, at, id)
		}
	}
}

func TestCursorIsOpaqueAndSafeInAURL(t *testing.T) {
	t.Parallel()

	encoded := domain.Cursor{CreatedAt: time.Date(2028, 3, 1, 10, 0, 0, 0, time.UTC), ID: 7}.Encode()
	if strings.ContainsAny(encoded, "+/= ?&#") {
		t.Errorf("cursor %q needs escaping in a query string", encoded)
	}
}

func TestParseCursorRefusesAnythingItDidNotIssue(t *testing.T) {
	t.Parallel()

	for _, bad := range []string{
		"", "not-a-cursor", "!!!", "MTIz", // "123": no separator
		"MTIzLmFiYw", // "123.abc": id is not a number
		"YWJjLjEyMw", // "abc.123": time is not a number
		"MTIzLjA",    // "123.0": ids start at 1
		"MTIzLi0x",   // "123.-1"
		strings.Repeat("A", 500),
	} {
		if _, err := domain.ParseCursor(bad); !errors.Is(err, domain.ErrCursorInvalid) {
			t.Errorf("ParseCursor(%q) err = %v, want ErrCursorInvalid", bad, err)
		}
	}
}
