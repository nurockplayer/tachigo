package services

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tachigo/tachigo/internal/models"
)

func seedAirdropViewer(t *testing.T, db *gorm.DB, channelID string, accumulatedSeconds int64) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	if err := db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		userID,
	).Error; err != nil {
		t.Fatalf("seed viewer user: %v", err)
	}

	if err := db.Exec(
		`INSERT INTO watch_sessions (
			id, user_id, channel_id, accumulated_seconds, rewarded_seconds,
			last_heartbeat_at, click_cooldown_until, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, 0, ?, '1970-01-01 00:00:00', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), userID, channelID, accumulatedSeconds, time.Now(),
	).Error; err != nil {
		t.Fatalf("seed watch session: %v", err)
	}

	return userID
}

func TestAirdrop_NoActiveSessions(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	_, err := svc.Execute(AirdropRequest{
		ChannelID: "ch_empty",
		Amount:    100,
	})
	if !errors.Is(err, ErrNoActiveViewers) {
		t.Fatalf("want ErrNoActiveViewers, got %v", err)
	}
}

func TestAirdrop_DailyLimitExceeded(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	seedAirdropViewer(t, db, "ch_limit", 60)

	if err := db.Exec(
		`INSERT INTO channel_configs (channel_id, seconds_per_point, multiplier, daily_airdrop_limit, created_at, updated_at)
		 VALUES (?, 60, 1, 5000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		"ch_limit",
	).Error; err != nil {
		t.Fatalf("seed channel config: %v", err)
	}

	if _, err := svc.Execute(AirdropRequest{ChannelID: "ch_limit", Amount: 4500}); err != nil {
		t.Fatalf("first execute: %v", err)
	}

	_, err := svc.Execute(AirdropRequest{ChannelID: "ch_limit", Amount: 600})
	if !errors.Is(err, ErrDailyAirdropExceeded) {
		t.Fatalf("want ErrDailyAirdropExceeded, got %v", err)
	}

	var exceededErr *DailyAirdropExceededError
	if !errors.As(err, &exceededErr) {
		t.Fatalf("want DailyAirdropExceededError, got %T", err)
	}
	if exceededErr.Remaining != 500 {
		t.Fatalf("want remaining 500, got %d", exceededErr.Remaining)
	}
}

func TestAirdrop_AmountMustBePositive(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	seedAirdropViewer(t, db, "ch_positive", 60)

	_, err := svc.Execute(AirdropRequest{
		ChannelID: "ch_positive",
		Amount:    0,
	})
	if !errors.Is(err, ErrInvalidPointsAmount) {
		t.Fatalf("want ErrInvalidPointsAmount, got %v", err)
	}
}

