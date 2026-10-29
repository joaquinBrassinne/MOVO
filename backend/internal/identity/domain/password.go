package domain

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ErrHashInvalid reports a stored password hash that is not a PHC string this
// package produced. It is a data problem, never a wrong password.
var ErrHashInvalid = errors.New("identity: password hash is not a valid argon2id PHC string")

// argon2id parameters: the OWASP "m=19 MiB, t=2, p=1" profile.
const (
	argonMemoryKiB = 19 * 1024
	argonTime      = 2
	argonThreads   = 1
	argonKeyLen    = 32
	argonSaltLen   = 16
)

// maxConcurrentHashes bounds the argon2 runs in flight. Each one holds
// argonMemoryKiB while it works, so a burst of logins would otherwise take
// memory proportional to the burst; the rest queue.
const maxConcurrentHashes = 4

// hashSlots is the semaphore behind maxConcurrentHashes. It is never reassigned.
var hashSlots = make(chan struct{}, maxConcurrentHashes)

// phcEncoding is the unpadded standard base64 the PHC string format uses.
var phcEncoding = base64.RawStdEncoding

// deriveKey runs argon2id once the caller holds one of the hashing slots.
func deriveKey(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
	hashSlots <- struct{}{}
	defer func() { <-hashSlots }()

	return argon2.IDKey(password, salt, time, memory, threads, keyLen)
}

// HashPassword returns the argon2id hash of password as a PHC string with a
// fresh random salt. Callers validate the password with ValidatePassword first.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("identity: read salt: %w", err)
	}

	key := deriveKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		phcEncoding.EncodeToString(salt), phcEncoding.EncodeToString(key)), nil
}

// VerifyPassword reports whether password matches hash, comparing in constant
// time. A hash that cannot be read returns ErrHashInvalid.
func VerifyPassword(hash, password string) (bool, error) {
	params, salt, want, err := parsePHC(hash)
	if err != nil {
		return false, err
	}

	got := deriveKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(want)))

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

// parsePHC splits "$argon2id$v=19$m=..,t=..,p=..$salt$key".
func parsePHC(hash string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, ErrHashInvalid
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, ErrHashInvalid
	}

	var params argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads); err != nil ||
		params.memory == 0 || params.time == 0 || params.threads == 0 {
		return argonParams{}, nil, nil, ErrHashInvalid
	}

	salt, err := phcEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return argonParams{}, nil, nil, ErrHashInvalid
	}

	key, err := phcEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return argonParams{}, nil, nil, ErrHashInvalid
	}

	return params, salt, key, nil
}
