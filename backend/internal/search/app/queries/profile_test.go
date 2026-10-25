package queries_test

import (
	"context"
	"errors"
	"testing"

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
			view, err := queries.NewBusinessProfile(tt.finder, tt.urls).Handle(t.Context(), "studio")
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
