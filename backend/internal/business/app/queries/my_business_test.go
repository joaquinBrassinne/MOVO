package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type stubBusinesses struct {
	byOwner map[int64]domain.Business
}

func (s stubBusinesses) Create(context.Context, domain.Business) (domain.Business, error) {
	panic("not used by the query")
}

func (s stubBusinesses) FindByOwner(_ context.Context, ownerID int64) (domain.Business, error) {
	b, ok := s.byOwner[ownerID]
	if !ok {
		return domain.Business{}, business.ErrBusinessNotFound
	}
	return b, nil
}

func (s stubBusinesses) Save(context.Context, domain.Business) error {
	panic("not used by the query")
}

func (s stubBusinesses) SaveProfile(context.Context, domain.Business, business.AuditEntry) error {
	panic("not used by the query")
}

type stubBranches struct {
	branches []domain.Branch
	hours    map[int64][]domain.Slot
}

func (s stubBranches) Create(context.Context, domain.Branch) (domain.Branch, error) {
	panic("not used by the query")
}

func (s stubBranches) List(_ context.Context, businessID int64) ([]domain.Branch, error) {
	var out []domain.Branch
	for _, b := range s.branches {
		if b.BusinessID == businessID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (s stubBranches) Find(context.Context, int64, int64) (domain.Branch, error) {
	panic("not used by the query")
}

func (s stubBranches) Save(context.Context, domain.Branch) error { panic("not used by the query") }

func (s stubBranches) Delete(context.Context, int64, int64) error { panic("not used by the query") }

func (s stubBranches) Hours(_ context.Context, _, branchID int64) ([]domain.Slot, error) {
	return s.hours[branchID], nil
}

func (s stubBranches) ReplaceHours(context.Context, int64, int64, []domain.Slot) error {
	panic("not used by the query")
}

func TestGetMyBusinessReturnsOnlyTheOwnersData(t *testing.T) {
	t.Parallel()

	businesses := stubBusinesses{byOwner: map[int64]domain.Business{
		7: {ID: 1, OwnerID: 7, Name: "Mine"},
		8: {ID: 2, OwnerID: 8, Name: "Theirs"},
	}}
	branches := stubBranches{
		branches: []domain.Branch{{ID: 10, BusinessID: 1, Name: "Centro"}, {ID: 11, BusinessID: 2, Name: "Foreign"}},
		hours:    map[int64][]domain.Slot{10: {{Weekday: 1, Start: 540, End: 780}}},
	}
	query := queries.NewGetMyBusiness(businesses, branches, stubImages{}, stubURLs{})

	view, err := query.Handle(t.Context(), 7)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if view.Business.Name != "Mine" || len(view.Branches) != 1 {
		t.Fatalf("view = %+v, want only the owner's business and branch", view)
	}
	if got := view.Branches[0]; got.Branch.ID != 10 || len(got.Hours) != 1 {
		t.Errorf("branch view = %+v", got)
	}

	if _, err := query.Handle(t.Context(), 99); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("owner without business error = %v, want ErrBusinessNotFound", err)
	}
}

func TestGetMyBusinessWithoutBranchesReturnsAnEmptyList(t *testing.T) {
	t.Parallel()

	query := queries.NewGetMyBusiness(
		stubBusinesses{byOwner: map[int64]domain.Business{7: {ID: 1, OwnerID: 7}}},
		stubBranches{},
		stubImages{},
		stubURLs{},
	)
	view, err := query.Handle(t.Context(), 7)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if view.Branches == nil || len(view.Branches) != 0 {
		t.Errorf("branches = %#v, want a non-nil empty list so JSON renders []", view.Branches)
	}
}

type stubImages struct {
	images []domain.Image
}

func (s stubImages) Add(context.Context, int64, string, int) (domain.Image, error) {
	panic("not used by the query")
}

func (s stubImages) List(_ context.Context, businessID int64) ([]domain.Image, error) {
	var out []domain.Image
	for _, img := range s.images {
		if img.BusinessID == businessID {
			out = append(out, img)
		}
	}
	return out, nil
}

func (s stubImages) Remove(context.Context, int64, int64) (domain.Image, error) {
	panic("not used by the query")
}

func (s stubImages) Reorder(context.Context, int64, []int64) error { panic("not used by the query") }

type stubURLs struct{ err error }

func (s stubURLs) PresignDownload(_ context.Context, key string) (string, error) {
	return "https://files.test/" + key + "?sig=1", s.err
}

func TestGetMyBusinessIncludesThePhotosWithSignedUrlsInOrder(t *testing.T) {
	t.Parallel()

	businesses := stubBusinesses{byOwner: map[int64]domain.Business{7: {ID: 1, OwnerID: 7}, 8: {ID: 2, OwnerID: 8}}}
	images := stubImages{images: []domain.Image{
		{ID: 1, BusinessID: 1, ObjectKey: "businesses/1/a.jpg", Position: 0},
		{ID: 2, BusinessID: 1, ObjectKey: "businesses/1/b.jpg", Position: 1},
		{ID: 3, BusinessID: 2, ObjectKey: "businesses/2/foreign.jpg", Position: 0},
	}}

	view, err := queries.NewGetMyBusiness(businesses, stubBranches{}, images, stubURLs{}).Handle(t.Context(), 7)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(view.Images) != 2 || view.Images[0].Image.ID != 1 || view.Images[1].Image.Position != 1 {
		t.Fatalf("images = %+v, want only my two photos in order", view.Images)
	}
	if view.Images[0].URL != "https://files.test/businesses/1/a.jpg?sig=1" {
		t.Errorf("url = %q, want the signed URL of the object", view.Images[0].URL)
	}

	boom := errors.New("signer down")
	if _, err := queries.NewGetMyBusiness(businesses, stubBranches{}, images, stubURLs{err: boom}).Handle(t.Context(), 7); !errors.Is(err, boom) {
		t.Errorf("signing failure = %v, want it reported and not a photo without URL", err)
	}

	none, err := queries.NewGetMyBusiness(businesses, stubBranches{}, stubImages{}, stubURLs{}).Handle(t.Context(), 7)
	if err != nil || none.Images == nil || len(none.Images) != 0 {
		t.Errorf("no photos = %#v, %v; want a non-nil empty list", none.Images, err)
	}
}
