package domain

import (
	"testing"
	"time"
)

// Every hash holds ~19 MiB while it runs, so the number running at once is capped:
// a burst of logins queues instead of exhausting memory.
func TestHashingWaitsWhenAllSlotsAreTaken(t *testing.T) {
	for range cap(hashSlots) {
		hashSlots <- struct{}{}
	}

	hashed := make(chan struct{})
	verified := make(chan struct{})
	go func() {
		_, _ = HashPassword("a password long enough")
		close(hashed)
	}()
	go func() {
		_, _ = VerifyPassword("$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHQ$a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5", "x")
		close(verified)
	}()

	select {
	case <-hashed:
		t.Fatal("HashPassword ran with every slot taken")
	case <-verified:
		t.Fatal("VerifyPassword ran with every slot taken")
	case <-time.After(150 * time.Millisecond):
	}

	for range cap(hashSlots) {
		<-hashSlots
	}

	for _, done := range []chan struct{}{hashed, verified} {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("hashing did not resume after the slots were released")
		}
	}
}
