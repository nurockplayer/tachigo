package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/metrics"
	"github.com/tachigo/tachigo/internal/models"
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

func TestRunScheduledSnapshotsRecordsPartialFailureWithoutFailureCounter(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "test-client-id", "", nil)
	collector := metrics.NewCollector()
	svc.SetMetricsCollector(collector)

	user := insertRaffleTestUser(t, db)
	insertScheduledRaffle(t, db, user.ID, time.Now().UTC().Add(5*time.Minute), models.RaffleSourceTwitchAPI)

	if err := svc.RunScheduledSnapshots(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}

	text := collector.RenderPrometheus()
	for _, want := range []string{
		`tachigo_raffle_scheduler_runs_total{result="partial_failure"} 1`,
		`tachigo_raffle_scheduler_duration_seconds_count{result="partial_failure"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected scheduler partial failure metric %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, `tachigo_raffle_scheduler_failures_total{result="failure"}`) {
		t.Fatalf("partial per-raffle errors must not increment batch failure counter, got:\n%s", text)
	}
}
