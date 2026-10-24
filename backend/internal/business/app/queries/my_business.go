// Package queries holds the read use cases of the business module.
package queries

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// BranchView is a branch with its weekly opening hours.
type BranchView struct {
	Branch domain.Branch
	Hours  []domain.Slot
}

// ImageView is a photo with the short-lived URL that shows it.
type ImageView struct {
	Image domain.Image
	URL   string
}

// MyBusinessView is everything the owner's panel shows about the business.
type MyBusinessView struct {
	Business domain.Business
	Branches []BranchView
	Images   []ImageView
}

// GetMyBusiness reads the owner's business with its branches, hours and photos.
type GetMyBusiness struct {
	businesses business.BusinessRepository
	branches   business.BranchRepository
	images     business.ImageRepository
	urls       business.ImageURLs
}

// NewGetMyBusiness wires the GetMyBusiness query.
func NewGetMyBusiness(
	businesses business.BusinessRepository,
	branches business.BranchRepository,
	images business.ImageRepository,
	urls business.ImageURLs,
) *GetMyBusiness {
	return &GetMyBusiness{businesses: businesses, branches: branches, images: images, urls: urls}
}

// Handle returns business.ErrBusinessNotFound when the owner has not created the
// business yet, which the onboarding view uses to know it must start from scratch.
func (q *GetMyBusiness) Handle(ctx context.Context, ownerID int64) (MyBusinessView, error) {
	b, err := q.businesses.FindByOwner(ctx, ownerID)
	if err != nil {
		return MyBusinessView{}, fmt.Errorf("find business: %w", err)
	}

	branches, err := q.branches.List(ctx, b.ID)
	if err != nil {
		return MyBusinessView{}, fmt.Errorf("list branches: %w", err)
	}

	view := MyBusinessView{Business: b, Branches: make([]BranchView, 0, len(branches))}
	for _, branch := range branches {
		hours, err := q.branches.Hours(ctx, b.ID, branch.ID)
		if err != nil {
			return MyBusinessView{}, fmt.Errorf("read hours of branch %d: %w", branch.ID, err)
		}
		view.Branches = append(view.Branches, BranchView{Branch: branch, Hours: hours})
	}

	images, err := q.images.List(ctx, b.ID)
	if err != nil {
		return MyBusinessView{}, fmt.Errorf("list images: %w", err)
	}
	view.Images = make([]ImageView, 0, len(images))
	for _, img := range images {
		url, err := q.urls.PresignDownload(ctx, img.ObjectKey)
		if err != nil {
			return MyBusinessView{}, fmt.Errorf("sign url of image %d: %w", img.ID, err)
		}
		view.Images = append(view.Images, ImageView{Image: img, URL: url})
	}
	return view, nil
}
