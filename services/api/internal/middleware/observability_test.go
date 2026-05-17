package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDAndStructuredRequestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logs bytes.Buffer
	r := gin.New()
	r.Use(RequestID())
	r.Use(StructuredRequestLogger(log.New(&logs, "", 0)))
	r.GET("/api/v1/things/:id", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/things/123?access_token=secret-token", nil)
	req.Header.Set(RequestIDHeader, "req-safe-123")
	req.Header.Set("Authorization", "Bearer super-secret-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) != "req-safe-123" {
		t.Fatalf("expected request id header to be preserved, got %q", rec.Header().Get(RequestIDHeader))
	}

	line := logs.String()
	for _, want := range []string{
		"event=http_request",
		"request_id=req-safe-123",
		"method=GET",
		"route=/api/v1/things/:id",
		"status=202",
		"duration_ms=",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected log to contain %q, got %q", want, line)
		}
	}
	for _, leaked := range []string{"secret-token", "super-secret-token", "access_token", "Bearer"} {
		if strings.Contains(line, leaked) {
			t.Fatalf("request log leaked %q: %s", leaked, line)
		}
	}
}

func TestRequestIDGeneratesHeaderWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequestID())
	r.GET("/health", func(c *gin.Context) {
		if RequestIDFromGin(c) == "" {
			t.Fatal("expected request id in gin context")
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) == "" {
		t.Fatal("expected generated request id response header")
	}
}