func TestAirdrop_ProportionalDistribution(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	viewerA := seedAirdropViewer(t, db, "ch_ratio", 60)
	viewerB := seedAirdropViewer(t, db, "ch_ratio", 120)

	result, err := svc.Execute(AirdropRequest{
		ChannelID: "ch_ratio",
		Amount:    300,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.AffectedCount != 2 {
		t.Fatalf("want affected_count 2, got %d", result.AffectedCount)
	}

	balanceA, err := pointsSvc.GetBalance(viewerA, "ch_ratio")
	if err != nil {
		t.Fatalf("balance viewerA: %v", err)
	}
	if balanceA.SpendableBalance != 100 {
		t.Fatalf("viewerA want 100, got %d", balanceA.SpendableBalance)
	}

	balanceB, err := pointsSvc.GetBalance(viewerB, "ch_ratio")
	if err != nil {
		t.Fatalf("balance viewerB: %v", err)
	}
	if balanceB.SpendableBalance != 200 {
		t.Fatalf("viewerB want 200, got %d", balanceB.SpendableBalance)
	}
}

func TestAirdrop_WritesAirdropSource(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	seedAirdropViewer(t, db, "ch_source", 60)

	if _, err := svc.Execute(AirdropRequest{
		ChannelID: "ch_source",
		Amount:    100,
		Note:      "promo",
	}); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var tx models.PointsTransaction
	if err := db.Order("created_at DESC, id DESC").First(&tx).Error; err != nil {
		t.Fatalf("load tx: %v", err)
	}
	if tx.Source != models.TxSourceAirdrop {
		t.Fatalf("want source %q, got %q", models.TxSourceAirdrop, tx.Source)
	}
}

func TestAirdrop_ConcurrentExecute_DoesNotExceedDailyLimit(t *testing.T) {
	db := newConcurrentTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	channelID := "ch_concurrent_limit"
	seedAirdropViewer(t, db, channelID, 60)

	if err := db.Exec(
		`INSERT INTO channel_configs (channel_id, seconds_per_point, multiplier, daily_airdrop_limit, created_at, updated_at)
		 VALUES (?, 60, 1, 5000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		channelID,
	).Error; err != nil {
		t.Fatalf("seed channel config: %v", err)
	}

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Execute(AirdropRequest{
				ChannelID: channelID,
				Amount:    3000,
			})
			errCh <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	successes := 0
	exceeded := 0
	for err := range errCh {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrDailyAirdropExceeded):
			exceeded++
		default:
			t.Fatalf("unexpected concurrent execute error: %v", err)
		}
	}

	if successes != 1 {
		t.Fatalf("want exactly 1 successful airdrop, got %d", successes)
	}
	if exceeded != 1 {
		t.Fatalf("want exactly 1 daily limit error, got %d", exceeded)
	}

	todayTotal, err := svc.TodayTotal(channelID)
	if err != nil {
		t.Fatalf("today total: %v", err)
	}
	if todayTotal > 5000 {
		t.Fatalf("daily limit exceeded: got total %d", todayTotal)
	}
}

func TestAirdrop_StaleViewerExcluded(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	// Seed a fresh viewer.
	freshUser := seedAirdropViewer(t, db, "ch_stale", 60)

	// Seed a stale viewer: is_active=true but last_heartbeat_at is 5 minutes ago.
	staleUser := uuid.New()
	if err := db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		staleUser,
	).Error; err != nil {
		t.Fatalf("seed stale user: %v", err)
	}
	staleHeartbeat := time.Now().Add(-5 * time.Minute)
	if err := db.Exec(
		`INSERT INTO watch_sessions (
			id, user_id, channel_id, accumulated_seconds, rewarded_seconds,
			last_heartbeat_at, click_cooldown_until, is_active, created_at, updated_at
		) VALUES (?, ?, 'ch_stale', 120, 0, ?, '1970-01-01 00:00:00', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), staleUser, staleHeartbeat,
	).Error; err != nil {
		t.Fatalf("seed stale session: %v", err)
	}

	result, err := svc.Execute(AirdropRequest{ChannelID: "ch_stale", Amount: 100})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.AffectedCount != 1 {
		t.Fatalf("want 1 recipient (fresh only), got %d", result.AffectedCount)
	}
	if result.Distribution[0].UserID != freshUser {
		t.Fatalf("want freshUser in distribution, got %v", result.Distribution[0].UserID)
	}

	// Stale viewer should have received no points.
	staleBalance, err := pointsSvc.GetBalance(staleUser, "ch_stale")
	if err != nil {
		t.Fatalf("balance staleUser: %v", err)
	}
	if staleBalance.SpendableBalance != 0 {
		t.Fatalf("stale viewer should have 0 points, got %d", staleBalance.SpendableBalance)
	}
}

func TestAirdrop_TodayTotal_UTCDayBoundary(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	configSvc := NewChannelConfigService(db)
	svc := NewAirdropService(db, pointsSvc, configSvc)

	channelID := "ch_boundary"
	userID := seedAirdropViewer(t, db, channelID, 60)

	// Obtain or create the ledger so we can insert transactions directly.
	ledgerID := uuid.New()
	if err := db.Exec(
		`INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
		 VALUES (?, ?, ?, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		ledgerID, userID, channelID,
	).Error; err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	now := time.Now().UTC()
	utcMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Transaction created 1 second before UTC midnight — should NOT count today.
	yesterdayTs := utcMidnight.Add(-1 * time.Second)
	if err := db.Exec(
		`INSERT INTO points_transactions (ledger_id, source, delta, balance_after, created_at)
		 VALUES (?, ?, 200, 200, ?)`,
		ledgerID, models.TxSourceAirdrop, yesterdayTs,
	).Error; err != nil {
		t.Fatalf("seed yesterday tx: %v", err)
	}

	// Transaction created 1 second after UTC midnight — should count today.
	todayTs := utcMidnight.Add(1 * time.Second)
	if err := db.Exec(
		`INSERT INTO points_transactions (ledger_id, source, delta, balance_after, created_at)
		 VALUES (?, ?, 300, 500, ?)`,
		ledgerID, models.TxSourceAirdrop, todayTs,
	).Error; err != nil {
		t.Fatalf("seed today tx: %v", err)
	}

	total, err := svc.TodayTotal(channelID)
	if err != nil {
		t.Fatalf("TodayTotal: %v", err)
	}
	if total != 300 {
		t.Fatalf("want TodayTotal 300 (today only), got %d", total)
	}
}

func newConcurrentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := t.TempDir() + "/airdrop-concurrency.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open concurrent test db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(8)

	pragmas := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
	}
	for _, stmt := range pragmas {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("apply pragma %q: %v", stmt, err)
		}
	}

	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate concurrent test db: %v", err)
	}

	return db
}
