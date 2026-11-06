package domain

import "crypto/sha256"

// HashAccessToken is the stored form of a customer's access token: its SHA-256. It is the
// same rule the benefits module uses to store the token; modules do not share internals,
// so this is its own copy and the postgres tests pin that both agree.
func HashAccessToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
