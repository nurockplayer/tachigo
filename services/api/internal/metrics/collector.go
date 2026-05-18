package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	httpDurationBuckets      = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	schedulerDurationBuckets = []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300}
)

type Collector struct {
	mu sync.Mutex

	httpRequests map[httpKey]*histogram
	scheduler    map[string]*histogram
}

type httpKey struct {
	route        string
	statusFamily string
}

type histogram struct {
	count   uint64
	sum     float64
	buckets []uint64
}

func NewCollector() *Collector {
	return &Collector{
		httpRequests: make(map[httpKey]*histogram),
		scheduler:    make(map[string]*histogram),
	}
}

func (c *Collector) ObserveHTTPRequest(route string, status int, duration time.Duration) {
	if c == nil {
		return
	}
	key := httpKey{
		route:        safeRoute(route),
		statusFamily: statusFamily(status),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	h := c.httpRequests[key]
	if h == nil {
		h = newHistogram(len(httpDurationBuckets))
		c.httpRequests[key] = h
	}
	h.observe(duration.Seconds(), httpDurationBuckets)
}

func (c *Collector) ObserveRaffleSchedulerRun(result string, duration time.Duration) {
	if c == nil {
		return
	}
	result = safeResult(result)

	c.mu.Lock()
	defer c.mu.Unlock()

	h := c.scheduler[result]
	if h == nil {
		h = newHistogram(len(schedulerDurationBuckets))
		c.scheduler[result] = h
	}
	h.observe(duration.Seconds(), schedulerDurationBuckets)
}

func (c *Collector) RenderPrometheus() string {
	if c == nil {
		return ""
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var b strings.Builder
	writeHTTPMetrics(&b, c.httpRequests)
	writeSchedulerMetrics(&b, c.scheduler)
	return b.String()
}

func newHistogram(bucketCount int) *histogram {
	return &histogram{buckets: make([]uint64, bucketCount)}
}

func (h *histogram) observe(value float64, buckets []float64) {
	if value < 0 {
		value = 0
	}
	h.count++
	h.sum += value
	for i, bucket := range buckets {
		if value <= bucket {
			h.buckets[i]++
		}
	}
}

func writeHTTPMetrics(b *strings.Builder, metrics map[httpKey]*histogram) {
	keys := make([]httpKey, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route == keys[j].route {
			return keys[i].statusFamily < keys[j].statusFamily
		}
		return keys[i].route < keys[j].route
	})

	b.WriteString("# HELP tachigo_http_requests_total HTTP requests by route pattern and status family.\n")
	b.WriteString("# TYPE tachigo_http_requests_total counter\n")
	for _, key := range keys {
		h := metrics[key]
		fmt.Fprintf(b, "tachigo_http_requests_total{route=%q,status_family=%q} %d\n", escapeLabelValue(key.route), key.statusFamily, h.count)
	}

	b.WriteString("# HELP tachigo_http_request_errors_total HTTP 5xx responses by route pattern and status family.\n")
	b.WriteString("# TYPE tachigo_http_request_errors_total counter\n")
	for _, key := range keys {
		if key.statusFamily != "5xx" {
			continue
		}
		h := metrics[key]
		fmt.Fprintf(b, "tachigo_http_request_errors_total{route=%q,status_family=%q} %d\n", escapeLabelValue(key.route), key.statusFamily, h.count)
	}

	writeHTTPHistogram(b, keys, metrics)
}

func writeHTTPHistogram(b *strings.Builder, keys []httpKey, metrics map[httpKey]*histogram) {
	b.WriteString("# HELP tachigo_http_request_duration_seconds HTTP request duration by route pattern and status family.\n")
	b.WriteString("# TYPE tachigo_http_request_duration_seconds histogram\n")
	for _, key := range keys {
		h := metrics[key]
		for i, bucket := range httpDurationBuckets {
			fmt.Fprintf(b, "tachigo_http_request_duration_seconds_bucket{route=%q,status_family=%q,le=%q} %d\n", escapeLabelValue(key.route), key.statusFamily, formatBucket(bucket), h.buckets[i])
		}
		fmt.Fprintf(b, "tachigo_http_request_duration_seconds_bucket{route=%q,status_family=%q,le=\"+Inf\"} %d\n", escapeLabelValue(key.route), key.statusFamily, h.count)
		fmt.Fprintf(b, "tachigo_http_request_duration_seconds_sum{route=%q,status_family=%q} %g\n", escapeLabelValue(key.route), key.statusFamily, h.sum)
		fmt.Fprintf(b, "tachigo_http_request_duration_seconds_count{route=%q,status_family=%q} %d\n", escapeLabelValue(key.route), key.statusFamily, h.count)
	}
}

func writeSchedulerMetrics(b *strings.Builder, metrics map[string]*histogram) {
	results := make([]string, 0, len(metrics))
	for result := range metrics {
		results = append(results, result)
	}
	sort.Strings(results)

	b.WriteString("# HELP tachigo_raffle_scheduler_runs_total Raffle scheduler runs by result.\n")
	b.WriteString("# TYPE tachigo_raffle_scheduler_runs_total counter\n")
	for _, result := range results {
		h := metrics[result]
		fmt.Fprintf(b, "tachigo_raffle_scheduler_runs_total{result=%q} %d\n", result, h.count)
	}

	b.WriteString("# HELP tachigo_raffle_scheduler_failures_total Raffle scheduler failed runs.\n")
	b.WriteString("# TYPE tachigo_raffle_scheduler_failures_total counter\n")
	for _, result := range results {
		if result != "failure" {
			continue
		}
		h := metrics[result]
		fmt.Fprintf(b, "tachigo_raffle_scheduler_failures_total{result=%q} %d\n", result, h.count)
	}

	b.WriteString("# HELP tachigo_raffle_scheduler_duration_seconds Raffle scheduler run duration by result.\n")
	b.WriteString("# TYPE tachigo_raffle_scheduler_duration_seconds histogram\n")
	for _, result := range results {
		h := metrics[result]
		for i, bucket := range schedulerDurationBuckets {
			fmt.Fprintf(b, "tachigo_raffle_scheduler_duration_seconds_bucket{result=%q,le=%q} %d\n", result, formatBucket(bucket), h.buckets[i])
		}
		fmt.Fprintf(b, "tachigo_raffle_scheduler_duration_seconds_bucket{result=%q,le=\"+Inf\"} %d\n", result, h.count)
		fmt.Fprintf(b, "tachigo_raffle_scheduler_duration_seconds_sum{result=%q} %g\n", result, h.sum)
		fmt.Fprintf(b, "tachigo_raffle_scheduler_duration_seconds_count{result=%q} %d\n", result, h.count)
	}
}

func statusFamily(status int) string {
	if status < 100 || status > 599 {
		return "unknown"
	}
	return fmt.Sprintf("%dxx", status/100)
}

func safeRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return "__unmatched__"
	}
	return route
}

func safeResult(result string) string {
	switch result {
	case "success", "partial_failure", "failure":
		return result
	default:
		return "unknown"
	}
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func formatBucket(bucket float64) string {
	return fmt.Sprintf("%g", bucket)
}
