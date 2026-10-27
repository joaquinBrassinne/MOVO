package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

func TestNewBenefitAcceptsAnyPercentFromOneToOneHundred(t *testing.T) {
	t.Parallel()

	for _, percent := range []int{1, 10, 15, 20, 37, 100} {
		got, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypePercent, Percent: percent})
		if err != nil {
			t.Fatalf("percent %d: %v", percent, err)
		}
		if got.Type != domain.TypePercent || got.Percent != percent || got.AmountARS != 0 || got.Title != "" {
			t.Errorf("percent %d: benefit = %+v", percent, got)
		}
		if !got.Active || got.AdminPaused {
			t.Errorf("percent %d: a new benefit must be active and not paused, got %+v", percent, got)
		}
	}
}

func TestNewBenefitOfEachType(t *testing.T) {
	t.Parallel()

	amount, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypeAmount, AmountARS: 2000, Conditions: "  Solo lunes "})
	if err != nil || amount.AmountARS != 2000 || amount.Percent != 0 || amount.Conditions != "Solo lunes" {
		t.Errorf("amount = %+v, %v", amount, err)
	}

	custom, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypeCustom, Title: "  Una clase gratis "})
	if err != nil || custom.Title != "Una clase gratis" || custom.Percent != 0 || custom.AmountARS != 0 || custom.Conditions != "" {
		t.Errorf("custom = %+v, %v", custom, err)
	}

	// The fields of another type are dropped, not stored.
	pct, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypePercent, Percent: 10, AmountARS: 5, Title: "x"})
	if err != nil || pct.AmountARS != 0 || pct.Title != "" {
		t.Errorf("percent = %+v, %v; want the amount and the title dropped", pct, err)
	}
}

func TestNewBenefitRejectsWhatDoesNotFit(t *testing.T) {
	t.Parallel()

	long := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name  string
		input domain.BenefitInput
		want  error
	}{
		{"percent zero", domain.BenefitInput{Type: domain.TypePercent, Percent: 0}, domain.ErrPercentInvalid},
		{"percent negative", domain.BenefitInput{Type: domain.TypePercent, Percent: -5}, domain.ErrPercentInvalid},
		{"percent 101", domain.BenefitInput{Type: domain.TypePercent, Percent: 101}, domain.ErrPercentInvalid},
		{"amount zero", domain.BenefitInput{Type: domain.TypeAmount, AmountARS: 0}, domain.ErrAmountInvalid},
		{"amount negative", domain.BenefitInput{Type: domain.TypeAmount, AmountARS: -100}, domain.ErrAmountInvalid},
		{"old welcome gift", domain.BenefitInput{Type: "welcome_gift", Title: "x"}, domain.ErrBenefitTypeInvalid},
		{"unknown type", domain.BenefitInput{Type: "two_for_one"}, domain.ErrBenefitTypeInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := domain.NewBenefit(tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}

	texts := []struct {
		name   string
		input  domain.BenefitInput
		field  string
		reason domain.TextReason
	}{
		{"custom without title", domain.BenefitInput{Type: domain.TypeCustom}, "title", domain.TextInvalid},
		{"custom blank title", domain.BenefitInput{Type: domain.TypeCustom, Title: "   "}, "title", domain.TextInvalid},
		{"title of 81", domain.BenefitInput{Type: domain.TypeCustom, Title: long(81)}, "title", domain.TextInvalid},
		{"conditions of 121", domain.BenefitInput{Type: domain.TypePercent, Percent: 10, Conditions: long(121)}, "conditions", domain.TextInvalid},
		{"title with a line break", domain.BenefitInput{Type: domain.TypeCustom, Title: "uno\ndos"}, "title", domain.TextInvalid},
		{"title with a url", domain.BenefitInput{Type: domain.TypeCustom, Title: "Mirá https://promo.example"}, "title", domain.TextHasLink},
		{"title with www", domain.BenefitInput{Type: domain.TypeCustom, Title: "entrá a www.promo"}, "title", domain.TextHasLink},
		{"title with a bare domain", domain.BenefitInput{Type: domain.TypeCustom, Title: "promo.com.ar gratis"}, "title", domain.TextHasLink},
		{"conditions with a link", domain.BenefitInput{Type: domain.TypeAmount, AmountARS: 5, Conditions: "más en wa.me/549111"}, "conditions", domain.TextHasLink},
		{"title with a phone", domain.BenefitInput{Type: domain.TypeCustom, Title: "Llamá al 11 4567-8901"}, "title", domain.TextHasPhone},
		{"conditions with a phone", domain.BenefitInput{Type: domain.TypePercent, Percent: 10, Conditions: "escribí +54 9 351 123 4567"}, "conditions", domain.TextHasPhone},
	}
	for _, tt := range texts {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewBenefit(tt.input)
			var text *domain.TextError
			if !errors.As(err, &text) || text.Field != tt.field || text.Reason != tt.reason {
				t.Fatalf("error = %v, want %s/%s", err, tt.field, tt.reason)
			}
		})
	}
}

