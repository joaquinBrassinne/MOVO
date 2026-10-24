package postgres_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

var now = time.Now().UTC().Truncate(time.Second)

func digest(seed string) []byte { return domain.HashRefreshToken(seed) }

func newUser(t *testing.T, pool *pgxpool.Pool, email string) domain.User {
	t.Helper()

	user, err := domain.NewUser(email, "correct horse battery", domain.RoleBusinessAdmin)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}

	stored, err := postgres.NewUsers(pool).Create(t.Context(), user)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return stored
}

func TestUsersCreateAndFind(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	users := postgres.NewUsers(pool)

	created := newUser(t, pool, "owner@example.com")
	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("Create did not assign ID and CreatedAt: %+v", created)
	}

	byEmail, err := users.FindByEmail(t.Context(), "owner@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	byID, err := users.FindByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	for name, got := range map[string]domain.User{"by email": byEmail, "by id": byID} {
		if got.ID != created.ID || got.Email != "owner@example.com" || got.Role != domain.RoleBusinessAdmin {
			t.Errorf("%s = %+v, want the stored owner", name, got)
		}
		if !got.CheckPassword("correct horse battery") {
			t.Errorf("%s: stored hash does not verify the password", name)
		}
	}
}

func TestUsersReportDuplicatesAndMisses(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	users := postgres.NewUsers(pool)
	existing := newUser(t, pool, "owner@example.com")

	duplicate := domain.User{Email: existing.Email, PasswordHash: "hash", Role: domain.RoleBusinessAdmin}
	if _, err := users.Create(t.Context(), duplicate); !errors.Is(err, identity.ErrEmailTaken) {
		t.Errorf("duplicate Create error = %v, want %v", err, identity.ErrEmailTaken)
	}
	if _, err := users.FindByEmail(t.Context(), "ghost@example.com"); !errors.Is(err, identity.ErrUserNotFound) {
		t.Errorf("FindByEmail error = %v, want %v", err, identity.ErrUserNotFound)
	}
	if _, err := users.FindByID(t.Context(), 999999); !errors.Is(err, identity.ErrUserNotFound) {
		t.Errorf("FindByID error = %v, want %v", err, identity.ErrUserNotFound)
	}
}

func TestRotateReplacesTheTokenOnce(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	store := postgres.NewRefreshTokens(pool)
	user := newUser(t, pool, "owner@example.com")

	if err := store.Issue(t.Context(), user.ID, digest("t1"), now.Add(time.Hour)); err != nil {
		t.Fatalf("Issue: %v", err)
	}

	owner, err := store.Rotate(t.Context(), digest("t1"), digest("t2"), now.Add(time.Hour), now)
	if err != nil || owner != user.ID {
		t.Fatalf("Rotate = (%d, %v), want (%d, nil)", owner, err, user.ID)
	}

	if _, err := store.Rotate(t.Context(), digest("t2"), digest("t3"), now.Add(time.Hour), now); err != nil {
		t.Errorf("rotating the successor failed: %v", err)
	}
}

func TestRotateRejectsUnknownExpiredAndLoggedOutTokens(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	store := postgres.NewRefreshTokens(pool)
	user := newUser(t, pool, "owner@example.com")

	if err := store.Issue(t.Context(), user.ID, digest("expired"), now.Add(-time.Minute)); err != nil {
		t.Fatalf("Issue expired: %v", err)
	}
	if err := store.Issue(t.Context(), user.ID, digest("loggedout"), now.Add(time.Hour)); err != nil {
		t.Fatalf("Issue loggedout: %v", err)
	}
	if err := store.Revoke(t.Context(), digest("loggedout"), now); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := store.Issue(t.Context(), user.ID, digest("bystander"), now.Add(time.Hour)); err != nil {
		t.Fatalf("Issue bystander: %v", err)
	}

	for _, seed := range []string{"unknown", "expired", "loggedout"} {
		_, err := store.Rotate(t.Context(), digest(seed), digest("next-"+seed), now.Add(time.Hour), now)
		if !errors.Is(err, identity.ErrRefreshTokenInvalid) {
			t.Errorf("Rotate(%s) error = %v, want %v", seed, err, identity.ErrRefreshTokenInvalid)
		}
	}

	// None of those is a replay, so the user's other session must survive.
	if _, err := store.Rotate(t.Context(), digest("bystander"), digest("b2"), now.Add(time.Hour), now); err != nil {
		t.Errorf("bystander session was revoked by an unrelated rejection: %v", err)
	}
}

