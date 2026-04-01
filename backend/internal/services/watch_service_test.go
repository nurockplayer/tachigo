package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
)

// seedWatchUser inserts a minimal users row so FK constraints pass in watch tests.
func seedWatchUser(t *testing.T, svc *WatchService) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := svc.db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

// backdateHeartbeat manually sets last_heartbeat_at to simulate elapsed time.
func backdateHeartbeat(t *testing.T, svc *WatchService, sessionID uuid.UUID, ago time.Duration) {
	t.Helper()
	ts := time.Now().Add(-ago)
	if err := svc.db.Model(&models.WatchSession{}).
		Where("id = ?", sessionID).
		Update("last_heartbeat_at", ts).Error; err != nil {
		t.Fatalf("backdate heartbeat: %v", err)
	}
}

// reloadSession fetches the latest DB state of a session.
func reloadSession(t *testing.T, svc *WatchService, id uuid.UUID) *models.WatchSession {
	t.Helper()
	var s models.WatchSession
	if err := svc.db.First(&s, "id = ?", id).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	return &s
}

// ─── StartSession ─────────────────────────────────────────────────────────────

func TestStartSession_CreatesNewSession(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	s, err := svc.StartSession(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.UserID != userID {
		t.Errorf("user_id: want %s, got %s", userID, s.UserID)
	}
	if s.ChannelID != "ch_abc" {
		t.Errorf("channel_id: want ch_abc, got %s", s.ChannelID)
	}
	if !s.IsActive {
		t.Error("expected is_active = true")
	}
	if s.EndedAt != nil {
		t.Error("expected ended_at = nil")
	}
}

func TestStartSession_ReturnsExistingActive(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	s1, _ := svc.StartSession(userID, "ch_abc")
	s2, err := svc.StartSession(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1.ID != s2.ID {
		t.Errorf("expected same session ID, got %s vs %s", s1.ID, s2.ID)
	}
}

func TestStartSession_ClosesStaleAndCreatesNew(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	s1, _ := svc.StartSession(userID, "ch_abc")
	backdateHeartbeat(t, svc, s1.ID, 3*time.Minute) // exceed staleThreshold (2 min)

	s2, err := svc.StartSession(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1.ID == s2.ID {
		t.Error("expected new session after stale close")
	}

	old := reloadSession(t, svc, s1.ID)
	if old.IsActive {
		t.Error("expected old session is_active = false")
	}
	if old.EndedAt == nil {
		t.Error("expected old session ended_at to be set")
	}
}

func TestStartSession_DifferentChannels(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	sA, _ := svc.StartSession(userID, "ch_A")
	sB, _ := svc.StartSession(userID, "ch_B")

	if sA.ID == sB.ID {
		t.Error("expected different sessions for different channels")
	}
}

// ─── Heartbeat ───────────────────────────────────────────────────────────────

func TestHeartbeat_NoActiveSession(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	_, err := svc.Heartbeat(userID, "ch_abc")
	if err != ErrNoActiveSession {
		t.Errorf("want ErrNoActiveSession, got %v", err)
	}
}

func TestHeartbeat_IgnoresFastHeartbeat(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	svc.StartSession(userID, "ch_abc")

	// Heartbeat immediately (< 20 s elapsed) — should be a no-op.
	result, err := svc.Heartbeat(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PointsEarned != 0 {
		t.Errorf("expected 0 points for fast heartbeat, got %d", result.PointsEarned)
	}
	if result.Session.AccumulatedSeconds != 0 {
		t.Errorf("expected accumulated_seconds unchanged (0), got %d", result.Session.AccumulatedSeconds)
	}
}

func TestHeartbeat_AccumulatesSecondsNoBelowMinute(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	s, _ := svc.StartSession(userID, "ch_abc")
	backdateHeartbeat(t, svc, s.ID, 25*time.Second)

	result, err := svc.Heartbeat(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Session.AccumulatedSeconds <= 0 {
		t.Error("expected accumulated_seconds > 0")
	}
	if result.PointsEarned != 0 {
		t.Errorf("expected 0 points (< 60 s), got %d", result.PointsEarned)
	}
}

func TestHeartbeat_AwardsPointsAt60Seconds(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)
	channelID := "ch_abc"

	// Three heartbeats of ~25 s each → 75 s accumulated → 1 point.
	s, _ := svc.StartSession(userID, channelID)
	backdateHeartbeat(t, svc, s.ID, 25*time.Second)
	svc.Heartbeat(userID, channelID) // +25 s = 25

	s = reloadSession(t, svc, s.ID)
	backdateHeartbeat(t, svc, s.ID, 25*time.Second)
	svc.Heartbeat(userID, channelID) // +25 s = 50

	s = reloadSession(t, svc, s.ID)
	backdateHeartbeat(t, svc, s.ID, 25*time.Second)
	result, err := svc.Heartbeat(userID, channelID) // +25 s = 75 → 1 point
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PointsEarned != 1 {
		t.Errorf("expected 1 point at 75 s, got %d", result.PointsEarned)
	}

	spendable, cumulative, _ := svc.GetBalance(userID, channelID)
	if spendable != 1 || cumulative != 1 {
		t.Errorf("balance: want (1,1), got (%d,%d)", spendable, cumulative)
	}
}

func TestHeartbeat_CapsLargeGapAt30Seconds(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	s, _ := svc.StartSession(userID, "ch_abc")
	backdateHeartbeat(t, svc, s.ID, 10*time.Minute) // simulate long disconnect

	result, err := svc.Heartbeat(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Session.AccumulatedSeconds > 30 {
		t.Errorf("expected delta capped at 30 s, got %d accumulated", result.Session.AccumulatedSeconds)
	}
}

// ─── EndSession ──────────────────────────────────────────────────────────────

func TestEndSession_ClosesActiveSession(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	s, _ := svc.StartSession(userID, "ch_abc")
	if err := svc.EndSession(userID, "ch_abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	closed := reloadSession(t, svc, s.ID)
	if closed.IsActive {
		t.Error("expected is_active = false after EndSession")
	}
	if closed.EndedAt == nil {
		t.Error("expected ended_at to be set")
	}
}

func TestEndSession_Idempotent(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	// No session exists — should not error.
	if err := svc.EndSession(userID, "ch_abc"); err != nil {
		t.Errorf("unexpected error with no session: %v", err)
	}

	// Call twice after creating a session — should not error.
	svc.StartSession(userID, "ch_abc")
	svc.EndSession(userID, "ch_abc")
	if err := svc.EndSession(userID, "ch_abc"); err != nil {
		t.Errorf("unexpected error on second EndSession: %v", err)
	}
}

// ─── GetBalance ──────────────────────────────────────────────────────────────

func TestGetBalance_NoLedger(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	spendable, cumulative, err := svc.GetBalance(userID, "ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spendable != 0 || cumulative != 0 {
		t.Errorf("want (0,0), got (%d,%d)", spendable, cumulative)
	}
}

func TestGetBalance_PerChannelIsolation(t *testing.T) {
	svc := NewWatchService(newTestDB(t))
	userID := seedWatchUser(t, svc)

	// Earn 1 point on ch_A only (three 25 s heartbeats = 75 s).
	sA, _ := svc.StartSession(userID, "ch_A")
	for i := 0; i < 3; i++ {
		sA = reloadSession(t, svc, sA.ID)
		backdateHeartbeat(t, svc, sA.ID, 25*time.Second)
		svc.Heartbeat(userID, "ch_A")
	}

	spA, _, _ := svc.GetBalance(userID, "ch_A")
	spB, _, _ := svc.GetBalance(userID, "ch_B")

	if spA == 0 {
		t.Error("expected non-zero balance for ch_A after earning points")
	}
	if spB != 0 {
		t.Errorf("expected 0 balance for ch_B (never watched), got %d", spB)
	}
}
