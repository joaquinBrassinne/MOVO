package queries_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

var base = time.Date(2028, 3, 1, 10, 0, 0, 0, time.UTC)

// fakeReader holds reviews newest first and answers like the store: after the cursor, at
// most limit of them. It records what it was asked.
type fakeReader struct {
	all      []reviews.ListedReview
	err      error
	gotAfter *domain.Cursor
	gotLimit int
}

func (f *fakeReader) ListPublic(_ context.Context, _ int64, after *domain.Cursor, limit int) ([]reviews.ListedReview, error) {
	f.gotAfter, f.gotLimit = after, limit
	if f.err != nil {
		return nil, f.err
	}
	var out []reviews.ListedReview
	for _, r := range f.all {
		if after != nil && (r.CreatedAt.After(after.CreatedAt) || (r.CreatedAt.Equal(after.CreatedAt) && r.ID >= after.ID)) {
			continue
		}
		if len(out) == limit {
			break
		}
		out = append(out, r)
	}
	return out, nil
}

// reviewsNewestFirst makes n reviews, the first being the newest.
func reviewsNewestFirst(n int) []reviews.ListedReview {
	out := make([]reviews.ListedReview, n)
	for i := range out {
		out[i] = reviews.ListedReview{ID: int64(n - i), Rating: 5, DisplayName: fmt.Sprintf("Ana %d", i), CreatedAt: base.Add(-time.Duration(i) * time.Hour)}
	}
	return out
}

func TestListReviewsPagesThroughEverythingOnceWithACursor(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{all: reviewsNewestFirst(5)}
	q := queries.NewListReviews(reader)

	var seen []int64
	cursor := ""
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("the pages never end")
		}
		page, err := q.Handle(t.Context(), queries.ListReviewsInput{BranchID: 1, Cursor: cursor, Limit: 2})
		if err != nil {
			t.Fatalf("Handle: %v", err)
		}
		for _, r := range page.Reviews {
			seen = append(seen, r.ID)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	want := []int64{5, 4, 3, 2, 1}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Errorf("reviews seen = %v, want %v, each exactly once and newest first", seen, want)
	}
}

func TestListReviewsAsksOneMoreToKnowIfThereIsANextPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		total      int
		limit      int
		wantItems  int
		wantNext   bool
		wantAsking int
	}{
		{"more than a page", 3, 2, 2, true, 3},
		{"exactly a page", 2, 2, 2, false, 3},
		{"less than a page", 1, 2, 1, false, 3},
		{"none", 0, 2, 0, false, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := &fakeReader{all: reviewsNewestFirst(tt.total)}
			page, err := queries.NewListReviews(reader).Handle(t.Context(), queries.ListReviewsInput{BranchID: 1, Limit: tt.limit})
			if err != nil {
				t.Fatalf("Handle: %v", err)
			}
			if len(page.Reviews) != tt.wantItems || (page.NextCursor != "") != tt.wantNext || reader.gotLimit != tt.wantAsking {
				t.Errorf("items %d, next %q, asked %d; want items %d, next=%v, asked %d",
					len(page.Reviews), page.NextCursor, reader.gotLimit, tt.wantItems, tt.wantNext, tt.wantAsking)
			}
		})
	}
}

func TestListReviewsNextCursorPointsAfterTheLastReviewOfThePage(t *testing.T) {
	t.Parallel()

	all := reviewsNewestFirst(3)
	page, err := queries.NewListReviews(&fakeReader{all: all}).Handle(t.Context(), queries.ListReviewsInput{BranchID: 1, Limit: 2})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	cursor, err := domain.ParseCursor(page.NextCursor)
	if err != nil {
		t.Fatalf("ParseCursor: %v", err)
	}
	if last := all[1]; cursor.ID != last.ID || !cursor.CreatedAt.Equal(last.CreatedAt) {
		t.Errorf("cursor = %+v, want the position of the last review of the page %+v", cursor, last)
	}
}

func TestListReviewsLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		limit     int
		wantAsked int
		wantErr   error
	}{
		{"zero takes the default", 0, queries.DefaultLimit + 1, nil},
		{"the maximum is allowed", queries.MaxLimit, queries.MaxLimit + 1, nil},
		{"over the maximum", queries.MaxLimit + 1, 0, queries.ErrLimitInvalid},
		{"negative", -1, 0, queries.ErrLimitInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := &fakeReader{}
			_, err := queries.NewListReviews(reader).Handle(t.Context(), queries.ListReviewsInput{BranchID: 1, Limit: tt.limit})
			if !errors.Is(err, tt.wantErr) || reader.gotLimit != tt.wantAsked {
				t.Errorf("err = %v, asked %d; want %v, asked %d", err, reader.gotLimit, tt.wantErr, tt.wantAsked)
			}
		})
	}
}

func TestListReviewsRefusesACursorItDidNotIssueWithoutReadingAnything(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{all: reviewsNewestFirst(3)}
	_, err := queries.NewListReviews(reader).Handle(t.Context(), queries.ListReviewsInput{BranchID: 1, Cursor: "garbage"})
	if !errors.Is(err, domain.ErrCursorInvalid) || reader.gotLimit != 0 {
		t.Errorf("err = %v, asked %d; want ErrCursorInvalid and no read", err, reader.gotLimit)
	}
}

func TestListReviewsPassesTheCursorToTheStore(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{all: reviewsNewestFirst(3)}
	cursor := domain.Cursor{CreatedAt: base, ID: 3}
	if _, err := queries.NewListReviews(reader).Handle(t.Context(), queries.ListReviewsInput{BranchID: 1, Cursor: cursor.Encode(), Limit: 2}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if reader.gotAfter == nil || reader.gotAfter.ID != 3 || !reader.gotAfter.CreatedAt.Equal(base) {
		t.Errorf("the store was asked after %+v, want %+v", reader.gotAfter, cursor)
	}
}

func TestListReviewsReportsABranchTheStoreDoesNotShow(t *testing.T) {
	t.Parallel()

	_, err := queries.NewListReviews(&fakeReader{err: reviews.ErrBranchNotFound}).Handle(t.Context(), queries.ListReviewsInput{BranchID: 1})
	if !errors.Is(err, reviews.ErrBranchNotFound) {
		t.Errorf("err = %v, want ErrBranchNotFound", err)
	}
}
