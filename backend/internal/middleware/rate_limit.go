package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vishalyadav/jobplatform/pkg/response"
)

// ratelimitconfig holds configuration for the rate limiter.
type RateLimitConfig struct {
	// requests is the maximum number of requests allowed in the window.
	Requests int
	// window is the sliding window duration.
	Window time.Duration
}

// ratelimit returns middleware that enforces per-user rate limiting using
// a redis sliding window counter. unauthenticated requests are keyed by ip.
//
// algorithm: fixed-window counter via redis incr + expire.
// on the first request in a window, incr creates the key (returns 1) and
// expire sets the ttl. subsequent requests just incr. when count exceeds
// the limit, 429 too many requests is returned with retry-after header.
func RateLimit(rdb *redis.Client, cfg RateLimitConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// determine the rate limit key.
			// if the user is authenticated (user_id in context), use that.
			// otherwise fall back to ip address.
			key := rateLimitKey(r)

			ctx := r.Context()

			// incr atomically increments and returns the new count.
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// if redis is down, fail open — don't block legitimate traffic.
				logger.Warn("rate limiter: Redis error, failing open", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// if this is the first request in the window, set the ttl.
			if count == 1 {
				rdb.Expire(ctx, key, cfg.Window)
			}

			// get the remaining ttl to calculate retry-after.
			ttl, _ := rdb.TTL(ctx, key).Result()

			// set rate limit headers (draft standard).
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, cfg.Requests-int(count))))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			if int(count) > cfg.Requests {
				retryAfter := int(ttl.Seconds())
				if retryAfter <= 0 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))

				logger.Warn("rate limit exceeded",
					"key", key,
					"count", count,
					"limit", cfg.Requests,
					"request_id", RequestIDFromContext(ctx),
				)

				response.Error(w, http.StatusTooManyRequests,
					fmt.Sprintf("rate limit exceeded, retry after %d seconds", retryAfter))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ratelimitkey returns the redis key for rate limiting.
// uses the authenticated user id if available, otherwise the client ip.
func rateLimitKey(r *http.Request) string {
	userID := UserIDFromContext(r.Context())
	if userID.String() != "00000000-0000-0000-0000-000000000000" {
		return fmt.Sprintf("rl:user:%s", userID)
	}
	return fmt.Sprintf("rl:ip:%s", r.RemoteAddr)
}

// max returns the larger of a or b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
