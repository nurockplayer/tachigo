package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/metrics"
)

func TestHTTPMetricsRecordsRoutePatternAndStatusFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)
	collector := metrics.NewCollector()
	engine := gin.New()
	engine.Use(HTTPMetrics(collector))
	engine.GET("/claim/:token", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/claim/secret-token?access_token=should-not-appear", nil)
	req.Header.Set("Authorization", "Bearer should-not-appear")
	req.Header.Set("Cookie", "session=should-not-appear")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	text := collector.RenderPrometheus()
	if !strings.Contains(text, `tachigo_http_requests_total{route="/claim/:token",status_family="2xx"} 1`) {
		t.Fatalf("expected route-pattern request metric, got:\n%s", text)
	}
	for _, forbidden := range []string{"secret-token", "access_token", "should-not-appear", "Authorization", "Cookie"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metrics leaked forbidden request detail %q:\n%s", forbidden, text)
		}
	}
}

func TestHTTPMetricsRecordsUnmatchedRouteWithoutRawPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	collector := metrics.NewCollector()
	engine := gin.New()
	engine.Use(HTTPMetrics(collector))

	req := httptest.NewRequest(http.MethodGet, "/missing/private-user-123?access_token=should-not-appear", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	text := collector.RenderPrometheus()
	if !strings.Contains(text, `tachigo_http_requests_total{route="__unmatched__",status_family="4xx"} 1`) {
		t.Fatalf("expected unmatched route bucket, got:\n%s", text)
	}
	for _, forbidden := range []string{"private-user-123", "access_token", "should-not-appear"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metrics leaked unmatched request detail %q:\n%s", forbidden, text)
		}
	}
}

func TestMetricsBearerGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/metrics", MetricsBearerGuard("expected-token"), func(c *gin.Context) {
		c.String(http.StatusOK, "metrics")
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to return 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected wrong token to return 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected valid token to return 200, got %d", rec.Code)
	}
}
