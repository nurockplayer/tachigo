package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestCollectorRendersHTTPMetricsWithoutRequestSecrets(t *testing.T) {
	collector := NewCollector()

	collector.ObserveHTTPRequest("/api/v1/claim/:token", 200, 12*time.Millisecond)
	collector.ObserveHTTPRequest("/api/v1/claim/:token", 503, 30*time.Millisecond)

	text := collector.RenderPrometheus()

	for _, want := range []string{
		`tachigo_http_requests_total{route="/api/v1/claim/:token",status_family="2xx"} 1`,
		`tachigo_http_requests_total{route="/api/v1/claim/:token",status_family="5xx"} 1`,
		`tachigo_http_request_errors_total{route="/api/v1/claim/:token",status_family="5xx"} 1`,
		`tachigo_http_request_duration_seconds_count{route="/api/v1/claim/:token",status_family="2xx"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected rendered metrics to contain %q, got:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"access_token", "Authorization", "Bearer", "cookie", "voucher", "receipt", "user_id"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metrics text leaked forbidden value %q:\n%s", forbidden, text)
		}
	}
}

func TestCollectorRendersRaffleSchedulerMetricsByResult(t *testing.T) {
	collector := NewCollector()

	collector.ObserveRaffleSchedulerRun("success", 1400*time.Millisecond)
	collector.ObserveRaffleSchedulerRun("failure", 2*time.Second)

	text := collector.RenderPrometheus()

	for _, want := range []string{
		`tachigo_raffle_scheduler_runs_total{result="success"} 1`,
		`tachigo_raffle_scheduler_runs_total{result="failure"} 1`,
		`tachigo_raffle_scheduler_failures_total{result="failure"} 1`,
		`tachigo_raffle_scheduler_duration_seconds_count{result="success"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected rendered metrics to contain %q, got:\n%s", want, text)
		}
	}
}
