package commands

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// GeocodeAddressInput is the address an owner typed in the branch form.
type GeocodeAddressInput struct {
	OwnerID int64
	Address string
	City    string
}

// GeocodeAddress finds where an address is so the owner can place the pin.
type GeocodeAddress struct {
	businesses business.BusinessRepository
	geocoder   business.Geocoder
}

// NewGeocodeAddress wires the GeocodeAddress use case.
func NewGeocodeAddress(businesses business.BusinessRepository, geocoder business.Geocoder) *GeocodeAddress {
	return &GeocodeAddress{businesses: businesses, geocoder: geocoder}
}

// Handle looks the address up. Only an owner with a business may use it, which
// keeps the shared public geocoder away from anonymous traffic. The search is
// always explicit (one call per press of a button), never as-you-type.
func (c *GeocodeAddress) Handle(ctx context.Context, in GeocodeAddressInput) (business.Place, error) {
	if _, err := editableBusiness(ctx, c.businesses, in.OwnerID); err != nil {
		return business.Place{}, err
	}

	address, city := strings.TrimSpace(in.Address), strings.TrimSpace(in.City)
	if address == "" || utf8.RuneCountInString(address) > domain.MaxAddressLen {
		return business.Place{}, domain.ErrAddressInvalid
	}
	if city == "" || utf8.RuneCountInString(city) > domain.MaxCityLen {
		return business.Place{}, domain.ErrCityInvalid
	}

	place, err := c.geocoder.Geocode(ctx, address+", "+city+", Argentina")
	if err != nil {
		return business.Place{}, fmt.Errorf("geocode: %w", err)
	}
	return place, nil
}
