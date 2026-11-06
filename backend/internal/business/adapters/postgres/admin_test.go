package postgres_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

func newAdminUser(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ('root@example.com', 'hash', 'super_admin') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return id
}

func entry(actor, businessID int64, action string) business.AuditEntry {
	return business.AuditEntry{
		ActorID: actor, Action: action, Entity: "business", EntityID: businessID,
		Metadata: map[string]any{"status_from": "draft", "status_to": "pending_review"},
	}
}

func TestAdminApplyStoresTheNewStateAndTheAuditEntryTogether(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	owner := newOwner(t, pool, "owner@example.com")
	admin := newAdminUser(t, pool)
	created, err := postgres.NewBusinesses(pool).Create(t.Context(), draft(t, owner, categoryID(t, pool, "yoga"), "Studio"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	previous, err := repo.FindByID(t.Context(), created.ID)
	if err != nil || previous.Status != domain.StatusDraft {
		t.Fatalf("FindByID = %+v, %v", previous, err)
	}
	submitted, _ := previous.Submit()
	approved, _ := submitted.Approve()
	approved = approved.MarkVerified(admin)
	approved, _ = approved.SetPilot(true)

	if err := repo.Apply(t.Context(), approved, previous, entry(admin, created.ID, "business.approve")); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := repo.FindByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != domain.StatusActive || !got.Pilot || !got.Verified() || got.VerifiedBy != admin {
		t.Errorf("business = %+v, want active, pilot and verified by the admin", got)
	}

	var (
		actor    int64
		action   string
		raw      []byte
		rowCount int
	)
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log`).Scan(&rowCount); err != nil || rowCount != 1 {
		t.Fatalf("audit rows = %d, %v; want 1", rowCount, err)
	}
	if err := pool.QueryRow(t.Context(), `SELECT actor_id, action, metadata FROM audit_log`).Scan(&actor, &action, &raw); err != nil {
		t.Fatalf("read audit: %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if actor != admin || action != "business.approve" || metadata["status_to"] != "pending_review" {
		t.Errorf("audit = %d %q %v", actor, action, metadata)
	}
}

func TestAdminApplyRefusesAStaleVersionAndWritesNothing(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	admin := newAdminUser(t, pool)
	created, err := postgres.NewBusinesses(pool).Create(t.Context(),
		draft(t, newOwner(t, pool, "owner@example.com"), categoryID(t, pool, "yoga"), "Studio"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	stale, _ := repo.FindByID(t.Context(), created.ID)

	// Another admin marks the pilot flag first, on the same status.
	other, _ := stale.SetPilot(true)
	if err := repo.Apply(t.Context(), other, stale, entry(admin, created.ID, "business.mark_pilot")); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	verified := stale.MarkVerified(admin)
	if err := repo.Apply(t.Context(), verified, stale, entry(admin, created.ID, "business.verify")); !errors.Is(err, business.ErrStatusConflict) {
		t.Fatalf("stale Apply error = %v, want ErrStatusConflict", err)
	}

	got, _ := repo.FindByID(t.Context(), created.ID)
	if !got.Pilot || got.Verified() {
		t.Errorf("business = %+v, the first change must survive and the stale one must not apply", got)
	}
	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_log`).Scan(&n); err != nil || n != 1 {
		t.Errorf("audit rows = %d, %v; a refused change must not be logged", n, err)
	}
}

func TestAdminApplyRollsBackTheChangeWhenTheAuditEntryFails(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	created, err := postgres.NewBusinesses(pool).Create(t.Context(),
		draft(t, newOwner(t, pool, "owner@example.com"), categoryID(t, pool, "yoga"), "Studio"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	previous, _ := repo.FindByID(t.Context(), created.ID)
	submitted, _ := previous.Submit()

	if err := repo.Apply(t.Context(), submitted, previous, entry(999999, created.ID, "business.submit")); err == nil {
		t.Fatal("Apply accepted an audit entry of an unknown actor")
	}
	if got, _ := repo.FindByID(t.Context(), created.ID); got.Status != domain.StatusDraft {
		t.Errorf("status = %q, the change must roll back with the failed audit entry", got.Status)
	}
}

func TestAdminFindByIDAndApplyOnAMissingBusiness(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	admin := newAdminUser(t, pool)

	if _, err := repo.FindByID(t.Context(), 424242); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("FindByID error = %v, want ErrBusinessNotFound", err)
	}
	ghost := domain.Business{ID: 424242, Status: domain.StatusActive}
	if err := repo.Apply(t.Context(), ghost, domain.Business{ID: 424242, Status: domain.StatusPendingReview}, entry(admin, 424242, "business.approve")); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("Apply error = %v, want ErrBusinessNotFound", err)
	}
}

func TestAdminListOrdersTheReviewQueueFirstAndFilters(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	businesses := postgres.NewBusinesses(pool)
	branches := postgres.NewBranches(pool)
	yoga := categoryID(t, pool, "yoga")

	create := func(email, name string, status domain.Status) domain.Business {
		b, err := businesses.Create(t.Context(), draft(t, newOwner(t, pool, email), yoga, name))
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		b.Status = status
		if err := businesses.Save(t.Context(), b); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
		return b
	}
	active := create("a@example.com", "Active One", domain.StatusActive)
	create("p@example.com", "Pending One", domain.StatusPendingReview)
	create("d@example.com", "Deleted One", domain.StatusDeleted)
	if _, err := branches.Create(t.Context(), newBranch(t, active.ID, "Centro")); err != nil {
		t.Fatalf("create branch: %v", err)
	}

	page, err := repo.List(t.Context(), business.AdminFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Total != 2 || len(page.Rows) != 2 || page.Rows[0].Business.Name != "Pending One" || page.Rows[1].Business.Name != "Active One" {
		t.Fatalf("page = %+v, want the review queue first and the deleted one hidden", page)
	}
	if row := page.Rows[1]; row.OwnerEmail != "a@example.com" || row.CategoryName != "Yoga" || row.BranchCount != 1 || page.Rows[0].BranchCount != 0 {
		t.Errorf("row = %+v", row)
	}

	deleted, err := repo.List(t.Context(), business.AdminFilter{Status: domain.StatusDeleted, Limit: 10})
	if err != nil || deleted.Total != 1 || deleted.Rows[0].Business.Name != "Deleted One" {
		t.Errorf("deleted filter = %+v, %v", deleted, err)
	}

	paged, err := repo.List(t.Context(), business.AdminFilter{Limit: 1, Offset: 1})
	if err != nil || paged.Total != 2 || len(paged.Rows) != 1 || paged.Rows[0].Business.Name != "Active One" {
		t.Errorf("second page = %+v, %v; want the total to ignore paging", paged, err)
	}
}

// seedDirectory creates four businesses with different status, category, owner and
// branch address, and returns their ids by name.
func seedDirectory(t *testing.T, pool *pgxpool.Pool) map[string]int64 {
	t.Helper()

	businesses := postgres.NewBusinesses(pool)
	branches := postgres.NewBranches(pool)
	ids := map[string]int64{}

	for _, c := range []struct {
		name, email, category, address string
		status                         domain.Status
	}{
		{"Alma Haus", "ana@alma.example", "gimnasios", "Gorriti 5520", domain.StatusActive},
		{"Flow Yoga", "bruno@flow.example", "yoga", "Vicente Lopez 1822", domain.StatusPendingReview},
		{"Box 100%", "carla@box.example", "gimnasios", "Cabildo 2140", domain.StatusPaused},
		{"Cerrado", "dario@closed.example", "yoga", "Callao 10", domain.StatusDeleted},
	} {
		b, err := businesses.Create(t.Context(), draft(t, newOwner(t, pool, c.email), categoryID(t, pool, c.category), c.name))
		if err != nil {
			t.Fatalf("create %s: %v", c.name, err)
		}
		b.Status = c.status
		if err := businesses.Save(t.Context(), b); err != nil {
			t.Fatalf("save %s: %v", c.name, err)
		}
		branch := newBranch(t, b.ID, "Sede")
		branch.Address = c.address
		if _, err := branches.Create(t.Context(), branch); err != nil {
			t.Fatalf("branch of %s: %v", c.name, err)
		}
		ids[c.name] = b.ID
	}
	return ids
}

func names(page business.AdminPage) []string {
	out := []string{}
	for _, row := range page.Rows {
		out = append(out, row.Business.Name)
	}
	return out
}

func TestAdminListSearchesFiltersAndCounts(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	ids := seedDirectory(t, pool)

	search := func(q string) []string {
		page, err := repo.List(t.Context(), business.AdminFilter{Query: q, Limit: 10})
		if err != nil {
			t.Fatalf("List(%q): %v", q, err)
		}
		return names(page)
	}

	if got := search("alma"); len(got) != 1 || got[0] != "Alma Haus" {
		t.Errorf("by name (any case) = %v", got)
	}
	if got := search("BRUNO@flow"); len(got) != 1 || got[0] != "Flow Yoga" {
		t.Errorf("by owner email = %v", got)
	}
	if got := search("cabildo"); len(got) != 1 || got[0] != "Box 100%" {
		t.Errorf("by branch address = %v", got)
	}
	// A number is matched as an id and as text (it may be a street number), so the
	// business with that id must be among the results.
	if got := search(itoa(ids["Flow Yoga"])); !slices.Contains(got, "Flow Yoga") {
		t.Errorf("by exact id = %v, want it to include Flow Yoga", got)
	}
	if got := search("%"); len(got) != 1 || got[0] != "Box 100%" {
		t.Errorf("a literal %% must match only the name that has one, got %v", got)
	}
	if got := search("_"); len(got) != 0 {
		t.Errorf("a literal underscore matched %v: wildcards must be escaped", got)
	}
	if got := search("   "); len(got) != 3 {
		t.Errorf("a blank search = %v, want the unfiltered list without the deleted one", got)
	}

	yoga, err := repo.List(t.Context(), business.AdminFilter{CategoryID: categoryID(t, pool, "yoga"), Limit: 10})
	if err != nil || len(yoga.Rows) != 1 || yoga.Rows[0].Business.Name != "Flow Yoga" || yoga.Total != 1 {
		t.Errorf("category filter = %v, %v; want only the non-deleted yoga business", names(yoga), err)
	}

	page, err := repo.List(t.Context(), business.AdminFilter{Status: domain.StatusActive, Query: "flow", Limit: 10})
	if err != nil || len(page.Rows) != 0 {
		t.Errorf("status and search together = %v, %v; want none", names(page), err)
	}
	// The tab counters describe the whole directory, whatever the filters say.
	want := map[domain.Status]int{domain.StatusActive: 1, domain.StatusPendingReview: 1, domain.StatusPaused: 1}
	for status, n := range want {
		if page.Counts[status] != n {
			t.Errorf("count of %s = %d, want %d", status, page.Counts[status], n)
		}
	}
	if page.Counts[domain.StatusDeleted] != 1 || page.Counts[domain.StatusDraft] != 0 {
		t.Errorf("counts = %v, want the deleted one counted apart and no drafts", page.Counts)
	}
}

func TestAdminFindAndHistory(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	repo := postgres.NewAdmin(pool)
	ids := seedDirectory(t, pool)
	admin := newAdminUser(t, pool)

	row, err := repo.Find(t.Context(), ids["Alma Haus"])
	if err != nil || row.OwnerEmail != "ana@alma.example" || row.CategoryName != "Gimnasios" || row.BranchCount != 1 {
		t.Fatalf("Find = %+v, %v", row, err)
	}
	if _, err := repo.Find(t.Context(), 424242); !errors.Is(err, business.ErrBusinessNotFound) {
		t.Errorf("missing business error = %v, want ErrBusinessNotFound", err)
	}

	previous, _ := repo.FindByID(t.Context(), ids["Alma Haus"])
	verified := previous.MarkVerified(admin)
	if err := repo.Apply(t.Context(), verified, previous, entry(admin, previous.ID, "business.verify")); err != nil {
		t.Fatalf("verify: %v", err)
	}
	unmarked, _ := verified.SetPilot(true)
	if err := repo.Apply(t.Context(), unmarked, verified, entry(admin, previous.ID, "business.mark_pilot")); err != nil {
		t.Fatalf("pilot: %v", err)
	}
	other, _ := repo.FindByID(t.Context(), ids["Flow Yoga"])
	if err := repo.Apply(t.Context(), other.MarkVerified(admin), other, entry(admin, other.ID, "business.verify")); err != nil {
		t.Fatalf("other: %v", err)
	}

	history, err := repo.History(t.Context(), previous.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("History = %+v, %v; want only the two entries of this business", history, err)
	}
	if history[0].Action != "business.mark_pilot" || history[1].Action != "business.verify" {
		t.Errorf("history = %v, %v; want newest first", history[0].Action, history[1].Action)
	}
	if history[0].ActorEmail != "root@example.com" || history[0].At.IsZero() || history[0].Metadata["status_to"] != "pending_review" {
		t.Errorf("record = %+v", history[0])
	}
	if empty, err := repo.History(t.Context(), ids["Box 100%"]); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("history without entries = %#v, %v; want a non-nil empty list", empty, err)
	}
}
