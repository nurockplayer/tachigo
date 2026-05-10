package middleware_test

import (
	"context"
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

func TestRequestTimeout_AddsDeadlineToRequestContext(t *testing.T) {
	router := gin.New()
	router.Use(middleware.RequestTimeout(50 * time.Millisecond))
	router.GET("/deadline", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); !ok {
			t.Fatal("expected request context deadline")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/deadline", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequestTimeout_Returns504WhenHandlerStopsOnDeadline(t *testing.T) {
	router := gin.New()
	router.Use(middleware.RequestTimeout(time.Millisecond))
	router.GET("/slow", func(c *gin.Context) {
		<-c.Request.Context().Done()
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("want 504, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequestTimeout_DisabledForNonPositiveDuration(t *testing.T) {
	router := gin.New()
	router.Use(middleware.RequestTimeout(0))
	router.GET("/disabled", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); ok {
			t.Fatal("expected request context without deadline")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/disabled", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
