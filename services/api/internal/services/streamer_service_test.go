package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

type streamerContextKey string

func seedStreamerUserRow(t *testing.T, db *gorm.DB, role models.UserRole) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, ?, TRUE, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id, role,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func seedTwitchAuthProvider(t *testing.T, db *gorm.DB, userID uuid.UUID, channelID string) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), userID, models.ProviderTwitch, channelID,
	).Error; err != nil {
		t.Fatalf("seed auth provider: %v", err)
	}
}

func TestRegisterContext_UsesRequestContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_ctx_register")
	key := streamerContextKey("register-context")
	seen := installDBContextProbe(t, db, key, "streamer-register")

	if _, err := svc.RegisterContext(context.WithValue(context.Background(), key, "streamer-register"), userID, "ch_ctx_register", "Ctx"); err != nil {
		t.Fatalf("register with context: %v", err)
	}
	if seen() == 0 {
		t.Fatal("expected RegisterContext DB operations to carry request context")
	}
}

func TestRegister_OK(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	svc := NewStreamerService(db, pointsSvc)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_abc")

	streamer, err := svc.Register(userID, "ch_abc", "Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streamer.UserID != userID || streamer.ChannelID != "ch_abc" {
		t.Fatalf("unexpected streamer: %+v", streamer)
	}

	streamer2, err := svc.Register(userID, "ch_abc", "Alice")
	if err != nil {
		t.Fatalf("unexpected error on second register: %v", err)
	}
	if streamer2.ID != streamer.ID {
		t.Fatalf("expected upsert to reuse row, got %s vs %s", streamer2.ID, streamer.ID)
	}
}

func TestRegister_UpdateDisplayName(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_abc")

	if _, err := svc.Register(userID, "ch_abc", "Old"); err != nil {
		t.Fatalf("register old: %v", err)
	}
	streamer, err := svc.Register(userID, "ch_abc", "New")
	if err != nil {
		t.Fatalf("register new: %v", err)
	}
	if streamer.DisplayName != "New" {
		t.Fatalf("display_name: want New, got %q", streamer.DisplayName)
	}
}

func TestOwnsChannelContext_UsesRequestContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_ctx_owns")
	if _, err := svc.Register(userID, "ch_ctx_owns", "Ctx Owns"); err != nil {
		t.Fatalf("register owned channel: %v", err)
	}
	key := streamerContextKey("owns-channel-context")
	seen := installDBContextProbe(t, db, key, "streamer-owns")

	owns, err := svc.OwnsChannelContext(context.WithValue(context.Background(), key, "streamer-owns"), userID, "ch_ctx_owns")
	if err != nil {
		t.Fatalf("owns channel with context: %v", err)
	}
	if !owns {
		t.Fatal("expected streamer to own channel")
	}
	if seen() == 0 {
		t.Fatal("expected OwnsChannelContext DB operations to carry request context")
	}
}

func TestListChannels_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)

	channels, err := svc.ListChannels(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 0 {
		t.Fatalf("want empty slice, got %d items", len(channels))
	}
}

func TestListChannels_MultipleChannels(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_A")
	seedTwitchAuthProvider(t, db, userID, "ch_B")

	if _, err := svc.Register(userID, "ch_A", "Alpha"); err != nil {
		t.Fatalf("register ch_A: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := svc.Register(userID, "ch_B", "Beta"); err != nil {
		t.Fatalf("register ch_B: %v", err)
	}

	channels, err := svc.ListChannels(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("want 2 channels, got %d", len(channels))
	}
}

func TestListAllContext_UsesRequestContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_ctx_list_all")
	if _, err := svc.Register(userID, "ch_ctx_list_all", "Ctx List"); err != nil {
		t.Fatalf("register list row: %v", err)
	}
	key := streamerContextKey("list-all-context")
	seen := installDBContextProbe(t, db, key, "streamer-list-all")

	streamers, err := svc.ListAllContext(context.WithValue(context.Background(), key, "streamer-list-all"))
	if err != nil {
		t.Fatalf("list all with context: %v", err)
	}
	if len(streamers) != 1 {
		t.Fatalf("expected 1 streamer, got %d", len(streamers))
	}
	if seen() == 0 {
		t.Fatal("expected ListAllContext DB operations to carry request context")
	}
}

