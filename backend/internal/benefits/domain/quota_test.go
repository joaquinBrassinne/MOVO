package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func intp(n int) *int { return &n }

func TestQuotaHasRoom(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		quota domain.Quota
		want  bool
	}{
		"no limit":          {domain.Quota{Limit: nil, Generated: 1_000_000}, true},
		"below the limit":   {domain.Quota{Limit: intp(3), Generated: 2}, true},
		"at the limit":      {domain.Quota{Limit: intp(3), Generated: 3}, false},
		"above the limit":   {domain.Quota{Limit: intp(3), Generated: 4}, false},
		"the first coupon":  {domain.Quota{Limit: intp(1), Generated: 0}, true},
		"limit of one used": {domain.Quota{Limit: intp(1), Generated: 1}, false},
	} {
		if got := tc.quota.HasRoom(); got != tc.want {
			t.Errorf("%s: HasRoom() = %v, want %v", name, got, tc.want)
		}
	}
}

func TestValidateQuotaLimit(t *testing.T) {
	t.Parallel()

	if err := domain.ValidateQuotaLimit(nil, 500); err != nil {
		t.Errorf("no limit: %v, want it always allowed", err)
	}
	if err := domain.ValidateQuotaLimit(intp(18), 18); err != nil {
		t.Errorf("limit equal to the generated: %v, want it allowed", err)
	}
	for _, limit := range []int{0, -1} {
		if err := domain.ValidateQuotaLimit(intp(limit), 0); !errors.Is(err, domain.ErrQuotaLimitInvalid) {
			t.Errorf("limit %d: %v, want ErrQuotaLimitInvalid", limit, err)
		}
	}

	err := domain.ValidateQuotaLimit(intp(10), 18)
	var below *domain.QuotaBelowGeneratedError
	if !errors.As(err, &below) || below.Generated != 18 {
		t.Errorf("limit 10 with 18 generated: %v, want a QuotaBelowGeneratedError carrying 18", err)
	}
}

func TestQuotaResetsOnTheFirstOfNextMonthInArgentina(t *testing.T) {
	t.Parallel()

	// 02:00 UTC on 1 March is still 29 February in Argentina: it resets on 1 March.
	got := domain.QuotaResetsOn(time.Date(2028, 3, 1, 2, 0, 0, 0, time.UTC))
	if want := time.Date(2028, 3, 1, 3, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("resets on %v, want %v", got.UTC(), want)
	}
	got = domain.QuotaResetsOn(time.Date(2028, 10, 27, 12, 0, 0, 0, time.UTC))
	if want := time.Date(2028, 11, 1, 3, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("resets on %v, want %v", got.UTC(), want)
	}
}
