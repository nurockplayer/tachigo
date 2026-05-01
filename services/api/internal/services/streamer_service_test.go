package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

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

func TestGetChannelStats_NoStreamer(t *testing.T) {
	db := newTestDB(t)
	svc := NewStreamerService(db, NewPointsService(db, NewWatchService(db)))

	_, err := svc.GetChannelStats("missing_channel")
	if !errors.Is(err, ErrStreamerNotFound) {
		t.Fatalf("want ErrStreamerNotFound, got %v", err)
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
		now.Add(-10 * time.Minute),
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
