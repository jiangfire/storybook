package middleware

import (
	"sync"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	r        rate.Limit
	burst    int
}

func NewIPLimiter(eventsPerMinute int, burst int) *IPLimiter {
	if burst <= 0 {
		burst = 1
	}

	l := &IPLimiter{
		visitors: make(map[string]*visitor),
		r:        rate.Every(time.Minute / time.Duration(eventsPerMinute)),
		burst:    burst,
	}

	go l.cleanup()
	return l
}

func (l *IPLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := l.getLimiter(ip)
		if !limiter.Allow() {
			api.TooManyRequests(c, "请求过于频繁，请稍后重试")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (l *IPLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, exists := l.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(l.r, l.burst)
		l.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (l *IPLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}
