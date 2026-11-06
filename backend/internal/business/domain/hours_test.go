package domain_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

func TestParseClock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw     string
		want    domain.Clock
		wantErr bool
	}{
		{"00:00", 0, false},
		{"09:30", 9*60 + 30, false},
		{"23:59", 23*60 + 59, false},
		{"24:00", 0, true},
		{"09:60", 0, true},
		{"9:30", 0, true},
		{"0930", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		got, err := domain.ParseClock(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseClock(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			continue
		}
		if err != nil && !errors.Is(err, domain.ErrClockInvalid) {
			t.Errorf("ParseClock(%q) error = %v, want ErrClockInvalid", tt.raw, err)
		}
		if got != tt.want {
			t.Errorf("ParseClock(%q) = %d, want %d", tt.raw, got, tt.want)
		}
	}

	if got := domain.Clock(9*60 + 5).String(); got != "09:05" {
		t.Errorf("String() = %q, want 09:05", got)
	}
}

func TestValidateSlots(t *testing.T) {
	t.Parallel()

	slot := func(weekday int, from, to domain.Clock) domain.Slot {
		return domain.Slot{Weekday: weekday, Start: from, End: to}
	}
	h := func(hh int) domain.Clock { return domain.Clock(hh * 60) }

	tests := []struct {
		name  string
		slots []domain.Slot
		want  error
	}{
		{"no slots means closed", nil, nil},
		{"split shift", []domain.Slot{slot(1, h(9), h(13)), slot(1, h(16), h(20))}, nil},
		{"touching slots", []domain.Slot{slot(1, h(9), h(13)), slot(1, h(13), h(20))}, nil},
		{"same hours on different days", []domain.Slot{slot(1, h(9), h(13)), slot(2, h(9), h(13))}, nil},
		{"sunday and saturday", []domain.Slot{slot(0, h(9), h(13)), slot(6, h(9), h(13))}, nil},
		{"overlap", []domain.Slot{slot(1, h(9), h(13)), slot(1, h(12), h(20))}, domain.ErrSlotsOverlap},
		{"overlap given out of order", []domain.Slot{slot(1, h(12), h(20)), slot(1, h(9), h(13))}, domain.ErrSlotsOverlap},
		{"duplicate", []domain.Slot{slot(1, h(9), h(13)), slot(1, h(9), h(13))}, domain.ErrSlotsOverlap},
		{"end before start", []domain.Slot{slot(1, h(13), h(9))}, domain.ErrSlotInvalid},
		{"empty slot", []domain.Slot{slot(1, h(9), h(9))}, domain.ErrSlotInvalid},
		{"weekday too high", []domain.Slot{slot(7, h(9), h(13))}, domain.ErrSlotInvalid},
		{"negative weekday", []domain.Slot{slot(-1, h(9), h(13))}, domain.ErrSlotInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := domain.ValidateSlots(tt.slots); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}
