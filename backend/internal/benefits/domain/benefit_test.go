package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestNewBenefit(t *testing.T) {
	t.Parallel()

	for _, percent := range []int{10, 15, 20} {
		got, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypePercent, Percent: percent, MonthlyLimit: 30})
		if err != nil {
			t.Fatalf("percent %d: %v", percent, err)
		}
		if got.Type != domain.TypePercent || got.Percent != percent || got.GiftDescription != "" || got.MonthlyLimit != 30 {
			t.Errorf("percent %d: benefit = %+v", percent, got)
		}
		if !got.Active {
			t.Error("a new benefit must be active")
		}
	}

	gift, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypeWelcomeGift, GiftDescription: "  Una clase gratis ", MonthlyLimit: 5})
	if err != nil {
		t.Fatalf("welcome gift: %v", err)
	}
	if gift.Type != domain.TypeWelcomeGift || gift.GiftDescription != "Una clase gratis" || gift.Percent != 0 {
		t.Errorf("welcome gift = %+v, want the trimmed description and no percent", gift)
	}

	tests := []struct {
		name  string
		input domain.BenefitInput
		want  error
	}{
		{"percent 25", domain.BenefitInput{Type: domain.TypePercent, Percent: 25, MonthlyLimit: 1}, domain.ErrPercentInvalid},
		{"percent missing", domain.BenefitInput{Type: domain.TypePercent, MonthlyLimit: 1}, domain.ErrPercentInvalid},
		{"gift missing", domain.BenefitInput{Type: domain.TypeWelcomeGift, MonthlyLimit: 1}, domain.ErrGiftDescriptionInvalid},
		{"gift blank", domain.BenefitInput{Type: domain.TypeWelcomeGift, GiftDescription: "  ", MonthlyLimit: 1}, domain.ErrGiftDescriptionInvalid},
		{"gift too long", domain.BenefitInput{Type: domain.TypeWelcomeGift, GiftDescription: strings.Repeat("a", 201), MonthlyLimit: 1}, domain.ErrGiftDescriptionInvalid},
		{"unknown type", domain.BenefitInput{Type: "two_for_one", MonthlyLimit: 1}, domain.ErrBenefitTypeInvalid},
		{"no limit", domain.BenefitInput{Type: domain.TypePercent, Percent: 10}, domain.ErrMonthlyLimitInvalid},
		{"negative limit", domain.BenefitInput{Type: domain.TypePercent, Percent: 10, MonthlyLimit: -1}, domain.ErrMonthlyLimitInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewBenefit(tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBenefitUpdateKeepsIdentityAndState(t *testing.T) {
	t.Parallel()

	current := domain.Benefit{ID: 7, BusinessID: 3, Type: domain.TypePercent, Percent: 10, MonthlyLimit: 20}.Deactivate()

	got, err := current.Update(domain.BenefitInput{Type: domain.TypeWelcomeGift, GiftDescription: "Mate", MonthlyLimit: 8})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != 7 || got.BusinessID != 3 || got.Active {
		t.Errorf("benefit = %+v, want identity and inactive state kept", got)
	}
	if got.Percent != 0 || got.GiftDescription != "Mate" {
		t.Errorf("benefit = %+v, want the percent dropped when it becomes a gift", got)
	}

	back, err := got.Update(domain.BenefitInput{Type: domain.TypePercent, Percent: 15, MonthlyLimit: 8})
	if err != nil {
		t.Fatalf("Update back: %v", err)
	}
	if back.GiftDescription != "" || back.Percent != 15 {
		t.Errorf("benefit = %+v, want the gift dropped when it becomes a percent", back)
	}

	if _, err := current.Update(domain.BenefitInput{Type: domain.TypePercent, Percent: 99, MonthlyLimit: 1}); err == nil {
		t.Fatal("an invalid update must fail")
	}
	if current.Percent != 10 {
		t.Error("a failed update must not change the benefit")
	}
}

func TestBenefitActivateAndDeactivate(t *testing.T) {
	t.Parallel()

	b := domain.Benefit{ID: 1, Active: true}
	if off := b.Deactivate(); off.Active || off.ID != 1 {
		t.Errorf("Deactivate = %+v", off)
	}
	if on := b.Deactivate().Activate(); !on.Active {
		t.Errorf("Activate = %+v", on)
	}
}

func TestBenefitHasRoom(t *testing.T) {
	t.Parallel()

	b := domain.Benefit{MonthlyLimit: 3}
	for generated, want := range map[int]bool{0: true, 2: true, 3: false, 4: false} {
		if got := b.HasRoom(generated); got != want {
			t.Errorf("HasRoom(%d) = %v, want %v", generated, got, want)
		}
	}
}

func TestQuotaMonthIsTheCalendarMonthInArgentina(t *testing.T) {
	t.Parallel()

	// 02:00 UTC on 1 March is still 29 February at 23:00 in Argentina.
	start, end := domain.QuotaMonth(time.Date(2028, 3, 1, 2, 0, 0, 0, time.UTC))
	if want := time.Date(2028, 2, 1, 3, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Errorf("start = %v, want %v", start.UTC(), want)
	}
	if want := time.Date(2028, 3, 1, 3, 0, 0, 0, time.UTC); !end.Equal(want) {
		t.Errorf("end = %v, want %v", end.UTC(), want)
	}

	// Noon UTC on 15 March is the same month in Argentina.
	start, end = domain.QuotaMonth(time.Date(2028, 3, 15, 12, 0, 0, 0, time.UTC))
	if !start.Equal(time.Date(2028, 3, 1, 3, 0, 0, 0, time.UTC)) || !end.Equal(time.Date(2028, 4, 1, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("interval = [%v, %v)", start.UTC(), end.UTC())
	}

	// December rolls over into the next year.
	_, end = domain.QuotaMonth(time.Date(2028, 12, 31, 20, 0, 0, 0, time.UTC))
	if !end.Equal(time.Date(2029, 1, 1, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("december end = %v", end.UTC())
	}
}
