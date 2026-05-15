package middleware

import (
	"strconv"
	"sync"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/metrics"
	"github.com/gin-gonic/gin"
)

type windowCounter struct {
	windowStart time.Time
	count       int
}

type UserRateLimiter struct {
	name    string
	mu      sync.Mutex
	store   map[string]*windowCounter
	limit   int
	window  time.Duration
	cleanup time.Duration
}

func NewUserRateLimiter(limitPerMinute int) *UserRateLimiter {
	return NewNamedUserRateLimiter("user", limitPerMinute)
}

// NewNamedUserRateLimiter constructs an additional per-user limiter with its
// own state. The name is used for the metric label and forwarded to logs so
// stacked limiters (e.g. baseline `user` + tighter `ai`) can be told apart
// without changing the public Middleware contract.
func NewNamedUserRateLimiter(name string, limitPerMinute int) *UserRateLimiter {
	if limitPerMinute <= 0 {
		limitPerMinute = 100
	}
	if name == "" {
		name = "user"
	}

	l := &UserRateLimiter{
		name:    name,
		store:   make(map[string]*windowCounter),
		limit:   limitPerMinute,
		window:  time.Minute,
		cleanup: 5 * time.Minute,
	}

	go l.reaper()
	return l
}

func (l *UserRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(CtxUserIDKey)
		if !ok {
			c.Next()
			return
		}

		userID, ok := raw.(uint)
		if !ok {
			c.Next()
			return
		}

		key := strconv.FormatUint(uint64(userID), 10)
		allowed, remaining, resetAt := l.take(key)

		c.Header("X-RateLimit-Limit", strconv.Itoa(l.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			metrics.RateLimitRejections.WithLabelValues(l.name).Inc()
			api.TooManyRequests(c, "请求过于频繁，请稍后重试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (l *UserRateLimiter) take(key string) (bool, int, time.Time) {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	rec, ok := l.store[key]
	if !ok || now.Sub(rec.windowStart) >= l.window {
		rec = &windowCounter{windowStart: now.Truncate(l.window), count: 0}
		l.store[key] = rec
	}

	resetAt := rec.windowStart.Add(l.window)
	if rec.count >= l.limit {
		return false, 0, resetAt
	}

	rec.count++
	remaining := l.limit - rec.count
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, resetAt
}

func (l *UserRateLimiter) reaper() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		l.mu.Lock()
		for key, rec := range l.store {
			if now.Sub(rec.windowStart) >= 2*l.window {
				delete(l.store, key)
			}
		}
		l.mu.Unlock()
	}
}
