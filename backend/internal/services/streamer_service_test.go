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

func seedStreamerRecord(t *testing.T, db *gorm.DB, userID uuid.UUID, agencyUserID *uuid.UUID, twitchLogin string) *models.Streamer {
	t.Helper()
	streamer := &models.Streamer{
		UserID:       userID,
		AgencyUserID: agencyUserID,
		TwitchLogin:  twitchLogin,
	}
	if err := db.Create(streamer).Error; err != nil {
		t.Fatalf("seed streamer: %v", err)
	}
	return streamer
}

func newStreamerService(t *testing.T) (*gorm.DB, *StreamerService, *WatchService) {
	t.Helper()
	db := newTestDB(t)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	return db, NewStreamerService(db, pointsSvc), watchSvc
}

func TestCreate_OK(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	agencyUserID := seedStreamerUserRow(t, db, models.RoleAgency)
	seedTwitchAuthProvider(t, db, userID, "alice_login")

	streamer, err := svc.Create(userID, &agencyUserID, "alice_login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streamer.UserID != userID {
		t.Fatalf("user_id: want %s, got %s", userID, streamer.UserID)
	}
	if streamer.AgencyUserID == nil || *streamer.AgencyUserID != agencyUserID {
		t.Fatalf("agency_user_id mismatch: %+v", streamer.AgencyUserID)
	}
	if streamer.TwitchLogin != "alice_login" {
		t.Fatalf("twitch_login: want alice_login, got %q", streamer.TwitchLogin)
	}

	var saved models.Streamer
	if err := db.Where("id = ?", streamer.ID).First(&saved).Error; err != nil {
		t.Fatalf("load streamer: %v", err)
	}
}

func TestCreate_ChannelNotOwned(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)

	_, err := svc.Create(userID, nil, "missing_login")
	if !errors.Is(err, ErrChannelNotOwned) {
		t.Fatalf("want ErrChannelNotOwned, got %v", err)
	}
}

func TestCreate_Idempotent(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	agencyA := seedStreamerUserRow(t, db, models.RoleAgency)
	agencyB := seedStreamerUserRow(t, db, models.RoleAgency)
	seedTwitchAuthProvider(t, db, userID, "alice_login")

	first, err := svc.Create(userID, &agencyA, "alice_login")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.Create(userID, &agencyB, "alice_login")
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same id, got %s vs %s", first.ID, second.ID)
	}
	if second.AgencyUserID == nil || *second.AgencyUserID != agencyB {
		t.Fatalf("expected agency_user_id updated to %s, got %+v", agencyB, second.AgencyUserID)
	}
}

