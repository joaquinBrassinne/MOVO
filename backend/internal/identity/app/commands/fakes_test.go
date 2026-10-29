package commands_test

import (
	"bytes"
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

var epoch = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour
)

type fakeUsers struct {
	mu      sync.Mutex
	byEmail map[string]domain.User
	lookups int
}

func newFakeUsers() *fakeUsers { return &fakeUsers{byEmail: map[string]domain.User{}} }

func (f *fakeUsers) Create(_ context.Context, user domain.User) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, taken := f.byEmail[user.Email]; taken {
		return domain.User{}, identity.ErrEmailTaken
	}
	user.ID = int64(len(f.byEmail) + 1)
	f.byEmail[user.Email] = user
	return user, nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.lookups++
	if user, ok := f.byEmail[email]; ok {
		return user, nil
	}
	return domain.User{}, identity.ErrUserNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, id int64) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, user := range f.byEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return domain.User{}, identity.ErrUserNotFound
}

type tokenRecord struct {
	userID   int64
	hash     []byte
	expires  time.Time
	revoked  bool
	replaced bool
}

// fakeStore follows the contract of identity.RefreshTokenStore.
type fakeStore struct {
	mu     sync.Mutex
	tokens []*tokenRecord
}

func (f *fakeStore) find(hash []byte) *tokenRecord {
	for _, token := range f.tokens {
		if bytes.Equal(token.hash, hash) {
			return token
		}
	}
	return nil
}

func (f *fakeStore) Issue(_ context.Context, userID int64, hash []byte, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.tokens = append(f.tokens, &tokenRecord{userID: userID, hash: hash, expires: expiresAt})
	return nil
}

func (f *fakeStore) Rotate(_ context.Context, presented, next []byte, nextExpiresAt, now time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	old := f.find(presented)
	switch {
	case old == nil:
		return 0, identity.ErrRefreshTokenInvalid
	case old.replaced:
		for _, token := range f.tokens {
			if token.userID == old.userID {
				token.revoked = true
			}
		}
		return old.userID, identity.ErrRefreshTokenReused
	case old.revoked || !now.Before(old.expires):
		return 0, identity.ErrRefreshTokenInvalid
	}

	old.revoked, old.replaced = true, true
	f.tokens = append(f.tokens, &tokenRecord{userID: old.userID, hash: next, expires: nextExpiresAt})
	return old.userID, nil
}

func (f *fakeStore) Revoke(_ context.Context, hash []byte, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if token := f.find(hash); token != nil {
		token.revoked = true
	}
	return nil
}

// holds reports whether the store keeps exactly this value as a token digest.
func (f *fakeStore) holds(value []byte) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.find(value) != nil
}

type fakeAccess struct{}

func (fakeAccess) Issue(userID int64, role domain.Role, now time.Time) (string, time.Time, error) {
	return "access-" + strconv.FormatInt(userID, 10) + "-" + string(role), now.Add(accessTTL), nil
}

type fakeLimiter struct {
	blocked  map[string]time.Duration
	fails    []string
	resets   []string
	attempts []string
}

// Attempt stands in for the sign-up budget: it records the key and refuses the blocked ones.
func (f *fakeLimiter) Attempt(key string) (time.Duration, bool) {
	f.attempts = append(f.attempts, key)
	retry, blocked := f.blocked[key]
	return retry, !blocked
}

func (f *fakeLimiter) Allow(key string) (time.Duration, bool) {
	retry, blocked := f.blocked[key]
	return retry, !blocked
}
func (f *fakeLimiter) Fail(key string)  { f.fails = append(f.fails, key) }
func (f *fakeLimiter) Reset(key string) { f.resets = append(f.resets, key) }

type harness struct {
	users    *fakeUsers
	store    *fakeStore
	limiter  *fakeLimiter
	register *commands.Register
	login    *commands.Login
	refresh  *commands.Refresh
	logout   *commands.Logout
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{users: newFakeUsers(), store: &fakeStore{}, limiter: &fakeLimiter{blocked: map[string]time.Duration{}}}
	now := func() time.Time { return epoch }
	sessions := commands.NewSessions(fakeAccess{}, h.store, h.users, refreshTTL, now, nil)

	h.register = commands.NewRegister(h.users, h.limiter, sessions)
	h.login = commands.NewLogin(h.users, h.limiter, sessions)
	h.refresh = commands.NewRefresh(sessions)
	h.logout = commands.NewLogout(h.store, now)
	return h
}
