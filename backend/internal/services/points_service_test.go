package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
)

// newPointsSvc creates a PointsService backed by an in-memory SQLite test DB.
func newPointsSvc(t *testing.T) (*PointsService, *WatchService) {
	t.Helper()
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	return pointsSvc, watchSvc
}

// seedViewer inserts a viewer user and returns their UUID.
func seedViewer(t *testing.T, svc *PointsService) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := svc.db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, id,
	).Error; err != nil {
		t.Fatalf("seed viewer: %v", err)
	}
	return id
}

// seedStreamer inserts a streamer user linked to a Twitch channel and returns their UUID.
func seedStreamer(t *testing.T, svc *PointsService, channelID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := svc.db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'streamer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, id,
	).Error; err != nil {
		t.Fatalf("seed streamer user: %v", err)
	}
	providerID := uuid.New()
	if err := svc.db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, 'twitch', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		providerID, id, channelID,
	).Error; err != nil {
		t.Fatalf("seed streamer auth_provider: %v", err)
	}
	return id
}

// ─── GetBalance ──────────────────────────────────────────────────────────────

func TestPointsService_GetBalance_NoLedger(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	bal, err := svc.GetBalance(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bal.SpendableBalance != 0 || bal.CumulativeTotal != 0 {
		t.Errorf("want (0,0), got (%d,%d)", bal.SpendableBalance, bal.CumulativeTotal)
	}
}

func TestPointsService_GetBalance_AfterEarning(t *testing.T) {
	svc, watchSvc := newPointsSvc(t)
	userID := seedViewer(t, svc)

	// Earn 1 point via WatchService (3 × 25 s heartbeats = 75 s → 1 pt)
	s, _ := watchSvc.StartSession(userID, "ch_abc")
	for i := 0; i < 3; i++ {
		s = reloadSession(t, watchSvc, s.ID)
		backdateHeartbeat(t, watchSvc, s.ID, 25*time.Second)
		watchSvc.Heartbeat(userID, "ch_abc")
	}

	bal, err := svc.GetBalance(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bal.SpendableBalance != 1 || bal.CumulativeTotal != 1 {
		t.Errorf("want (1,1), got (%d,%d)", bal.SpendableBalance, bal.CumulativeTotal)
	}
}

// ─── DeductPoints ────────────────────────────────────────────────────────────

func TestPointsService_DeductPoints_InsufficientBalance(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	err := svc.DeductPoints(userID, "ch_abc", 100, "test spend")
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("want ErrInsufficientBalance, got %v", err)
	}
}

func TestPointsService_DeductPoints_Success(t *testing.T) {
	svc, watchSvc := newPointsSvc(t)
	userID := seedViewer(t, svc)

	// Earn 2 points first
	s, _ := watchSvc.StartSession(userID, "ch_abc")
	for i := 0; i < 5; i++ {
		s = reloadSession(t, watchSvc, s.ID)
		backdateHeartbeat(t, watchSvc, s.ID, 25*time.Second)
		watchSvc.Heartbeat(userID, "ch_abc")
	}

	if err := svc.DeductPoints(userID, "ch_abc", 1, "redeem"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bal, _ := svc.GetBalance(userID, "ch_abc")
	if bal.SpendableBalance != 1 {
		t.Errorf("spendable: want 1, got %d", bal.SpendableBalance)
	}
	// cumulative_total must not change
	if bal.CumulativeTotal != 2 {
		t.Errorf("cumulative: want 2 (unchanged), got %d", bal.CumulativeTotal)
	}
}

// ─── AddPoints ───────────────────────────────────────────────────────────────

func TestPointsService_AddPoints_IncreasesBothBalances(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	if err := svc.AddPoints(userID, "ch_abc", models.TxSourceBits, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bal, _ := svc.GetBalance(userID, "ch_abc")
	if bal.SpendableBalance != 500 || bal.CumulativeTotal != 500 {
		t.Errorf("want (500,500), got (%d,%d)", bal.SpendableBalance, bal.CumulativeTotal)
	}
}

func TestPointsService_AddPointsWithMeta_PersistsSKU(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)
	sku := "bits_100"

	if err := svc.AddPointsWithMeta(
		userID,
		"ch_abc",
		models.TxSourceBits,
		100,
		PointsCreditMeta{SKU: &sku},
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	txs, err := svc.ListTransactions(userID, "ch_abc")
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("want 1 transaction, got %d", len(txs))
	}
	if txs[0].SKU == nil || *txs[0].SKU != "bits_100" {
		t.Fatalf("want sku bits_100, got %#v", txs[0].SKU)
	}
}

// ─── ListTransactions ────────────────────────────────────────────────────────

func TestPointsService_ListTransactions_EmptyWhenNoLedger(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	txs, err := svc.ListTransactions(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("want 0 transactions, got %d", len(txs))
	}
}

func TestPointsService_ListTransactions_RecordsDeduction(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	svc.AddPoints(userID, "ch_abc", models.TxSourceBits, 100)
	svc.DeductPoints(userID, "ch_abc", 30, "spend test")

	txs, err := svc.ListTransactions(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("want 2 transactions, got %d", len(txs))
	}
}

func TestPointsService_ListTransactions_IsScopedToRequestedChannel(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	if err := svc.AddPoints(userID, "ch_abc", models.TxSourceBits, 100); err != nil {
		t.Fatalf("seed ch_abc: %v", err)
	}
	if err := svc.AddPoints(userID, "ch_other", models.TxSourceBits, 999); err != nil {
		t.Fatalf("seed ch_other: %v", err)
	}

	txs, err := svc.ListTransactions(userID, "ch_abc")
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("want 1 transaction for ch_abc, got %d", len(txs))
	}
	if txs[0].Delta != 100 {
		t.Fatalf("want ch_abc delta 100, got %d", txs[0].Delta)
	}
	if txs[0].BalanceAfter != 100 {
		t.Fatalf("want ch_abc balance_after 100, got %d", txs[0].BalanceAfter)
	}
}

func TestPointsService_ListTransactions_ReturnsLatest50NewestFirst(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	if err := svc.AddPoints(userID, "ch_abc", models.TxSourceBits, 1); err != nil {
		t.Fatalf("seed first point: %v", err)
	}

	var ledger models.PointsLedger
	if err := svc.db.Where("user_id = ? AND channel_id = ?", userID, "ch_abc").First(&ledger).Error; err != nil {
		t.Fatalf("get ledger: %v", err)
	}
	if err := svc.db.Where("ledger_id = ?", ledger.ID).Delete(&models.PointsTransaction{}).Error; err != nil {
		t.Fatalf("clear bootstrap transaction: %v", err)
	}

	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 54; i++ {
		ts := base.Add(time.Duration(i) * time.Second)
		if err := svc.db.Exec(
			`INSERT INTO points_transactions (id, ledger_id, source, delta, balance_after, created_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), ledger.ID, string(models.TxSourceBits), 1, int64(i+1), ts,
		).Error; err != nil {
			t.Fatalf("seed tx %d: %v", i, err)
		}
	}

	txs, err := svc.ListTransactions(userID, "ch_abc")
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txs) != 50 {
		t.Fatalf("want 50 transactions, got %d", len(txs))
	}

	for i := 1; i < len(txs); i++ {
		left := txs[i-1]
		right := txs[i]
		if left.CreatedAt.Before(right.CreatedAt) {
			t.Fatalf("transactions not sorted descending at index %d: %v < %v", i, left.CreatedAt, right.CreatedAt)
		}
	}
}

func TestPointsService_ListTransactions_UsesIDAsTieBreakerWhenCreatedAtMatches(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	if err := svc.AddPoints(userID, "ch_abc", models.TxSourceBits, 1); err != nil {
		t.Fatalf("seed first point: %v", err)
	}

	var ledger models.PointsLedger
	if err := svc.db.Where("user_id = ? AND channel_id = ?", userID, "ch_abc").First(&ledger).Error; err != nil {
		t.Fatalf("get ledger: %v", err)
	}
	if err := svc.db.Where("ledger_id = ?", ledger.ID).Delete(&models.PointsTransaction{}).Error; err != nil {
		t.Fatalf("clear bootstrap transaction: %v", err)
	}

	createdAt := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	lowID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	highID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	lowNote := "low-id"
	highNote := "high-id"

	for _, tx := range []models.PointsTransaction{
		{
			ID:           lowID,
			LedgerID:     ledger.ID,
			Source:       models.TxSourceBits,
			Delta:        1,
			BalanceAfter: 1,
			Note:         &lowNote,
			CreatedAt:    createdAt,
		},
		{
			ID:           highID,
			LedgerID:     ledger.ID,
			Source:       models.TxSourceBits,
			Delta:        1,
			BalanceAfter: 2,
			Note:         &highNote,
			CreatedAt:    createdAt,
		},
	} {
		tx := tx
		if err := svc.db.Create(&tx).Error; err != nil {
			t.Fatalf("create transaction %s: %v", tx.ID, err)
		}
	}

	txs, err := svc.ListTransactions(userID, "ch_abc")
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("want 2 transactions, got %d", len(txs))
	}
	if txs[0].Note == nil || *txs[0].Note != "high-id" {
		t.Fatalf("want first transaction high-id, got %#v", txs[0].Note)
	}
	if txs[1].Note == nil || *txs[1].Note != "low-id" {
		t.Fatalf("want second transaction low-id, got %#v", txs[1].Note)
	}
}

// ─── AddWatchTime / GetWatchStats ────────────────────────────────────────────

func TestPointsService_AddWatchTime_Accumulates(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	svc.AddWatchTime(userID, "ch_abc", 30)
	svc.AddWatchTime(userID, "ch_abc", 45)

	stats, err := svc.GetWatchStats(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalWatchSeconds != 75 {
		t.Errorf("want 75 s, got %d", stats.TotalWatchSeconds)
	}
}

func TestPointsService_GetWatchStats_ZeroWhenNone(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)

	stats, err := svc.GetWatchStats(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalWatchSeconds != 0 {
		t.Errorf("want 0 s, got %d", stats.TotalWatchSeconds)
	}
}

// ─── AddBroadcastTime ────────────────────────────────────────────────────────

func TestPointsService_AddHeartbeatTime_RollsBackWhenBroadcastWriteFails(t *testing.T) {
	svc, _ := newPointsSvc(t)
	userID := seedViewer(t, svc)
	channelID := "ch_atomic"
	seedStreamer(t, svc, channelID)

	if err := svc.db.Exec(`DROP TABLE broadcast_time_logs`).Error; err != nil {
		t.Fatalf("drop broadcast_time_logs: %v", err)
	}

	if err := svc.AddHeartbeatTime(userID, channelID, 30); err == nil {
		t.Fatal("expected AddHeartbeatTime to fail when broadcast log write fails")
	}

	stats, err := svc.GetWatchStats(userID, channelID)
	if err != nil {
		t.Fatalf("unexpected watch stats error: %v", err)
	}
	if stats.TotalWatchSeconds != 0 {
		t.Fatalf("expected watch_time_stats rollback, got %d", stats.TotalWatchSeconds)
	}

	var broadcastCount int64
	if err := svc.db.Raw(
		`SELECT COUNT(*) FROM broadcast_time_stats WHERE channel_id = ?`,
		channelID,
	).Scan(&broadcastCount).Error; err != nil {
		t.Fatalf("count broadcast_time_stats: %v", err)
	}
	if broadcastCount != 0 {
		t.Fatalf("expected broadcast_time_stats rollback, got %d rows", broadcastCount)
	}
}

func TestPointsService_AddBroadcastTime_NoOpWhenStreamerNotRegistered(t *testing.T) {
	svc, _ := newPointsSvc(t)

	// No auth_provider for this channelID — should silently return nil
	err := svc.AddBroadcastTime("ch_no_streamer", 30)
	if err != nil {
		t.Errorf("expected no error for unregistered channel, got %v", err)
	}

	var count int64
	svc.db.Raw("SELECT COUNT(*) FROM broadcast_time_logs").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 log entries, got %d", count)
	}
}

func TestPointsService_AddBroadcastTime_WritesLogAndStat(t *testing.T) {
	svc, _ := newPointsSvc(t)
	channelID := "ch_with_streamer"
	streamerID := seedStreamer(t, svc, channelID)

	if err := svc.AddBroadcastTime(channelID, 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := svc.AddBroadcastTime(channelID, 45); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// broadcast_time_stats should have lifetime total
	var total int64
	svc.db.Raw("SELECT total_broadcast_seconds FROM broadcast_time_stats WHERE streamer_id = ?", streamerID).Scan(&total)
	if total != 75 {
		t.Errorf("broadcast_time_stats: want 75 s, got %d", total)
	}

	// broadcast_time_logs should have 2 entries
	var count int64
	svc.db.Raw("SELECT COUNT(*) FROM broadcast_time_logs WHERE streamer_id = ?", streamerID).Scan(&count)
	if count != 2 {
		t.Errorf("broadcast_time_logs: want 2 entries, got %d", count)
	}
}

// ─── GetBroadcastStats ───────────────────────────────────────────────────────

func TestPointsService_GetBroadcastStats_TimeWindows(t *testing.T) {
	svc, _ := newPointsSvc(t)
	channelID := "ch_stats"
	streamerID := seedStreamer(t, svc, channelID)

	// Insert log entries directly with controlled recorded_at timestamps
	now := time.Now()
	entries := []struct {
		seconds    int64
		recordedAt time.Time
	}{
		{60, now.Add(-10 * time.Minute)}, // today
		{120, now.Add(-25 * time.Hour)},  // yesterday (outside daily, inside monthly)
		{180, now.AddDate(0, -1, -1)},    // last month (outside monthly, inside yearly)
	}
	for _, e := range entries {
		svc.db.Exec(
			`INSERT INTO broadcast_time_logs (id, streamer_id, channel_id, seconds, recorded_at) VALUES (?, ?, ?, ?, ?)`,
			uuid.New(), streamerID, channelID, e.seconds, e.recordedAt,
		)
	}

	stats, err := svc.GetBroadcastStats(streamerID, channelID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.DailySeconds != 60 {
		t.Errorf("daily: want 60, got %d", stats.DailySeconds)
	}
	if stats.MonthlySeconds != 60+120 {
		t.Errorf("monthly: want 180, got %d", stats.MonthlySeconds)
	}
	if stats.YearlySeconds != 60+120+180 {
		t.Errorf("yearly: want 360, got %d", stats.YearlySeconds)
	}
}
