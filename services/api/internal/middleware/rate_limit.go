package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitKeyFunc func(*gin.Context) string

type RateLimitConfig struct {
	Name    string
	Limit   int
	Window  time.Duration
	KeyFunc RateLimitKeyFunc
}

type RateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	entries map[string]rateLimitEntry
}

type rateLimitEntry struct {
	windowStart time.Time
	count       int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		now:     time.Now,
		entries: make(map[string]rateLimitEntry),
	}
}

func (l *RateLimiter) Limit(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || cfg.Limit <= 0 || cfg.Window <= 0 {
			c.Next()
			return
		}

		key := cfg.Name + "|" + ClientIPRateLimitKey(c)
		if cfg.KeyFunc != nil {
			key = cfg.Name + "|" + cfg.KeyFunc(c)
		}

		allowed, retryAfter := l.allow(key, cfg.Limit, cfg.Window)
		if !allowed {
			c.Header("Retry-After", retryAfterSeconds(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

func (l *RateLimiter) allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= window {
		l.entries[key] = rateLimitEntry{windowStart: now, count: 1}
		l.cleanupLocked(now, window)
		return true, 0
	}

	if entry.count >= limit {
		return false, entry.windowStart.Add(window).Sub(now)
	}

	entry.count++
	l.entries[key] = entry
	return true, 0
}

func retryAfterSeconds(d time.Duration) string {
	seconds := int((d + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

func (l *RateLimiter) cleanupLocked(now time.Time, window time.Duration) {
	if len(l.entries) < 10000 {
		return
	}
	for key, entry := range l.entries {
		if now.Sub(entry.windowStart) >= window {
			delete(l.entries, key)
		}
	}
}

func ClientIPRateLimitKey(c *gin.Context) string {
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	if c.Request == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if c.Request.RemoteAddr != "" {
		return c.Request.RemoteAddr
	}
	return "unknown"
}
