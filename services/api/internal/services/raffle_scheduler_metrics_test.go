package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/metrics"
)

func TestRunScheduledSnapshotsRecordsSchedulerMetrics(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)
	collector := metrics.NewCollector()
	svc.SetMetricsCollector(collector)

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("RunScheduledSnapshots: %v", err)
	}

	text := collector.RenderPrometheus()
	for _, want := range []string{
		`tachigo_raffle_scheduler_runs_total{result="success"} 1`,
		`tachigo_raffle_scheduler_duration_seconds_count{result="success"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected scheduler metric %q, got:\n%s", want, text)
		}
	}
}

func TestRunScheduledSnapshotsRecordsFailureMetrics(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	svc := NewRaffleService(db, "", "", nil)
	collector := metrics.NewCollector()
	svc.SetMetricsCollector(collector)

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err == nil {
		t.Fatal("expected RunScheduledSnapshots to fail after closing db")
	}

	text := collector.RenderPrometheus()
	for _, want := range []string{
		`tachigo_raffle_scheduler_runs_total{result="failure"} 1`,
		`tachigo_raffle_scheduler_failures_total{result="failure"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected scheduler failure metric %q, got:\n%s", want, text)
		}
	}
}
