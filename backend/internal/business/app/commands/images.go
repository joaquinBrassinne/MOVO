package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// imageExtensions maps the accepted content types to the object key extension.
var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// PresignImageUploadInput asks for a URL to upload one image of the owner's business.
type PresignImageUploadInput struct {
	OwnerID     int64
	ContentType string
	Size        int64
}

// PresignedImage is the answer: where to upload and the key to remember.
type PresignedImage struct {
	ObjectKey string
	Upload    storage.PresignedUpload
}

// PresignImageUpload creates the short-lived URL a client uploads an image to.
type PresignImageUpload struct {
	businesses business.BusinessRepository
	storage    business.ImageStorage
	newID      func() string
}

// NewPresignImageUpload wires the use case. A nil newID uses random 128-bit ids.
func NewPresignImageUpload(businesses business.BusinessRepository, images business.ImageStorage, newID func() string) *PresignImageUpload {
	if newID == nil {
		newID = randomID
	}
	return &PresignImageUpload{businesses: businesses, storage: images, newID: newID}
}

// Handle validates the type and size, builds a key under the caller's business
// (the client never chooses it) and presigns the upload. Validation failures are
// the storage errors; the file bytes never pass through the API.
func (c *PresignImageUpload) Handle(ctx context.Context, in PresignImageUploadInput) (PresignedImage, error) {
	owned, err := editableBusiness(ctx, c.businesses, in.OwnerID)
	if err != nil {
		return PresignedImage{}, err
	}

	key := "businesses/" + strconv.FormatInt(owned.ID, 10) + "/" + c.newID() + imageExtensions[in.ContentType]
	upload, err := c.storage.PresignUpload(ctx, storage.UploadRequest{Key: key, ContentType: in.ContentType, Size: in.Size})
	if err != nil {
		return PresignedImage{}, fmt.Errorf("presign upload: %w", err)
	}
	return PresignedImage{ObjectKey: key, Upload: upload}, nil
}

func randomID() string {
	b := make([]byte, 16)
	// crypto/rand.Read never returns an error on supported platforms (Go 1.24+).
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
