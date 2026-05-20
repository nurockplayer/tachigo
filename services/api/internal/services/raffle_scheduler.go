package services

import (
	"context"
	"log"
	"time"
)

// RaffleScheduler fires RunScheduledSnapshots at 23:55 UTC every day.
type RaffleScheduler struct {
	svc *RaffleService
}

func NewRaffleScheduler(svc *RaffleService) *RaffleScheduler {
	return &RaffleScheduler{svc: svc}
}

// Start launches the background goroutine. Stops when ctx is cancelled.
func (rs *RaffleScheduler) Start(ctx context.Context) {
	go func() {
		for {
			now := time.Now().UTC()
			next := nextSchedulerRun(now)
			log.Printf("event=raffle_scheduler_wait job=raffle_scheduled_snapshots next_run=%s", next.Format(time.RFC3339))
			select {
			case <-ctx.Done():
				log.Printf("event=raffle_scheduler_stop job=raffle_scheduled_snapshots")
				return
			case <-time.After(next.Sub(now)):
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				if err := rs.svc.RunScheduledSnapshots(runCtx, time.Now().UTC()); err != nil {
					log.Printf("event=raffle_scheduler_batch_error job=raffle_scheduled_snapshots err=%q", err)
				}
				cancel()
			}
		}
	}()
}

// nextSchedulerRun returns the next 23:55 UTC after now.
func nextSchedulerRun(now time.Time) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 55, 0, 0, time.UTC)
	if now.Before(today) {
		return today
	}
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 55, 0, 0, time.UTC)
}
