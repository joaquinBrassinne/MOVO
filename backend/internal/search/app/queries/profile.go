package queries

import (
	"context"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// ProfileView is a public profile with the URLs that show its photos.
type ProfileView struct {
	search.Profile
	ImageURLs []string
}

// BusinessProfile serves the public page of a business.
type BusinessProfile struct {
	finder search.ProfileFinder
	urls   search.ImageURLs
}

// NewBusinessProfile builds the query.
func NewBusinessProfile(finder search.ProfileFinder, urls search.ImageURLs) *BusinessProfile {
	return &BusinessProfile{finder: finder, urls: urls}
}

// Handle returns the profile of the business with that slug, or search.ErrBusinessNotFound.
// A photo whose URL cannot be made is left out: the page is worth more than the photo.
func (b *BusinessProfile) Handle(ctx context.Context, slug string) (ProfileView, error) {
	p, err := b.finder.Profile(ctx, slug)
	if err != nil {
		return ProfileView{}, err
	}
	view := ProfileView{Profile: p, ImageURLs: make([]string, 0, len(p.ImageKeys))}
	for _, key := range p.ImageKeys {
		url, err := b.urls.PresignDownload(ctx, key)
		if err != nil {
			continue
		}
		view.ImageURLs = append(view.ImageURLs, url)
	}
	return view, nil
}