func TestStreamerGetByID_Found(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	streamer := seedStreamerRecord(t, db, userID, nil, "alice_login")

	got, err := svc.GetByID(streamer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != streamer.ID || got.UserID != userID || got.TwitchLogin != "alice_login" {
		t.Fatalf("unexpected streamer: %+v", got)
	}
}

func TestStreamerGetByID_NotFound(t *testing.T) {
	_, svc, _ := newStreamerService(t)

	_, err := svc.GetByID(uuid.New())
	if !errors.Is(err, ErrStreamerNotFound) {
		t.Fatalf("want ErrStreamerNotFound, got %v", err)
	}
}

func TestGetByUserID_Found(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedStreamerRecord(t, db, userID, nil, "alice_login")

	got, err := svc.GetByUserID(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != userID || got.TwitchLogin != "alice_login" {
		t.Fatalf("unexpected streamer: %+v", got)
	}
}

func TestGetByUserID_NotFound(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	userID := seedStreamerUserRow(t, db, models.RoleStreamer)

	_, err := svc.GetByUserID(userID)
	if !errors.Is(err, ErrStreamerNotFound) {
		t.Fatalf("want ErrStreamerNotFound, got %v", err)
	}
}

func TestListByAgencyUserID_FiltersCorrectly(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	agencyA := seedStreamerUserRow(t, db, models.RoleAgency)
	agencyB := seedStreamerUserRow(t, db, models.RoleAgency)
	streamerA1 := seedStreamerUserRow(t, db, models.RoleStreamer)
	streamerA2 := seedStreamerUserRow(t, db, models.RoleStreamer)
	streamerB1 := seedStreamerUserRow(t, db, models.RoleStreamer)

	seedStreamerRecord(t, db, streamerA1, &agencyA, "a1_login")
	seedStreamerRecord(t, db, streamerA2, &agencyA, "a2_login")
	seedStreamerRecord(t, db, streamerB1, &agencyB, "b1_login")

	streamers, err := svc.ListByAgencyUserID(agencyA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streamers) != 2 {
		t.Fatalf("want 2 streamers, got %d", len(streamers))
	}
	for _, streamer := range streamers {
		if streamer.AgencyUserID == nil || *streamer.AgencyUserID != agencyA {
			t.Fatalf("unexpected agency_user_id: %+v", streamer)
		}
	}
}

func TestListByAgencyUserID_Empty(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)

	streamers, err := svc.ListByAgencyUserID(agencyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streamers == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(streamers) != 0 {
		t.Fatalf("want empty slice, got %d items", len(streamers))
	}
}

func TestOwnsStreamer_True(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	streamerUserID := seedStreamerUserRow(t, db, models.RoleStreamer)
	streamer := seedStreamerRecord(t, db, streamerUserID, &agencyID, "owned_login")

	owns, err := svc.OwnsStreamer(agencyID, streamer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !owns {
		t.Fatal("expected owns=true")
	}
}

func TestOwnsStreamer_False(t *testing.T) {
	db, svc, _ := newStreamerService(t)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	otherAgencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	streamerUserID := seedStreamerUserRow(t, db, models.RoleStreamer)
	streamer := seedStreamerRecord(t, db, streamerUserID, &otherAgencyID, "other_login")

	owns, err := svc.OwnsStreamer(agencyID, streamer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owns {
		t.Fatal("expected owns=false")
	}
}

func TestGetStats_AllEightMetrics(t *testing.T) {
	db, svc, watchSvc := newStreamerService(t)
	streamerUserID := seedStreamerUserRow(t, db, models.RoleStreamer)
	seedTwitchAuthProvider(t, db, streamerUserID, "streamer_login")

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	logTimes := []time.Time{
		now,
		startOfDay.Add(-24 * time.Hour),
		startOfMonth.Add(-24 * time.Hour),
	}
	if !logTimes[1].Before(startOfMonth) {
		logTimes[1] = startOfMonth.Add(2 * time.Hour)
	}
	if !logTimes[2].Before(startOfYear) {
		logTimes[2] = startOfYear.Add(2 * time.Hour)
	}

	seconds := []int64{30, 30, 30}
	for i, recordedAt := range logTimes {
		if err := db.Create(&models.BroadcastTimeLog{
			StreamerID: streamerUserID,
			ChannelID:  "streamer_login",
			Seconds:    seconds[i],
			RecordedAt: recordedAt,
		}).Error; err != nil {
			t.Fatalf("seed broadcast log %d: %v", i, err)
		}
	}

	viewerID := seedStreamerUserRow(t, db, models.RoleViewer)
	if _, err := watchSvc.StartSession(viewerID, "streamer_login"); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if err := db.Model(&models.WatchSession{}).
		Where("user_id = ? AND channel_id = ?", viewerID, "streamer_login").
		Update("accumulated_seconds", 45).Error; err != nil {
		t.Fatalf("seed watch session: %v", err)
	}

	if err := db.Create(&models.PointsLedger{
		UserID:           viewerID,
		ChannelID:        "streamer_login",
		CumulativeTotal:  100,
		SpendableBalance: 60,
	}).Error; err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	expectedDaily := int64(0)
	expectedMonthly := int64(0)
	expectedYearly := int64(0)
	for i, recordedAt := range logTimes {
		if !recordedAt.Before(startOfDay) {
			expectedDaily += seconds[i]
		}
		if !recordedAt.Before(startOfMonth) {
			expectedMonthly += seconds[i]
		}
		if !recordedAt.Before(startOfYear) {
			expectedYearly += seconds[i]
		}
	}

	stats, err := svc.GetStats(streamerUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CurrentSessionSeconds != 45 {
		t.Fatalf("current_session_seconds: want 45, got %d", stats.CurrentSessionSeconds)
	}
	if stats.DailySeconds != expectedDaily {
		t.Fatalf("daily_seconds: want %d, got %d", expectedDaily, stats.DailySeconds)
	}
	if stats.MonthlySeconds != expectedMonthly {
		t.Fatalf("monthly_seconds: want %d, got %d", expectedMonthly, stats.MonthlySeconds)
	}
	if stats.YearlySeconds != expectedYearly {
		t.Fatalf("yearly_seconds: want %d, got %d", expectedYearly, stats.YearlySeconds)
	}
	if stats.UniqueMiners != 1 {
		t.Fatalf("unique_miners: want 1, got %d", stats.UniqueMiners)
	}
	if stats.AvgSessionSeconds != 45 {
		t.Fatalf("avg_session_seconds: want 45, got %v", stats.AvgSessionSeconds)
	}
	if stats.TotalTokenMinted != 100 {
		t.Fatalf("total_token_minted: want 100, got %d", stats.TotalTokenMinted)
	}
	if stats.SpendableInCirculation != 60 {
		t.Fatalf("spendable_in_circulation: want 60, got %d", stats.SpendableInCirculation)
	}
}

func TestGetStats_NoStreamer(t *testing.T) {
	_, svc, _ := newStreamerService(t)

	_, err := svc.GetStats(uuid.New())
	if !errors.Is(err, ErrStreamerNotFound) {
		t.Fatalf("want ErrStreamerNotFound, got %v", err)
	}
}
