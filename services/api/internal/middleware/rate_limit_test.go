package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/middleware"
)

func TestRateLimiter_Returns429AfterLimit(t *testing.T) {
	limiter := middleware.NewRateLimiter()
	router := gin.New()
	router.GET("/limited", limiter.Limit(middleware.RateLimitConfig{
		Name:    "test",
		Limit:   2,
		Window:  time.Minute,
		KeyFunc: middleware.ClientIPRateLimitKey,
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 2; i++ {
		if got := doLimitedRequest(router); got != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i+1, got)
		}
	}

	if got := doLimitedRequest(router); got != http.StatusTooManyRequests {
		t.Fatalf("want 429 after limit, got %d", got)
	}
}

func doLimitedRequest(router *gin.Engine) int {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	router.ServeHTTP(rec, req)
	return rec.Code
}
