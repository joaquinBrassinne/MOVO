package commands

import (
	"context"
	"fmt"
	"log/slog"

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
	max        int
}

// NewRegisterImage wires the RegisterImage use case. max is how many photos a business
// may register (MAX_BUSINESS_IMAGES).
func NewRegisterImage(businesses business.BusinessRepository, images business.ImageRepository, storage business.ImageStorage, max int) *RegisterImage {
	return &RegisterImage{businesses: businesses, images: images, storage: storage, max: max}
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

	image, err := c.images.Add(ctx, owned.ID, in.ObjectKey, c.max)
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
	cleanup    business.ObjectCleanup
	log        *slog.Logger
}

// NewRemoveImage wires the RemoveImage use case. A nil logger discards logs.
func NewRemoveImage(businesses business.BusinessRepository, images business.ImageRepository, storage business.ImageStorage, cleanup business.ObjectCleanup, log *slog.Logger) *RemoveImage {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &RemoveImage{businesses: businesses, images: images, storage: storage, cleanup: cleanup, log: log}
}

// Handle removes the registration, which is what the business shows, and then the
// object. The repository queues the object's key in the same transaction as the
// removal, so the photo is gone for good and the delete is never forgotten: if the
// storage fails now, the key stays queued with the reason and PurgeObjects retries it.
func (c *RemoveImage) Handle(ctx context.Context, in RemoveImageInput) error {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return err
	}

	removed, err := c.images.Remove(ctx, owned.ID, in.ImageID)
	if err != nil {
		return fmt.Errorf("remove image: %w", err)
	}

	if err := deleteQueued(ctx, c.storage, c.cleanup, removed.ObjectKey); err != nil {
		c.log.Warn("photo object not deleted, queued for retry", "key", removed.ObjectKey, "error", err)
	}
	return nil
}

// deleteQueued deletes one queued object and updates its queue entry: dropped on success,
// marked as failed (and kept) otherwise. It returns the storage error, if any.
func deleteQueued(ctx context.Context, storage business.ImageStorage, cleanup business.ObjectCleanup, key string) error {
	if err := storage.Delete(ctx, key); err != nil {
		// The attempt is recorded best effort: the key is queued either way.
		_ = cleanup.Failed(ctx, key, err.Error())
		return err
	}
	if err := cleanup.Done(ctx, key); err != nil {
		return fmt.Errorf("drop queued object: %w", err)
	}
	return nil
}

// PurgeObjects retries the deletes that failed: the objects of removed photos that are
// still queued. It is meant to run on a schedule (the worker, once it exists).
type PurgeObjects struct {
	storage business.ImageStorage
	cleanup business.ObjectCleanup
	log     *slog.Logger
}

// NewPurgeObjects wires the PurgeObjects use case. A nil logger discards logs.
func NewPurgeObjects(storage business.ImageStorage, cleanup business.ObjectCleanup, log *slog.Logger) *PurgeObjects {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &PurgeObjects{storage: storage, cleanup: cleanup, log: log}
}

// Handle tries up to limit queued deletes and returns how many worked and how many
// failed again. A failure keeps the key queued; one failing object never stops the rest.
func (p *PurgeObjects) Handle(ctx context.Context, limit int) (deleted, failed int, err error) {
	keys, err := p.cleanup.Pending(ctx, limit)
	if err != nil {
		return 0, 0, fmt.Errorf("list queued objects: %w", err)
	}
	for _, key := range keys {
		if err := deleteQueued(ctx, p.storage, p.cleanup, key); err != nil {
			p.log.Warn("queued photo object still not deleted", "key", key, "error", err)
			failed++
			continue
		}
		deleted++
	}
	return deleted, failed, nil
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
