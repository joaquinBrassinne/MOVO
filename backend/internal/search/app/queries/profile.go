package queries

import (
	"context"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/search"
)

// ProfileView is a public profile with the URLs that show its photos.
type ProfileView struct {
	search.Profile
	ImageURLs []string
	// OpenNow is the status of each public branch, by branch id, at the time of the read.
	OpenNow map[int64]search.OpenStatus
}

// BusinessProfile serves the public page of a business.
type BusinessProfile struct {
	finder search.ProfileFinder
	urls   search.ImageURLs
	now    func() time.Time
}

// NewBusinessProfile builds the query. A nil clock uses time.Now.
func NewBusinessProfile(finder search.ProfileFinder, urls search.ImageURLs, now func() time.Time) *BusinessProfile {
	if now == nil {
		now = time.Now
	}
	return &BusinessProfile{finder: finder, urls: urls, now: now}
}

// Handle returns the profile of the business with that slug, or search.ErrBusinessNotFound.
// A photo whose URL cannot be made is left out: the page is worth more than the photo.
func (b *BusinessProfile) Handle(ctx context.Context, slug string) (ProfileView, error) {
	p, err := b.finder.Profile(ctx, slug)
	if err != nil {
		return ProfileView{}, err
	}
	view := ProfileView{
		Profile:   p,
		ImageURLs: make([]string, 0, len(p.ImageKeys)),
		OpenNow:   make(map[int64]search.OpenStatus, len(p.Branches)),
	}
	now := b.now()
	for _, branch := range p.Branches {
		view.OpenNow[branch.BranchID] = search.StatusAt(branch.Hours, now)
	}
	for _, key := range p.ImageKeys {
		url, err := b.urls.PresignDownload(ctx, key)
		if err != nil {
			continue
		}
		view.ImageURLs = append(view.ImageURLs, url)
	}
	return view, nil
}
