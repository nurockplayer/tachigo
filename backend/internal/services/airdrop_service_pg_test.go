//go:build integration

package services

import (
	"errors"
	"sync"
	"testing"
)

func TestAirdrop_ConcurrentSERIALIZABLE(t *testing.T) {
	db := newPGTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	channelID := "ch_pg_serializable"
	if err := db.Exec(
		`INSERT INTO channel_configs (channel_id, seconds_per_point, multiplier, daily_airdrop_limit, created_at, updated_at)
		 VALUES (?, 60, 1, 100, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		channelID,
	).Error; err != nil {
		t.Fatalf("seed channel config: %v", err)
	}

	for range 5 {
		seedAirdropViewer(t, db, channelID, 60)
	}

	start := make(chan struct{})
	errCh := make(chan error, 3)

	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Execute(AirdropRequest{
				ChannelID: channelID,
				Amount:    40,
			})
			errCh <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err == nil {
			continue
		}
		if errors.Is(err, ErrDailyAirdropExceeded) {
			continue
		}
		t.Fatalf("unexpected execute error: %v", err)
	}

	todayTotal, err := svc.TodayTotal(channelID)
	if err != nil {
		t.Fatalf("today total: %v", err)
	}
	if todayTotal > 100 {
		t.Fatalf("want today total <= 100, got %d", todayTotal)
	}
}
