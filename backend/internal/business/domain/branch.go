package domain

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

// Validation errors for branches.
var (
	ErrBranchNameInvalid = errors.New("business: branch name is not valid")
	ErrAddressInvalid    = errors.New("business: branch address is not valid")
	ErrCityInvalid       = errors.New("business: branch city is not valid")
	ErrLocationInvalid   = errors.New("business: branch location is not valid")
	ErrBookingURLInvalid = errors.New("business: booking url is not valid")
)

const (
	maxBranchNameLen = 100
	MaxAddressLen    = 200
	MaxCityLen       = 100
	maxBookingURLLen = 500
)

// Branch is a physical place of a business. Public search runs per branch.
type Branch struct {
	ID         int64
	BusinessID int64
	Name       string
	Address    string
	City       string
	Latitude   float64
	Longitude  float64
	BookingURL string
	Active     bool
}

// NewBranchInput is what an owner provides to create or edit a branch.
type NewBranchInput struct {
	Name       string
	Address    string
	City       string
	Latitude   float64
	Longitude  float64
	BookingURL string
}

// NewBranch validates the input and builds an active branch. BusinessID and ID
// are assigned by the use case and the repository.
func NewBranch(in NewBranchInput) (Branch, error) {
	return Branch{Active: true}.Update(in)
}

// Update validates the input and returns the branch with the new details,
// keeping its identity and whether it is active.
func (b Branch) Update(in NewBranchInput) (Branch, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxBranchNameLen {
		return Branch{}, ErrBranchNameInvalid
	}
	address := strings.TrimSpace(in.Address)
	if address == "" || utf8.RuneCountInString(address) > MaxAddressLen {
		return Branch{}, ErrAddressInvalid
	}
	city := strings.TrimSpace(in.City)
	if city == "" || utf8.RuneCountInString(city) > MaxCityLen {
		return Branch{}, ErrCityInvalid
	}
	if !validLocation(in.Latitude, in.Longitude) {
		return Branch{}, ErrLocationInvalid
	}
	bookingURL, err := validateBookingURL(in.BookingURL)
	if err != nil {
		return Branch{}, err
	}

	b.Name, b.Address, b.City = name, address, city
	b.Latitude, b.Longitude = in.Latitude, in.Longitude
	b.BookingURL = bookingURL
	return b, nil
}

// validLocation rejects out-of-range coordinates and the (0, 0) point, which is
// what a form sends when the owner never placed the pin.
func validLocation(lat, lon float64) bool {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return false
	}
	return lat != 0 || lon != 0
}

// validateBookingURL accepts an empty value or an absolute http(s) URL. Any other
// scheme is refused because the link is rendered to the public.
func validateBookingURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if len(raw) > maxBookingURLLen {
		return "", ErrBookingURLInvalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", ErrBookingURLInvalid
	}
	return raw, nil
}
