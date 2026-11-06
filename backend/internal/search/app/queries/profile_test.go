package queries_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
	"github.com/joaquinBrassinne/MOVO/internal/search/app/queries"
)

type fakeProfiles struct {
	profile search.Profile
	err     error
}

func (f fakeProfiles) Profile(context.Context, string) (search.Profile, error) {
	return f.profile, f.err
}

type fakeURLs struct{ failFor string }

func (f fakeURLs) PresignDownload(_ context.Context, key string) (string, error) {
	if key == f.failFor {
		return "", errors.New("storage down")
	}
	return "https://files.test/" + key, nil
}

func TestBusinessProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		finder   fakeProfiles
		urls     fakeURLs
		wantErr  error
		wantURLs []string
	}{
		{
			name:     "photos keep the owner's order",
			finder:   fakeProfiles{profile: search.Profile{ImageKeys: []string{"a.jpg", "b.jpg"}}},
			wantURLs: []string{"https://files.test/a.jpg", "https://files.test/b.jpg"},
		},
		{
			name:     "a photo whose URL fails is left out, the page still loads",
			finder:   fakeProfiles{profile: search.Profile{ImageKeys: []string{"a.jpg", "b.jpg", "c.jpg"}}},
			urls:     fakeURLs{failFor: "b.jpg"},
			wantURLs: []string{"https://files.test/a.jpg", "https://files.test/c.jpg"},
		},
		{
			name:     "no photos gives an empty list, not nil",
			finder:   fakeProfiles{},
			wantURLs: []string{},
		},
		{
			name:    "a business that is not public is reported as not found",
			finder:  fakeProfiles{err: search.ErrBusinessNotFound},
			wantErr: search.ErrBusinessNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := queries.NewBusinessProfile(tt.finder, tt.urls, nil).Handle(t.Context(), "studio")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if view.ImageURLs == nil || len(view.ImageURLs) != len(tt.wantURLs) {
				t.Fatalf("image urls = %#v, want %#v", view.ImageURLs, tt.wantURLs)
			}
			for i := range tt.wantURLs {
				if view.ImageURLs[i] != tt.wantURLs[i] {
					t.Errorf("image url %d = %q, want %q", i, view.ImageURLs[i], tt.wantURLs[i])
				}
			}
		})
	}
}

func TestBusinessProfileReportsEachBranchOpenStatusAtTheClock(t *testing.T) {
	t.Parallel()

	profile := search.Profile{Branches: []search.ProfileBranch{
		{BranchID: 1, Hours: []search.Slot{{Weekday: 1, Start: "09:00", End: "13:00"}, {Weekday: 1, Start: "16:00", End: "20:00"}}},
		{BranchID: 2, Hours: []search.Slot{{Weekday: 2, Start: "08:00", End: "12:00"}}},
		{BranchID: 3},
	}}
	zone, err := time.LoadLocation(search.ZoneName)
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	monday := time.Date(2026, 10, 26, 10, 30, 0, 0, zone)

	view, err := queries.NewBusinessProfile(fakeProfiles{profile: profile}, fakeURLs{}, func() time.Time { return monday }).
		Handle(t.Context(), "studio")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	want := map[int64]search.OpenStatus{
		1: {Open: true, ClosesAt: "13:00"},
		2: {HasNext: true, OpensAt: "08:00", OpensWeekday: 2},
		3: {},
	}
	for id, status := range want {
		if got := view.OpenNow[id]; got != status {
			t.Errorf("branch %d status = %+v, want %+v", id, got, status)
		}
	}
}
