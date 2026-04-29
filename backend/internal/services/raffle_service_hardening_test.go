package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

// TestDrawNext_SkipsAlreadyDrawnEntry verifies that if one entry is already
// drawn (seeded directly into DB), DrawNext picks the remaining entry rather
// than misreporting ErrRaffleExhausted.
func TestDrawNext_SkipsAlreadyDrawnEntry(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "hardening_owner1@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	entryAID := seedEntry(t, db, raffleID, nil, "player_a")
	seedEntry(t, db, raffleID, nil, "player_b")

	seedDraw(t, db, raffleID, entryAID, "pre-seeded-token-a")

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext should succeed when entries remain: %v", err)
	}
	if draw.Entry.TwitchLogin != "player_b" {
		t.Errorf("expected player_b to win, got %q", draw.Entry.TwitchLogin)
	}
}

// TestDrawNext_ExhaustedWhenAllDrawn verifies ErrRaffleExhausted is returned
// only when every entry has a corresponding draw.
func TestDrawNext_ExhaustedWhenAllDrawn(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "hardening_owner2@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	entryID := seedEntry(t, db, raffleID, nil, "only_player")
	seedDraw(t, db, raffleID, entryID, "existing-token")

	svc := &RaffleService{db: db}
	_, err := svc.DrawNext(raffleID, ownerID)
	if !errors.Is(err, ErrRaffleExhausted) {
		t.Errorf("expected ErrRaffleExhausted, got %v", err)
	}
}

// TestDrawNext_Concurrent_OneWinner is a behavioural regression test: two
// goroutines race on a single-entry raffle and exactly one wins.
// NOTE: SetMaxOpenConns(1) serialises the goroutines in SQLite, so this test
// validates clean exhaustion but does NOT exercise the duplicate-key retry path.
// See TestDrawNext_RetriesOnDuplicateKeyConflict for that path.
//
// SQLite :memory: gives each connection its own empty database, so we cap the
// pool at 1 connection to ensure both goroutines share the same in-memory store.
func TestDrawNext_Concurrent_OneWinner(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	ownerID := seedUserWithEmail(t, db, "hardening_owner3@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, nil, "sole_winner")

	svc := &RaffleService{db: db}

	type result struct {
		draw *models.RaffleDraw
		err  error
	}
	results := make([]result, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			d, err := svc.DrawNext(raffleID, ownerID)
			results[idx] = result{d, err}
		}(i)
	}
	wg.Wait()

	wins := 0
	for _, r := range results {
		if r.err == nil {
			wins++
		} else if !errors.Is(r.err, ErrRaffleExhausted) {
			t.Errorf("unexpected error (want nil or ErrRaffleExhausted): %v", r.err)
		}
	}
	if wins != 1 {
		t.Errorf("expected exactly 1 winner, got %d", wins)
	}
}

// TestDrawNext_RetriesOnDuplicateKeyConflict verifies that when tx.Create hits
// a (raffle_id, entry_id) unique constraint violation, DrawNext retries the
// SELECT and picks a different, still-available entry instead of returning
// ErrRaffleExhausted.
func TestDrawNext_RetriesOnDuplicateKeyConflict(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	ownerID := seedUserWithEmail(t, db, "retry_owner@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	entryBID := seedEntry(t, db, raffleID, nil, "entry_b")
	seedEntry(t, db, raffleID, nil, "entry_a")
	seedEntry(t, db, raffleID, nil, "entry_c")
	seedDraw(t, db, raffleID, entryBID, "pre-seeded-b-token")

	injected := false
	var injectedEntryID string
	if err := db.Callback().Create().Before("gorm:create").Register("test:race_injector",
		func(scope *gorm.DB) {
			if injected {
				return
			}
			draw, ok := scope.Statement.Dest.(*models.RaffleDraw)
			if !ok || draw.RaffleID != raffleID {
				return
			}
			injected = true
			injectedEntryID = draw.EntryID.String()
			h := sha256.Sum256([]byte("injected-conflict-token"))
			conflictHash := hex.EncodeToString(h[:])
			scope.Statement.ConnPool.ExecContext( //nolint:errcheck
				scope.Statement.Context,
				`INSERT INTO raffle_draws (id, raffle_id, entry_id, claim_token, claim_expires_at, drawn_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				uuid.New().String(),
				draw.RaffleID.String(),
				draw.EntryID.String(),
				conflictHash,
				time.Now().Add(7*24*time.Hour).Format("2006-01-02 15:04:05"),
				time.Now().Format("2006-01-02 15:04:05"),
			)
		},
	); err != nil {
		t.Fatalf("register race injector: %v", err)
	}
	t.Cleanup(func() { db.Callback().Create().Remove("test:race_injector") })

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext should retry after duplicate-key conflict and succeed: %v", err)
	}
	if !injected {
		t.Error("race injector did not fire; retry path was not exercised")
	}
	if draw.EntryID.String() == injectedEntryID {
		t.Errorf("expected a different entry after retry, got same entry as injected conflict: %s", injectedEntryID)
	}
	if draw.Entry.TwitchLogin != "entry_a" && draw.Entry.TwitchLogin != "entry_c" {
		t.Errorf("expected entry_a or entry_c after retry, got %q", draw.Entry.TwitchLogin)
	}
}

// TestDrawNext_ClaimTokenNotStoredRaw verifies the DB stores the SHA-256 hash
// of the claim token, not the raw UUID.
func TestDrawNext_ClaimTokenNotStoredRaw(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "hardening_owner4@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, nil, "hash_player")

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}
	if draw.ClaimTokenRaw == "" {
		t.Fatal("ClaimTokenRaw must be non-empty on a fresh draw")
	}

	var storedToken string
	if err := db.Raw("SELECT claim_token FROM raffle_draws WHERE id = ?", draw.ID.String()).
		Scan(&storedToken).Error; err != nil {
		t.Fatalf("query stored token: %v", err)
	}
	if storedToken == draw.ClaimTokenRaw {
		t.Error("DB must not store the raw token")
	}
	h := sha256.Sum256([]byte(draw.ClaimTokenRaw))
	expected := hex.EncodeToString(h[:])
	if storedToken != expected {
		t.Errorf("DB should store SHA-256 hash\n got  %q\n want %q", storedToken, expected)
	}
}

// TestGetDrawByToken_FindsByRawToken verifies that passing the raw token to
// GetDrawByToken (which hashes it internally) returns the correct draw.
func TestGetDrawByToken_FindsByRawToken(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "hardening_owner5@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, nil, "lookup_player")

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}

	found, err := svc.GetDrawByToken(draw.ClaimTokenRaw)
	if err != nil {
		t.Fatalf("GetDrawByToken with raw token: %v", err)
	}
	if found.ID != draw.ID {
		t.Errorf("expected draw ID %s, got %s", draw.ID, found.ID)
	}
}

// TestGetDrawByToken_RejectsWrongToken verifies ErrClaimNotFound is returned
// for an unrecognised raw token.
func TestGetDrawByToken_RejectsWrongToken(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "hardening_owner6@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, nil, "reject_player")

	svc := &RaffleService{db: db}
	if _, err := svc.DrawNext(raffleID, ownerID); err != nil {
		t.Fatalf("DrawNext: %v", err)
	}

	_, err := svc.GetDrawByToken("definitely-wrong-token")
	if !errors.Is(err, ErrClaimNotFound) {
		t.Errorf("expected ErrClaimNotFound, got %v", err)
	}
}
