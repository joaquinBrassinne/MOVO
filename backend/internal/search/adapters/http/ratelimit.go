package searchhttp

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/platform/clientip"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// CodeRateLimited is the code of a search refused because the client asked too often.
const CodeRateLimited = "rate_limited"

// Limiter counts one request of a key and says whether it may go on. When it may not,
// retryAfter is how long until the key can try again.
type Limiter interface {
	Allow(key string) (retryAfter time.Duration, ok bool)
}

// WithRateLimit limits GET /search and GET /home/featured per client address, so a script
// cannot drain the database that every visitor shares. A nil limiter leaves them unlimited.
func (h *Handlers) WithRateLimit(l Limiter) *Handlers {
	h.limiter = l
	return h
}

// limit refuses the request with 429 once the client has spent its budget.
func (h *Handlers) limit(next http.Handler) http.Handler {
	if h.limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retry, ok := h.limiter.Allow("search:" + clientip.From(r)); !ok {
			seconds := max(1, int(math.Ceil(retry.Seconds())))
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			httpx.WriteError(w, http.StatusTooManyRequests, CodeRateLimited, "too many requests, try again later", map[string]int{"retry_after_seconds": seconds})
			return
		}
		next.ServeHTTP(w, r)
	})
}
