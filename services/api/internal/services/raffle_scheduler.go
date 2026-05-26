package services

import (
	"context"
	"log"
	"time"
)

// snapshotCheckInterval is how often the scheduler scans for raffles whose
// scheduled_at has arrived (or is within the look-ahead window). It must be
// <= the look-ahead window in RunScheduledSnapshots so every scheduled raffle is
// covered by at least one scan.
const snapshotCheckInterval = 5 * time.Minute

// RaffleScheduler periodically triggers snapshots for raffles whose scheduled_at
// has arrived. It scans every snapshotCheckInterval rather than once a day, so a
// raffle scheduled for any time of day is picked up — not only those near a
// single daily run.
type RaffleScheduler struct {
	svc *RaffleService
}

func NewRaffleScheduler(svc *RaffleService) *RaffleScheduler {
	return &RaffleScheduler{svc: svc}
}

// Start launches the background goroutine. It runs an initial scan immediately —
// so raffles whose scheduled time elapsed while the server was down are caught on
// restart — and then re-scans every snapshotCheckInterval until ctx is cancelled.
func (rs *RaffleScheduler) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(snapshotCheckInterval)
		defer ticker.Stop()

		rs.runOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				log.Printf("event=raffle_scheduler_stop job=raffle_scheduled_snapshots")
				return
			case <-ticker.C:
				rs.runOnce(ctx)
			}
		}
	}()
}

func (rs *RaffleScheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := rs.svc.RunScheduledSnapshots(runCtx, time.Now().UTC()); err != nil {
		log.Printf("event=raffle_scheduler_batch_error job=raffle_scheduled_snapshots err=%q", err)
	}
}
