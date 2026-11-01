// Package memory holds the in-process adapters of the search module.
package memory

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"sync"
	"time"
)

// Deduper implements search.Deduper in memory. A key is kept as a salted SHA-256 digest
// that nobody can turn back into an address, only for the window, and never written to
// disk. The salt is random per process, so a restart forgets everything.
type Deduper struct {
	window  time.Duration
	maxKeys int
	now     func() time.Time
	salt    [16]byte

	mu   sync.Mutex
	seen map[[16]byte]time.Time
}

// NewDeduper builds a Deduper that ignores a repeated key for window. At most maxKeys are
// remembered: when the table is full of live entries a new key is let through and not
// remembered, so memory stays bounded and a flood costs accuracy, never availability. A
// nil now uses time.Now.
func NewDeduper(window time.Duration, maxKeys int, now func() time.Time) *Deduper {
	if now == nil {
		now = time.Now
	}
	d := &Deduper{window: window, maxKeys: maxKeys, now: now, seen: make(map[[16]byte]time.Time)}
	if _, err := rand.Read(d.salt[:]); err != nil {
		// No entropy means no salt: the digests stay unlinkable to a person only as far as
		// the table is private, which it is. Carry on rather than refuse to serve.
		d.salt = [16]byte{}
	}
	return d
}

// Seen records the key and reports whether it was already recorded within the window. A
// repeat does not extend the window: it runs from the first count.
func (d *Deduper) Seen(_ context.Context, key string) (bool, error) {
	digest := d.digest(key)
	now := d.now()

	d.mu.Lock()
	defer d.mu.Unlock()

	if at, ok := d.seen[digest]; ok && now.Sub(at) < d.window {
		return true, nil
	}
	if len(d.seen) >= d.maxKeys {
		d.sweep(now)
		if len(d.seen) >= d.maxKeys {
			return false, nil
		}
	}
	d.seen[digest] = now
	return false, nil
}

func (d *Deduper) digest(key string) [16]byte {
	sum := sha256.Sum256(append(d.salt[:], key...))
	var out [16]byte
	copy(out[:], sum[:])
	return out
}

// sweep drops the entries whose window is over. The caller holds the lock.
func (d *Deduper) sweep(now time.Time) {
	for digest, at := range d.seen {
		if now.Sub(at) >= d.window {
			delete(d.seen, digest)
		}
	}
}
