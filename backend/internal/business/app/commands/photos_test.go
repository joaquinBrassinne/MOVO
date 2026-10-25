package commands_test

import (
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// uploaded simulates the client uploading to a presigned URL: the object exists.
// maxPhotos is the limit these tests configure: small, so the limit is reached quickly.
const maxPhotos = 3

func uploaded(mem *storage.Memory, businessID int64, name string) string {
	key := "businesses/" + itoa(businessID) + "/" + name + ".jpg"
	mem.Put(key)
	return key
}

func TestRegisterImagePutsThePhotoAtTheEndOfTheList(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem, images := storage.NewMemory(), &fakeImages{}
	register := commands.NewRegisterImage(businesses, images, mem, maxPhotos)

	for i, name := range []string{"a", "b", "c"} {
		got, err := register.Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, mine.ID, name)})
		if err != nil || got.Position != i || got.BusinessID != mine.ID {
			t.Fatalf("photo %q = %+v, %v; want position %d", name, got, err, i)
		}
	}
}

func TestRegisterImageRejections(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	theirs, _ := businesses.FindByOwner(t.Context(), 8)
	mem, images := storage.NewMemory(), &fakeImages{}
	register := commands.NewRegisterImage(businesses, images, mem, maxPhotos)

	registered := uploaded(mem, mine.ID, "first")
	if _, err := register.Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: registered}); err != nil {
		t.Fatalf("first registration: %v", err)
	}

	tests := []struct {
		name  string
		input commands.RegisterImageInput
		want  error
	}{
		{"owner without business", commands.RegisterImageInput{OwnerID: 99, ObjectKey: registered}, business.ErrBusinessNotFound},
		{"key of another business", commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, theirs.ID, "stolen")}, domain.ErrImageKeyInvalid},
		{"never uploaded", commands.RegisterImageInput{OwnerID: 7, ObjectKey: "businesses/" + itoa(mine.ID) + "/ghost.jpg"}, business.ErrImageNotUploaded},
		{"registered twice", commands.RegisterImageInput{OwnerID: 7, ObjectKey: registered}, business.ErrImageExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := register.Handle(t.Context(), tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
	if got := len(images.forBusiness(mine.ID)); got != 1 {
		t.Errorf("registered photos = %d, rejected ones must not be stored", got)
	}
}

func TestRegisterImageStopsAtTheLimit(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem, images := storage.NewMemory(), &fakeImages{}
	register := commands.NewRegisterImage(businesses, images, mem, maxPhotos)

	for i := range maxPhotos {
		if _, err := register.Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, mine.ID, "p"+itoa(int64(i)))}); err != nil {
			t.Fatalf("photo %d: %v", i+1, err)
		}
	}
	_, err := register.Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, mine.ID, "one-too-many")})
	if !errors.Is(err, business.ErrImageLimitReached) {
		t.Fatalf("photo over the limit error = %v, want ErrImageLimitReached", err)
	}
}

func TestRemoveImageClosesTheGapAndDeletesTheObject(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem, cleanup := storage.NewMemory(), &fakeCleanup{}
	images := &fakeImages{cleanup: cleanup}
	register := commands.NewRegisterImage(businesses, images, mem, maxPhotos)
	var ids []int64
	var keys []string
	for _, name := range []string{"a", "b", "c"} {
		key := uploaded(mem, mine.ID, name)
		img, err := register.Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: key})
		if err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
		ids, keys = append(ids, img.ID), append(keys, key)
	}

	remove := commands.NewRemoveImage(businesses, images, mem, cleanup, nil)
	if err := remove.Handle(t.Context(), commands.RemoveImageInput{OwnerID: 7, ImageID: ids[0]}); err != nil {
		t.Fatalf("remove the cover: %v", err)
	}

	left := images.forBusiness(mine.ID)
	if len(left) != 2 || left[0].ID != ids[1] || left[0].Position != 0 || left[1].Position != 1 {
		t.Errorf("remaining = %+v, want the second photo promoted to cover with no gap", left)
	}
	if deleted := mem.Deleted(); len(deleted) != 1 || deleted[0] != keys[0] {
		t.Errorf("deleted objects = %v, want %q", deleted, keys[0])
	}
	if err := remove.Handle(t.Context(), commands.RemoveImageInput{OwnerID: 7, ImageID: ids[0]}); !errors.Is(err, business.ErrImageNotFound) {
		t.Errorf("second removal error = %v, want ErrImageNotFound", err)
	}
}

func TestOneOwnerCannotRemoveOrReorderThePhotosOfAnother(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem, cleanup := storage.NewMemory(), &fakeCleanup{}
	images := &fakeImages{cleanup: cleanup}
	img, err := commands.NewRegisterImage(businesses, images, mem, maxPhotos).Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, mine.ID, "a")})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := commands.NewRemoveImage(businesses, images, mem, cleanup, nil).Handle(t.Context(), commands.RemoveImageInput{OwnerID: 8, ImageID: img.ID}); !errors.Is(err, business.ErrImageNotFound) {
		t.Errorf("foreign removal error = %v, want ErrImageNotFound", err)
	}
	if err := commands.NewReorderImages(businesses, images).Handle(t.Context(), commands.ReorderImagesInput{OwnerID: 8, IDs: []int64{img.ID}}); !errors.Is(err, business.ErrImageOrderInvalid) {
		t.Errorf("foreign reorder error = %v, want ErrImageOrderInvalid", err)
	}
	if len(images.forBusiness(mine.ID)) != 1 || len(mem.Deleted()) != 0 {
		t.Error("a foreign attempt must change nothing")
	}
}

