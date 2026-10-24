package httpx

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

// Codes of the authentication and authorization failures.
const (
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
)

// Principal is the authenticated caller of a request. Role is the plain string
// stored in the token, so platform needs no knowledge of any module's types.
type Principal struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

// TokenVerifier turns the bearer token of a request into a Principal. It
// reports an error for anything it does not fully trust (bad signature, wrong
// algorithm, expired).
type TokenVerifier interface {
	Verify(token string) (Principal, error)
}

type principalKey struct{}

// PrincipalFrom returns the caller authenticated by Authenticate.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}

	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

// Authenticate requires a valid "Authorization: Bearer <token>" header. Every
// failure answers the same 401 so a caller cannot tell a malformed header from
// a forged or expired token.
func Authenticate(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w)
				return
			}

			principal, err := verifier.Verify(token)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
		})
	}
}

// RequireRole lets through only callers whose role is one of roles. It must run
// after Authenticate; without a principal it fails closed with a 401.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFrom(r.Context())
			if !ok {
				writeUnauthorized(w)
				return
			}

			if !slices.Contains(roles, principal.Role) {
				WriteError(w, http.StatusForbidden, CodeForbidden, "you do not have access to this resource", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extracts the token of a "Bearer <token>" header; the scheme is
// case-insensitive and exactly one token must follow it.
func bearerToken(header string) (string, bool) {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}
	return fields[1], true
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication is required", nil)
}
