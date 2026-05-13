package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

// seedDraftRaffle creates a raffle in draft status via the service layer.
func seedDraftRaffle(t *testing.T, db *gorm.DB, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	svc := &RaffleService{db: db}
	raffle, err := svc.Create(ownerID, "Test Raffle")
	if err != nil {
		t.Fatalf("seedDraftRaffle: %v", err)
	}
	return raffle.ID
}

func TestActivate_SuccessFromDraft(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "activate_owner1@example.com")
	raffleID := seedDraftRaffle(t, db, ownerID)

	svc := &RaffleService{db: db}
	raffle, err := svc.Activate(raffleID, ownerID)
	if err != nil {
		t.Fatalf("Activate should succeed from draft: %v", err)
	}
	if raffle.Status != models.RaffleStatusActive {
		t.Errorf("expected status active, got %q", raffle.Status)
	}
}

func TestActivate_ReturnsErrRaffleNotDraft_WhenAlreadyActive(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "activate_owner2@example.com")
	raffleID := seedDraftRaffle(t, db, ownerID)

	svc := &RaffleService{db: db}
	if _, err := svc.Activate(raffleID, ownerID); err != nil {
		t.Fatalf("first activate: %v", err)
	}

	_, err := svc.Activate(raffleID, ownerID)
	if !errors.Is(err, ErrRaffleNotDraft) {
		t.Errorf("expected ErrRaffleNotDraft on second activate, got %v", err)
	}
}

func TestActivate_ReturnsErrRaffleNotDraft_WhenCompleted(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "activate_owner3@example.com")
	raffleID := seedDraftRaffle(t, db, ownerID)

	svc := &RaffleService{db: db}
	// activate first so Complete can proceed (Complete only blocks on completed)
	if _, err := svc.Activate(raffleID, ownerID); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.Complete(raffleID, ownerID); err != nil {
		t.Fatalf("complete: %v", err)
	}

	_, err := svc.Activate(raffleID, ownerID)
	if !errors.Is(err, ErrRaffleNotDraft) {
		t.Errorf("expected ErrRaffleNotDraft when activating completed raffle, got %v", err)
	}
}

func TestImportCSV_ReturnsErrRaffleNotDraft_WhenActive(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "activate_owner4@example.com")
	raffleID := seedDraftRaffle(t, db, ownerID)

	svc := &RaffleService{db: db}
	if _, err := svc.Activate(raffleID, ownerID); err != nil {
		t.Fatalf("activate: %v", err)
	}

	csv := strings.NewReader("viewer1\n")
	_, err := svc.ImportCSV(raffleID, ownerID, csv)
	if !errors.Is(err, ErrRaffleNotDraft) {
		t.Errorf("expected ErrRaffleNotDraft when importing to active raffle, got %v", err)
	}
}