func TestRotateReplayRevokesTheWholeFamilyOfThatUserOnly(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	store := postgres.NewRefreshTokens(pool)
	victim := newUser(t, pool, "victim@example.com")
	other := newUser(t, pool, "other@example.com")

	for seed, id := range map[string]int64{"v1": victim.ID, "o1": other.ID} {
		if err := store.Issue(t.Context(), id, digest(seed), now.Add(time.Hour)); err != nil {
			t.Fatalf("Issue %s: %v", seed, err)
		}
	}
	if _, err := store.Rotate(t.Context(), digest("v1"), digest("v2"), now.Add(time.Hour), now); err != nil {
		t.Fatalf("first rotation: %v", err)
	}

	owner, err := store.Rotate(t.Context(), digest("v1"), digest("v3"), now.Add(time.Hour), now)
	if !errors.Is(err, identity.ErrRefreshTokenReused) || owner != victim.ID {
		t.Fatalf("replay = (%d, %v), want (%d, %v)", owner, err, victim.ID, identity.ErrRefreshTokenReused)
	}

	if _, err := store.Rotate(t.Context(), digest("v2"), digest("v4"), now.Add(time.Hour), now); !errors.Is(err, identity.ErrRefreshTokenInvalid) {
		t.Errorf("victim's live token error = %v, want it revoked (%v)", err, identity.ErrRefreshTokenInvalid)
	}
	if _, err := store.Rotate(t.Context(), digest("o1"), digest("o2"), now.Add(time.Hour), now); err != nil {
		t.Errorf("another user's token was revoked: %v", err)
	}
}

// TestRotateOfTheSameTokenIsRaceSafe pins the property the conditional UPDATE
// exists for: of many simultaneous refreshes with one token, exactly one wins.
func TestRotateOfTheSameTokenIsRaceSafe(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	store := postgres.NewRefreshTokens(pool)
	user := newUser(t, pool, "owner@example.com")

	if err := store.Issue(t.Context(), user.ID, digest("shared"), now.Add(time.Hour)); err != nil {
		t.Fatalf("Issue: %v", err)
	}

	const attempts = 12
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
		failures  []error
	)

	for i := range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := store.Rotate(t.Context(), digest("shared"), digest(string(rune('a'+i))), now.Add(time.Hour), now)

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			} else {
				failures = append(failures, err)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("successful rotations = %d, want exactly 1", successes)
	}
	for _, err := range failures {
		if !errors.Is(err, identity.ErrRefreshTokenReused) && !errors.Is(err, identity.ErrRefreshTokenInvalid) {
			t.Errorf("losing rotation error = %v, want a reuse or invalid error", err)
		}
	}
}

func TestRevokeIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := dbtest.New(t)
	store := postgres.NewRefreshTokens(pool)
	user := newUser(t, pool, "owner@example.com")

	if err := store.Issue(t.Context(), user.ID, digest("t1"), now.Add(time.Hour)); err != nil {
		t.Fatalf("Issue: %v", err)
	}

	for range 2 {
		if err := store.Revoke(t.Context(), digest("t1"), now); err != nil {
			t.Errorf("Revoke: %v", err)
		}
	}
	if err := store.Revoke(t.Context(), digest("never-issued"), now); err != nil {
		t.Errorf("Revoke of an unknown token: %v", err)
	}
}
