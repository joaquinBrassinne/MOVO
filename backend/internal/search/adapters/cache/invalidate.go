package cache

import (
	"log/slog"
	"net/http"
	"strings"
)

// InvalidateOnWrite returns middleware that drops the search cache after a successful write
// that can change what the public search shows: the owner's business, branches, services and
// benefits, every admin action (visibility, plan, subscription, review moderation) and a new
// review (it moves the rating). It sits at the HTTP edge on purpose: those writes live in
// several modules and none may import the search internals.
func InvalidateOnWrite(store Store, log *slog.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status < http.StatusBadRequest && changesSearch(r.Method, r.URL.Path) {
				if err := Invalidate(r.Context(), store); err != nil {
					log.WarnContext(r.Context(), "search cache invalidation failed", "error", err)
				}
			}
		})
	}
}

// writesThatLeaveSearchAlone are the writes under the watched prefixes that never touch what
// the search shows: they are frequent or private, and invalidating on them would only empty
// the cache for nothing.
var writesThatLeaveSearchAlone = []string{
	"/api/v1/me/business/codes/redeem",
	"/api/v1/me/business/geocode",
	"/api/v1/me/business/images/presign",
	"/api/v1/admin/password-resets",
}

func changesSearch(method, path string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	for _, skip := range writesThatLeaveSearchAlone {
		if path == skip {
			return false
		}
	}
	return strings.HasPrefix(path, "/api/v1/me/business") ||
		strings.HasPrefix(path, "/api/v1/admin/") ||
		(strings.HasPrefix(path, "/api/v1/access/") && strings.HasSuffix(path, "/reviews"))
}

// statusWriter remembers the status code; a handler that never calls WriteHeader answers 200.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the real writer (flush, deadlines).
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