func TestPlainTextThatLooksHarmlessIsAccepted(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"2x1 en clases de lunes a viernes",
		"Válido de 18.30 a 20 hs",
		"Compra mínima $10.000",
		"Hasta agotar stock. Comprando hoy",
	} {
		if _, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypeCustom, Title: text, Conditions: text}); err != nil {
			t.Errorf("%q rejected: %v", text, err)
		}
	}
	// The limit counts characters, not bytes.
	if _, err := domain.NewBenefit(domain.BenefitInput{Type: domain.TypeCustom, Title: strings.Repeat("ñ", 80)}); err != nil {
		t.Errorf("80 multi-byte characters rejected: %v", err)
	}
}

func TestShortLabelOfEachType(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		benefit domain.Benefit
		want    string
	}{
		{domain.Benefit{Type: domain.TypePercent, Percent: 15}, "-15%"},
		{domain.Benefit{Type: domain.TypePercent, Percent: 100}, "-100%"},
		{domain.Benefit{Type: domain.TypeAmount, AmountARS: 150}, "-$150"},
		{domain.Benefit{Type: domain.TypeAmount, AmountARS: 2000}, "-$2.000"},
		{domain.Benefit{Type: domain.TypeAmount, AmountARS: 1234567}, "-$1.234.567"},
		{domain.Benefit{Type: domain.TypeCustom, Title: "Una clase gratis"}, "Beneficio"},
	} {
		if got := tt.benefit.ShortLabel(); got != tt.want {
			t.Errorf("ShortLabel(%+v) = %q, want %q", tt.benefit, got, tt.want)
		}
	}
}

func TestBenefitUpdateKeepsIdentityAndState(t *testing.T) {
	t.Parallel()

	current := domain.Benefit{ID: 7, BusinessID: 3, Type: domain.TypePercent, Percent: 10, AdminPaused: true}.Deactivate()

	got, err := current.Update(domain.BenefitInput{Type: domain.TypeCustom, Title: "Mate"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != 7 || got.BusinessID != 3 || got.Active || !got.AdminPaused {
		t.Errorf("benefit = %+v, want identity, the inactive state and the admin pause kept", got)
	}
	if got.Percent != 0 || got.Title != "Mate" {
		t.Errorf("benefit = %+v, want the percent dropped when it becomes a custom benefit", got)
	}

	back, err := got.Update(domain.BenefitInput{Type: domain.TypePercent, Percent: 15})
	if err != nil {
		t.Fatalf("Update back: %v", err)
	}
	if back.Title != "" || back.Percent != 15 {
		t.Errorf("benefit = %+v, want the title dropped when it becomes a percent", back)
	}

	if _, err := current.Update(domain.BenefitInput{Type: domain.TypePercent, Percent: 101}); err == nil {
		t.Fatal("an invalid update must fail")
	}
	if current.Percent != 10 {
		t.Error("a failed update must not change the benefit")
	}
}

func TestOwnerAndAdminHaveTheirOwnSwitch(t *testing.T) {
	t.Parallel()

	b := domain.Benefit{ID: 1, Active: true}
	if !b.Offered() {
		t.Error("an active, unpaused benefit must be offered")
	}
	paused := b.PauseByAdmin()
	if paused.Offered() || !paused.AdminPaused || !paused.Active {
		t.Errorf("paused = %+v, want it hidden while the owner's switch stays on", paused)
	}
	// The owner turning it off and on again does not lift the admin's pause.
	if again := paused.Deactivate().Activate(); again.Offered() || !again.AdminPaused {
		t.Errorf("after the owner toggles = %+v, want it still paused", again)
	}
	if resumed := paused.ResumeByAdmin(); !resumed.Offered() || resumed.AdminPaused {
		t.Errorf("resumed = %+v", resumed)
	}
	// Lifting the pause does not turn on what the owner turned off.
	if off := paused.Deactivate().ResumeByAdmin(); off.Offered() || off.AdminPaused {
		t.Errorf("resumed but owner-off = %+v, want it not offered", off)
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
