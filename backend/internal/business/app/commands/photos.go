package commands

import (
	"context"
	"fmt"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

// RegisterImageInput confirms that an image was uploaded to the key a presign gave.
type RegisterImageInput struct {
	OwnerID   int64
	ObjectKey string
}

// RegisterImage adds an uploaded photo to the business of the caller.
type RegisterImage struct {
	businesses business.BusinessRepository
	images     business.ImageRepository
	storage    business.ImageStorage
}

// NewRegisterImage wires the RegisterImage use case.
func NewRegisterImage(businesses business.BusinessRepository, images business.ImageRepository, storage business.ImageStorage) *RegisterImage {
	return &RegisterImage{businesses: businesses, images: images, storage: storage}
}

// Handle checks that the key is inside the business folder and that the object is
// really there before registering it at the end of the list. It returns
// domain.ErrImageKeyInvalid, business.ErrImageNotUploaded, business.ErrImageExists
// and business.ErrImageLimitReached.
func (c *RegisterImage) Handle(ctx context.Context, in RegisterImageInput) (domain.Image, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return domain.Image{}, err
	}
	if err := domain.ValidateImageKey(owned.ID, in.ObjectKey); err != nil {
		return domain.Image{}, err
	}

	uploaded, err := c.storage.Exists(ctx, in.ObjectKey)
	if err != nil {
		return domain.Image{}, fmt.Errorf("check upload: %w", err)
	}
	if !uploaded {
		return domain.Image{}, business.ErrImageNotUploaded
	}

	image, err := c.images.Add(ctx, owned.ID, in.ObjectKey, domain.MaxImages)
	if err != nil {
		return domain.Image{}, fmt.Errorf("add image: %w", err)
	}
	return image, nil
}

// RemoveImageInput identifies the photo to remove.
type RemoveImageInput struct {
	OwnerID int64
	ImageID int64
}

// RemoveImage deletes a photo of the business of the caller.
type RemoveImage struct {
	businesses business.BusinessRepository
	images     business.ImageRepository
	storage    business.ImageStorage
}

// NewRemoveImage wires the RemoveImage use case.
func NewRemoveImage(businesses business.BusinessRepository, images business.ImageRepository, storage business.ImageStorage) *RemoveImage {
	return &RemoveImage{businesses: businesses, images: images, storage: storage}
}

// Handle removes the registration and then the object. The registration is what
// the business shows, so it goes first; an object that fails to delete is an
// orphan that costs nothing but space and a cleanup can collect later.
func (c *RemoveImage) Handle(ctx context.Context, in RemoveImageInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}

	removed, err := c.images.Remove(ctx, owned.ID, in.ImageID)
	if err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	_ = c.storage.Delete(ctx, removed.ObjectKey)
	return nil
}

// ReorderImagesInput lists the photos in their new order; the first is the cover.
type ReorderImagesInput struct {
	OwnerID int64
	IDs     []int64
}

// ReorderImages changes the order of the photos, which is how the cover is chosen.
type ReorderImages struct {
	businesses business.BusinessRepository
	images     business.ImageRepository
}

// NewReorderImages wires the ReorderImages use case.
func NewReorderImages(businesses business.BusinessRepository, images business.ImageRepository) *ReorderImages {
	return &ReorderImages{businesses: businesses, images: images}
}

// Handle applies the order. The ids must be exactly the photos of the business,
// each once; anything else is business.ErrImageOrderInvalid.
func (c *ReorderImages) Handle(ctx context.Context, in ReorderImagesInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}
	if err := c.images.Reorder(ctx, owned.ID, in.IDs); err != nil {
		return fmt.Errorf("reorder images: %w", err)
	}
	return nil
}
