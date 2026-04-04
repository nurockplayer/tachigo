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
		 VALUES (?, ?, 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id, role,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func TestRegister_OK(t *testing.T) {
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	svc := NewStreamerService(db, pointsSvc)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)

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
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
	startOfMonth := time.Date(now.Year(), now.Month(), 2, 0, 0, 0, 0, now.Location())
	startOfYear := time.Date(now.Year(), 2, 1, 0, 0, 0, 0, now.Location())

	for _, recordedAt := range []time.Time{startOfDay, startOfMonth, startOfYear} {
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
	if stats.DailySeconds != 30 {
		t.Fatalf("daily_seconds: want 30, got %d", stats.DailySeconds)
	}
	if stats.MonthlySeconds != 60 {
		t.Fatalf("monthly_seconds: want 60, got %d", stats.MonthlySeconds)
	}
	if stats.YearlySeconds != 90 {
		t.Fatalf("yearly_seconds: want 90, got %d", stats.YearlySeconds)
	}
}
