package search_test

import (
	"errors"
	"math"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

func TestQueryMinRatingIsOptionalAndBetweenZeroAndFive(t *testing.T) {
	t.Parallel()

	valid := search.Query{Lat: -32.17, Lng: -64.11, RadiusM: 5_000, Limit: 20}
	tests := []struct {
		name   string
		rating float64
		ok     bool
	}{
		{"none", 0, true},
		{"a whole number of stars", 4, true},
		{"half a star", 4.5, true},
		{"the most", 5, true},
		{"below zero", -0.1, false},
		{"above five", 5.1, false},
		{"not a number", math.NaN(), false},
		{"infinity", math.Inf(1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := valid
			q.MinRating = tt.rating
			err := q.Validate()
			if tt.ok && err != nil {
				t.Errorf("Validate = %v, want it valid", err)
			}
			var qe *search.QueryError
			if !tt.ok && (!errors.As(err, &qe) || qe.Field != "min_rating" || !errors.Is(err, search.ErrInvalidQuery)) {
				t.Errorf("Validate = %v, want a min_rating QueryError", err)
			}
		})
	}
}