func TestGetSummaryStatsContext_UsesRequestContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))
	key := streamerContextKey("summary-context")
	seen := installDBContextProbe(t, db, key, "streamer-summary")

	stats, err := svc.GetSummaryStatsContext(context.WithValue(context.Background(), key, "streamer-summary"), []string{"ch_ctx_summary"})
	if err != nil {
		t.Fatalf("summary stats with context: %v", err)
	}
	if _, ok := stats["ch_ctx_summary"]; !ok {
		t.Fatal("expected initialized summary entry for channel")
	}
	if seen() == 0 {
		t.Fatal("expected GetSummaryStatsContext DB operations to carry request context")
	}
}

func TestGetChannelStats_NoStreamer(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))

	_, err := svc.GetChannelStats("missing_channel")
	if !errors.Is(err, ErrStreamerNotFound) {
		t.Fatalf("want ErrStreamerNotFound, got %v", err)
	}
}

func TestGetStatsContext_UsesRequestContext(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	svc := NewStreamerService(db, pointsSvc)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, userID, "ch_ctx_stats")
	streamer, err := svc.Register(userID, "ch_ctx_stats", "Ctx Stats")
	if err != nil {
		t.Fatalf("register stats row: %v", err)
	}
	key := streamerContextKey("stats-context")
	seen := installDBContextProbe(t, db, key, "streamer-stats")

	if _, err := svc.GetStatsContext(context.WithValue(context.Background(), key, "streamer-stats"), streamer.ID); err != nil {
		t.Fatalf("get stats with context: %v", err)
	}
	if seen() == 0 {
		t.Fatal("expected GetStatsContext DB operations to carry request context")
	}
}

func TestGetChannelStats_OK(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	svc := NewStreamerService(db, pointsSvc)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)

	if err := db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), userID, models.ProviderTwitch, "ch_stats",
	).Error; err != nil {
		t.Fatalf("seed auth provider: %v", err)
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	entries := []time.Time{
		now,
		startOfDay.Add(-time.Hour),
		startOfMonth.Add(-time.Hour),
	}
	var wantDaily, wantMonthly, wantYearly int64
	for _, recordedAt := range entries {
		if !recordedAt.Before(startOfDay) {
			wantDaily += 30
		}
		if !recordedAt.Before(startOfMonth) {
			wantMonthly += 30
		}
		if !recordedAt.Before(startOfYear) {
			wantYearly += 30
		}
		if err := db.Create(&models.BroadcastTimeLog{
			StreamerID: userID,
			ChannelID:  "ch_stats",
			Seconds:    30,
			RecordedAt: recordedAt,
		}).Error; err != nil {
			t.Fatalf("seed broadcast log: %v", err)
		}
	}

	viewerID := seedStreamerUserRow(t, db, models.RoleViewer)
	if _, err := watchSvc.StartSession(viewerID, "ch_stats"); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if err := db.Model(&models.WatchSession{}).
		Where("user_id = ? AND channel_id = ?", viewerID, "ch_stats").
		Update("accumulated_seconds", 45).Error; err != nil {
		t.Fatalf("seed active session seconds: %v", err)
	}

	stats, err := svc.GetChannelStats("ch_stats")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CurrentSessionSeconds != 45 {
		t.Fatalf("current_session_seconds: want 45, got %d", stats.CurrentSessionSeconds)
	}
	if stats.DailySeconds != wantDaily {
		t.Fatalf("daily_seconds: want %d, got %d", wantDaily, stats.DailySeconds)
	}
	if stats.MonthlySeconds != wantMonthly {
		t.Fatalf("monthly_seconds: want %d, got %d", wantMonthly, stats.MonthlySeconds)
	}
	if stats.YearlySeconds != wantYearly {
		t.Fatalf("yearly_seconds: want %d, got %d", wantYearly, stats.YearlySeconds)
	}
}
