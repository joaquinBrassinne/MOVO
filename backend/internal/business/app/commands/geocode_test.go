package commands_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type spyGeocoder struct {
	queries []string
	place   business.Place
	err     error
}

func (s *spyGeocoder) Geocode(_ context.Context, query string) (business.Place, error) {
	s.queries = append(s.queries, query)
	return s.place, s.err
}

func TestGeocodeAddressBuildsAnArgentinaQueryForTheOwner(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	geocoder := &spyGeocoder{place: business.Place{Latitude: -34.58, Longitude: -58.43, DisplayName: "Honduras 4820"}}

	got, err := commands.NewGeocodeAddress(businesses, geocoder).Handle(t.Context(), commands.GeocodeAddressInput{
		OwnerID: 7, Address: "  Honduras 4820 ", City: " Buenos Aires ",
	})
	if err != nil || got.Latitude != -34.58 {
		t.Fatalf("Handle = %+v, %v", got, err)
	}
	if len(geocoder.queries) != 1 || geocoder.queries[0] != "Honduras 4820, Buenos Aires, Argentina" {
		t.Errorf("queries = %q", geocoder.queries)
	}
}

func TestGeocodeAddressRejectionsNeverReachTheGeocoder(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	geocoder := &spyGeocoder{}
	geocode := commands.NewGeocodeAddress(businesses, geocoder)

	tests := []struct {
		name  string
		input commands.GeocodeAddressInput
		want  error
	}{
		{"caller without a business", commands.GeocodeAddressInput{OwnerID: 99, Address: "x", City: "y"}, business.ErrBusinessNotFound},
		{"empty address", commands.GeocodeAddressInput{OwnerID: 7, Address: " ", City: "y"}, domain.ErrAddressInvalid},
		{"address too long", commands.GeocodeAddressInput{OwnerID: 7, Address: strings.Repeat("a", 201), City: "y"}, domain.ErrAddressInvalid},
		{"empty city", commands.GeocodeAddressInput{OwnerID: 7, Address: "x", City: ""}, domain.ErrCityInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := geocode.Handle(t.Context(), tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
	if len(geocoder.queries) != 0 {
		t.Errorf("geocoder calls = %q, the shared public service must only see valid requests", geocoder.queries)
	}
}

func TestGeocodeAddressPassesTheGeocoderErrorsThrough(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	for _, want := range []error{business.ErrAddressNotFound, business.ErrGeocoderBusy, business.ErrGeocoderUnavailable} {
		geocode := commands.NewGeocodeAddress(businesses, &spyGeocoder{err: want})
		if _, err := geocode.Handle(t.Context(), commands.GeocodeAddressInput{OwnerID: 7, Address: "x", City: "y"}); !errors.Is(err, want) {
			t.Errorf("error = %v, want %v", err, want)
		}
	}
}
