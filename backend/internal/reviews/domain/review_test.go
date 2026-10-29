package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/reviews/domain"
)

var redeemedAt = time.Date(2028, 3, 1, 10, 0, 0, 0, time.UTC)

func TestStateOfAgreesWithTheRulesOfNewReview(t *testing.T) {
	t.Parallel()

	inside := redeemedAt.Add(domain.ReviewWindow)
	for name, tt := range map[string]struct {
		redeemed time.Time
		reviewed bool
		now      time.Time
		want     domain.FormState
	}{
		"not redeemed":                   {time.Time{}, false, redeemedAt, domain.FormNotRedeemed},
		"open the day of the redemption": {redeemedAt, false, redeemedAt, domain.FormOpen},
		"open the last moment":           {redeemedAt, false, inside, domain.FormOpen},
		"closed a moment later":          {redeemedAt, false, inside.Add(time.Second), domain.FormWindowClosed},
		"reviewed":                       {redeemedAt, true, redeemedAt, domain.FormReviewed},
		"reviewed beats the window":      {redeemedAt, true, inside.Add(time.Hour), domain.FormReviewed},
	} {
		if got := domain.StateOf(tt.redeemed, tt.reviewed, tt.now); got != tt.want {
			t.Errorf("%s: %q, want %q", name, got, tt.want)
		}
	}
}

func TestStarFilterRefusesWhatNoReviewCanMatch(t *testing.T) {
	t.Parallel()

	for _, ok := range []domain.StarFilter{{}, {Min: 4}, {Max: 2}, {Min: 5, Max: 5}, {Min: 2, Max: 3}} {
		if err := ok.Validate(); err != nil {
			t.Errorf("%+v = %v, want it valid", ok, err)
		}
	}
	for _, bad := range []domain.StarFilter{{Min: 6}, {Max: 6}, {Min: -1}, {Min: 5, Max: 3}} {
		if err := bad.Validate(); !errors.Is(err, domain.ErrStarFilterInvalid) {
			t.Errorf("%+v = %v, want ErrStarFilterInvalid", bad, err)
		}
	}
}

func redemption() domain.Redemption {
	return domain.Redemption{CouponID: 7, BranchID: 3, BusinessID: 2, CustomerID: 9, CustomerName: "Ana García", RedeemedAt: redeemedAt}
}

func TestNewReviewCopiesWhereAndWhoFromTheRedemption(t *testing.T) {
	t.Parallel()

	now := redeemedAt.Add(time.Hour)
	got, err := domain.NewReview(redemption(), domain.Input{Rating: 5, Comment: "  Muy buena clase  ", ShowName: true}, now)
	if err != nil {
		t.Fatalf("NewReview: %v", err)
	}
	want := domain.Review{
		CouponID: 7, BranchID: 3, BusinessID: 2, CustomerID: 9,
		Rating: 5, Comment: "Muy buena clase", DisplayName: "Ana G.", ShowName: true,
		Status: domain.StatusVisible, CreatedAt: now,
	}
	if got != want {
		t.Errorf("NewReview = %+v, want %+v", got, want)
	}
}

func TestNewReviewRefusesACouponThatWasNotRedeemed(t *testing.T) {
	t.Parallel()

	pending := redemption()
	pending.RedeemedAt = time.Time{}
	_, err := domain.NewReview(pending, domain.Input{Rating: 5}, redeemedAt)
	if !errors.Is(err, domain.ErrNotRedeemed) {
		t.Errorf("err = %v, want ErrNotRedeemed", err)
	}
}

func TestNewReviewWindowIs30DaysFromTheRedemption(t *testing.T) {
	t.Parallel()

	deadline := redeemedAt.Add(30 * 24 * time.Hour)
	tests := []struct {
		name string
		now  time.Time
		ok   bool
	}{
		{"right after redeeming", redeemedAt, true},
		{"one second before the deadline", deadline.Add(-time.Second), true},
		{"exactly at the deadline", deadline, true},
		{"one second after the deadline", deadline.Add(time.Second), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewReview(redemption(), domain.Input{Rating: 4}, tt.now)
			if tt.ok && err != nil {
				t.Errorf("err = %v, want none", err)
			}
			if !tt.ok && !errors.Is(err, domain.ErrWindowClosed) {
				t.Errorf("err = %v, want ErrWindowClosed", err)
			}
		})
	}
}

func TestNewReviewRatingIsOneToFive(t *testing.T) {
	t.Parallel()

	for _, rating := range []int{-1, 0, 6, 100} {
		if _, err := domain.NewReview(redemption(), domain.Input{Rating: rating}, redeemedAt); !errors.Is(err, domain.ErrRatingInvalid) {
			t.Errorf("rating %d: err = %v, want ErrRatingInvalid", rating, err)
		}
	}
	for rating := 1; rating <= 5; rating++ {
		if _, err := domain.NewReview(redemption(), domain.Input{Rating: rating}, redeemedAt); err != nil {
			t.Errorf("rating %d: %v", rating, err)
		}
	}
}

func TestNewReviewCommentIsOptionalPlainTextOfUpTo500Characters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment string
		want    domain.TextReason // empty when accepted
	}{
		{"none", "", ""},
		{"only spaces counts as none", "   ", ""},
		{"500 characters", strings.Repeat("á", 500), ""},
		{"501 characters", strings.Repeat("á", 501), domain.TextInvalid},
		{"a control character", "hola\x00", domain.TextInvalid},
		{"a link", "mirá www.ejemplo.com", domain.TextHasLink},
		{"a phone number", "escribime al 11 5555-1234", domain.TextHasPhone},
		{"a price is not a phone", "pagué $10.000 la clase", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			review, err := domain.NewReview(redemption(), domain.Input{Rating: 5, Comment: tt.comment}, redeemedAt)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("err = %v, want none", err)
				}
				if strings.TrimSpace(tt.comment) == "" && review.Comment != "" {
					t.Errorf("Comment = %q, want it empty", review.Comment)
				}
				return
			}
			var textErr *domain.TextError
			if !errors.As(err, &textErr) || textErr.Field != "comment" || textErr.Reason != tt.want {
				t.Errorf("err = %v, want a comment TextError with reason %q", err, tt.want)
			}
		})
	}
}

func TestNewReviewDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		customer string
		showName bool
		want     string
	}{
		{"first name and the initial of the last name", "Ana García", true, "Ana G."},
		{"the last word is the surname", "Ana María de la Torre", true, "Ana T."},
		{"the initial is upper case", "luis pérez", true, "luis P."},
		{"a non-ASCII initial", "Ana Épsilon", true, "Ana É."},
		{"extra spaces", "  Ana    García  ", true, "Ana G."},
		{"a single name", "Ana", true, "Ana"},
		{"hidden name", "Ana García", false, "Cliente de MOVO"},
		{"a phone typed as the name is never shown", "5491112345678", true, "Cliente de MOVO"},
		{"a link typed as the name is never shown", "www.ejemplo.com", true, "Cliente de MOVO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := redemption()
			r.CustomerName = tt.customer
			got, err := domain.NewReview(r, domain.Input{Rating: 5, ShowName: tt.showName}, redeemedAt)
			if err != nil {
				t.Fatalf("NewReview: %v", err)
			}
			if got.DisplayName != tt.want || got.ShowName != tt.showName {
				t.Errorf("DisplayName = %q show=%v, want %q show=%v", got.DisplayName, got.ShowName, tt.want, tt.showName)
			}
		})
	}
}