func TestReorderImagesChoosesTheCover(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem, images := storage.NewMemory(), &fakeImages{}
	register := commands.NewRegisterImage(businesses, images, mem, maxPhotos)
	var ids []int64
	for _, name := range []string{"a", "b", "c"} {
		img, _ := register.Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, mine.ID, name)})
		ids = append(ids, img.ID)
	}

	reorder := commands.NewReorderImages(businesses, images)
	if err := reorder.Handle(t.Context(), commands.ReorderImagesInput{OwnerID: 7, IDs: []int64{ids[2], ids[0], ids[1]}}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	for _, img := range images.forBusiness(mine.ID) {
		want := map[int64]int{ids[2]: 0, ids[0]: 1, ids[1]: 2}[img.ID]
		if img.Position != want {
			t.Errorf("photo %d position = %d, want %d", img.ID, img.Position, want)
		}
	}

	for name, order := range map[string][]int64{
		"missing one": {ids[0], ids[1]},
		"duplicate":   {ids[0], ids[0], ids[1]},
		"unknown id":  {ids[0], ids[1], 9999},
	} {
		if err := reorder.Handle(t.Context(), commands.ReorderImagesInput{OwnerID: 7, IDs: order}); !errors.Is(err, business.ErrImageOrderInvalid) {
			t.Errorf("%s error = %v, want ErrImageOrderInvalid", name, err)
		}
	}
}

func TestRemoveImageDropsTheQueuedDeleteOnceTheObjectIsGone(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem, cleanup := storage.NewMemory(), &fakeCleanup{}
	images := &fakeImages{cleanup: cleanup}
	img, err := commands.NewRegisterImage(businesses, images, mem, maxPhotos).
		Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: uploaded(mem, mine.ID, "a")})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := commands.NewRemoveImage(businesses, images, mem, cleanup, nil).Handle(t.Context(), commands.RemoveImageInput{OwnerID: 7, ImageID: img.ID}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if pending, _ := cleanup.Pending(t.Context(), 10); len(pending) != 0 {
		t.Errorf("queued deletes = %v, want none once the object is deleted", pending)
	}
	if deleted := mem.Deleted(); len(deleted) != 1 || deleted[0] != img.ObjectKey {
		t.Errorf("deleted objects = %v, want %q", deleted, img.ObjectKey)
	}
}

func TestRemoveImageWhenTheStorageFailsKeepsTheDeleteQueuedForRetry(t *testing.T) {
	t.Parallel()

	businesses, _ := twoOwners(t)
	mine, _ := businesses.FindByOwner(t.Context(), 7)
	mem := &flakyStorage{Memory: storage.NewMemory(), down: true}
	cleanup := &fakeCleanup{}
	images := &fakeImages{cleanup: cleanup}
	key := uploaded(mem.Memory, mine.ID, "a")
	img, err := commands.NewRegisterImage(businesses, images, mem, maxPhotos).
		Handle(t.Context(), commands.RegisterImageInput{OwnerID: 7, ObjectKey: key})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// The photo is gone for the owner even though its object could not be deleted.
	if err := commands.NewRemoveImage(businesses, images, mem, cleanup, nil).Handle(t.Context(), commands.RemoveImageInput{OwnerID: 7, ImageID: img.ID}); err != nil {
		t.Fatalf("remove with the storage down: %v", err)
	}
	if left := images.forBusiness(mine.ID); len(left) != 0 {
		t.Errorf("photos = %+v, the row must not stay when the object fails to delete", left)
	}
	if cleanup.attempts[key] != 1 || cleanup.reasons[key] != "storage unavailable" {
		t.Fatalf("queue = %v / %v, want the key with one failed attempt and its reason", cleanup.attempts, cleanup.reasons)
	}

	purge := commands.NewPurgeObjects(mem, cleanup, nil)
	if deleted, failed, err := purge.Handle(t.Context(), 10); err != nil || deleted != 0 || failed != 1 {
		t.Fatalf("purge with the storage still down = %d deleted, %d failed, %v; want 0, 1, nil", deleted, failed, err)
	}
	if cleanup.attempts[key] != 2 {
		t.Errorf("attempts = %d, want 2 after the retry", cleanup.attempts[key])
	}

	mem.down = false
	if deleted, failed, err := purge.Handle(t.Context(), 10); err != nil || deleted != 1 || failed != 0 {
		t.Fatalf("purge once the storage is back = %d deleted, %d failed, %v; want 1, 0, nil", deleted, failed, err)
	}
	if pending, _ := cleanup.Pending(t.Context(), 10); len(pending) != 0 {
		t.Errorf("queue = %v, want it empty", pending)
	}
	if deleted := mem.Deleted(); len(deleted) != 1 || deleted[0] != key {
		t.Errorf("deleted objects = %v, want %q", deleted, key)
	}
}
