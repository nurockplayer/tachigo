package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// captureMailer records the most recent Send call via a buffered channel.
type captureMailer struct {
	ch chan sentEmail
}

type sentEmail struct {
	to, subject, body string
}

func newCaptureMailer() *captureMailer {
	return &captureMailer{ch: make(chan sentEmail, 1)}
}

func (m *captureMailer) Send(to, subject, body string) error {
	m.ch <- sentEmail{to, subject, body}
	return nil
}

// expectEmail waits up to 2 s for an email and returns it.
func (m *captureMailer) expectEmail(t *testing.T) sentEmail {
	t.Helper()
	select {
	case e := <-m.ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: no email received")
		return sentEmail{}
	}
}

// noEmailMailer fails the test if Send is ever called.
type noEmailMailer struct{ t *testing.T }

func (m *noEmailMailer) Send(_, _, _ string) error {
	m.t.Error("Send called unexpectedly")
	return nil
}

// ── seed helpers ──────────────────────────────────────────────────────────────

func seedUserWithEmail(t *testing.T, db *gorm.DB, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(`
		INSERT INTO users (id, username, email, role, is_active, email_verified, created_at, updated_at)
		VALUES (?, ?, ?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id.String(), "user_"+id.String()[:8], email).Error; err != nil {
		t.Fatalf("seedUserWithEmail: %v", err)
	}
	return id
}

func seedUserWithoutEmail(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(`
		INSERT INTO users (id, username, role, is_active, email_verified, created_at, updated_at)
		VALUES (?, ?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id.String(), "user_"+id.String()[:8]).Error; err != nil {
		t.Fatalf("seedUserWithoutEmail: %v", err)
	}
	return id
}

func seedRaffle(t *testing.T, db *gorm.DB, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(`
		INSERT INTO raffles (id, user_id, title, status, source, created_at, updated_at)
		VALUES (?, ?, 'Test Raffle', 'active', 'csv', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id.String(), ownerID.String()).Error; err != nil {
		t.Fatalf("seedRaffle: %v", err)
	}
	return id
}

func seedEntry(t *testing.T, db *gorm.DB, raffleID uuid.UUID, userID *uuid.UUID, login string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var uid interface{}
	if userID != nil {
		uid = userID.String()
	}
	if err := db.Exec(`
		INSERT INTO raffle_entries (id, raffle_id, user_id, twitch_login, display_name, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, id.String(), raffleID.String(), uid, login, login).Error; err != nil {
		t.Fatalf("seedEntry: %v", err)
	}
	return id
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestDrawNext_SendsEmailToWinner(t *testing.T) {
	db := newTestDB(t)
	mailer := newCaptureMailer()

	ownerID := seedUserWithEmail(t, db, "owner@example.com")
	winnerID := seedUserWithEmail(t, db, "winner@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "winner_twitch")

	svc := &RaffleService{db: db, mailer: mailer, frontendURL: "http://localhost:3000"}

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw, got nil")
	}

	email := mailer.expectEmail(t)
	if email.to != "winner@example.com" {
		t.Errorf("expected to=winner@example.com, got %s", email.to)
	}
	if !strings.Contains(email.body, draw.ClaimToken) {
		t.Errorf("email body should contain claim token")
	}
	if !strings.Contains(email.body, "http://localhost:3000/claim/") {
		t.Errorf("email body should contain claim link")
	}
}

func TestDrawNext_SkipsEmailWhenNoUserID(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "owner@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, nil, "anonymous_twitch") // no linked account

	svc := &RaffleService{db: db, mailer: &noEmailMailer{t: t}, frontendURL: "http://localhost:3000"}

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw, got nil")
	}
	// noEmailMailer will call t.Error if Send is called; give goroutine time to run
	time.Sleep(50 * time.Millisecond)
}

func TestDrawNext_SkipsEmailWhenNoEmail(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "owner@example.com")
	winnerID := seedUserWithoutEmail(t, db)
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "noemail_twitch")

	svc := &RaffleService{db: db, mailer: &noEmailMailer{t: t}, frontendURL: "http://localhost:3000"}

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw, got nil")
	}
	time.Sleep(50 * time.Millisecond)
}

func TestDrawNext_EmailFailureDoesNotBlockDraw(t *testing.T) {
	db := newTestDB(t)
	// errMailer always returns an error
	errMailer := &errSendMailer{}
	ownerID := seedUserWithEmail(t, db, "owner@example.com")
	winnerID := seedUserWithEmail(t, db, "winner@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "winner_twitch")

	svc := &RaffleService{db: db, mailer: errMailer, frontendURL: "http://localhost:3000"}

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext should succeed even if email fails: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw, got nil")
	}
}

func TestDrawNext_NoEmailWhenMailerNil(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "owner@example.com")
	winnerID := seedUserWithEmail(t, db, "winner@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "winner_twitch")

	svc := &RaffleService{db: db, mailer: nil} // no mailer configured

	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw == nil {
		t.Fatal("expected draw, got nil")
	}
}

// ── helper types ──────────────────────────────────────────────────────────────

type errSendMailer struct{}

func (m *errSendMailer) Send(_, _, _ string) error {
	return context.DeadlineExceeded
}
