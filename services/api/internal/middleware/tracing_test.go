package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracingMiddlewareRecordsSafeRequestAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)

	r := gin.New()
	r.Use(RequestID())
	r.Use(Tracing(provider.Tracer("test")))
	r.GET("/api/v1/things/:id", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/things/123?access_token=secret-token", nil)
	req.Header.Set(RequestIDHeader, "trace-req-123")
	req.Header.Set("Authorization", "Bearer super-secret-token")
	req.Header.Set("Cookie", "session=secret-cookie")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans: want 1, got %d", len(spans))
	}
	span := spans[0]
	attrs := span.Attributes()
	wantAttrs := map[string]string{
		"http.request.method": "GET",
		"http.route":          "/api/v1/things/:id",
		"request_id":          "trace-req-123",
	}
	for key, want := range wantAttrs {
		if got := stringAttr(attrs, key); got != want {
			t.Fatalf("%s: want %q, got %q (attrs=%v)", key, want, got, attrs)
		}
	}
	if got := intAttr(attrs, "http.response.status_code"); got != http.StatusAccepted {
		t.Fatalf("http.response.status_code: want %d, got %d", http.StatusAccepted, got)
	}

	serialized := attrsToString(attrs)
	for _, leaked := range []string{"secret-token", "super-secret-token", "secret-cookie", "access_token", "Authorization", "Cookie", "/api/v1/things/123"} {
		if strings.Contains(serialized, leaked) {
			t.Fatalf("span attributes leaked %q: %s", leaked, serialized)
		}
	}
}

func TestTracingMiddlewareRecordsErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)

	r := gin.New()
	r.Use(RequestID())
	r.Use(Tracing(provider.Tracer("test")))
	r.GET("/boom", func(c *gin.Context) {
		_ = c.Error(errors.New("database password should not be captured"))
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans: want 1, got %d", len(spans))
	}
	if got := spans[0].Status().Code.String(); got != "Error" {
		t.Fatalf("span status: want Error, got %s", got)
	}
	if strings.Contains(attrsToString(spans[0].Attributes()), "database password") {
		t.Fatalf("span attributes leaked gin error text: %v", spans[0].Attributes())
	}
}

func TestTracingMiddlewareUsesGenericDescriptionForGinErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)

	r := gin.New()
	r.Use(RequestID())
	r.Use(Tracing(provider.Tracer("test")))
	r.GET("/soft-error", func(c *gin.Context) {
		_ = c.Error(errors.New("oauth token should not be captured"))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/soft-error", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans: want 1, got %d", len(spans))
	}
	status := spans[0].Status()
	if got := status.Code.String(); got != "Error" {
		t.Fatalf("span status: want Error, got %s", got)
	}
	if status.Description != "gin errors present" {
		t.Fatalf("span status description: want generic description, got %q", status.Description)
	}
	if strings.Contains(status.Description, "oauth token") {
		t.Fatalf("span status description leaked raw error: %q", status.Description)
	}
}

func stringAttr(attrs []attribute.KeyValue, key string) string {
	for _, attr := range attrs {
		if string(attr.Key) == key {
			return attr.Value.AsString()
		}
	}
	return ""
}

func intAttr(attrs []attribute.KeyValue, key string) int {
	for _, attr := range attrs {
		if string(attr.Key) == key {
			return int(attr.Value.AsInt64())
		}
	}
	return 0
}

func attrsToString(attrs []attribute.KeyValue) string {
	var b strings.Builder
	for _, attr := range attrs {
		b.WriteString(string(attr.Key))
		b.WriteByte('=')
		b.WriteString(attr.Value.Emit())
		b.WriteByte(' ')
	}
	return b.String()
}
