package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

// createTestStreamer inserts a minimal user row and returns its ID,
// suitable for use as a raffle owner in prize tier tests.
func createTestStreamer(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("createTestStreamer: uuid: %v", err)
	}
	username := "streamer_" + id.String()[:8]
	user := &models.User{
		ID:       id,
		Username: &username,
		Role:     models.RoleStreamer,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("createTestStreamer: create user: %v", err)
	}
	return id
}

func TestCreatePrizeTier_Success(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)

	raffle, err := svc.Create(streamer, "Test Raffle")
	if err != nil {
		t.Fatalf("create raffle: %v", err)
	}

	tier, err := svc.CreatePrizeTier(raffle.ID, streamer, CreatePrizeTierInput{
		Name:             "一等獎",
		PrizeDescription: "Switch 主機",
		WinnerCount:      1,
	})
	if err != nil {
		t.Fatalf("CreatePrizeTier: %v", err)
	}
	if tier.Name != "一等獎" || tier.WinnerCount != 1 || tier.DrawnCount != 0 || tier.Position != 1 {
		t.Fatalf("unexpected tier: %+v", tier)
	}
}

func TestCreatePrizeTier_InvalidWinnerCount(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")

	_, err := svc.CreatePrizeTier(raffle.ID, streamer, CreatePrizeTierInput{
		Name:        "無效獎",
		WinnerCount: 0,
	})
	if !errors.Is(err, ErrPrizeTierInvalidCount) {
		t.Fatalf("expected ErrPrizeTierInvalidCount, got %v", err)
	}
}

func TestListPrizeTiers_OrderedByPosition(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")

	svc.CreatePrizeTier(raffle.ID, streamer, CreatePrizeTierInput{Name: "二等獎", WinnerCount: 3})
	svc.CreatePrizeTier(raffle.ID, streamer, CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})

	tiers, err := svc.ListPrizeTiers(raffle.ID, streamer)
	if err != nil {
		t.Fatalf("ListPrizeTiers: %v", err)
	}
	if len(tiers) != 2 {
		t.Fatalf("want 2 tiers, got %d", len(tiers))
	}
	if tiers[0].Position != 1 || tiers[1].Position != 2 {
		t.Fatalf("wrong order: positions=%d,%d", tiers[0].Position, tiers[1].Position)
	}
}

func TestDeletePrizeTier_FailsIfDrawn(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")
	tier, _ := svc.CreatePrizeTier(raffle.ID, streamer, CreatePrizeTierInput{Name: "一等獎", WinnerCount: 1})

	// Simulate drawn_count > 0
	db.Model(tier).UpdateColumn("drawn_count", 1)

	err := svc.DeletePrizeTier(raffle.ID, tier.ID, streamer)
	if !errors.Is(err, ErrPrizeTierHasDraws) {
		t.Fatalf("expected ErrPrizeTierHasDraws, got %v", err)
	}
}

func TestUpdatePrizeTier_CannotSetCountBelowDrawn(t *testing.T) {
	db := newTestDB(t)
	svc := NewRaffleService(db, "", "", nil)
	streamer := createTestStreamer(t, db)
	raffle, _ := svc.Create(streamer, "Test Raffle")
	tier, _ := svc.CreatePrizeTier(raffle.ID, streamer, CreatePrizeTierInput{Name: "一等獎", WinnerCount: 3})
	db.Model(tier).UpdateColumn("drawn_count", 2)

	newCount := 1
	_, err := svc.UpdatePrizeTier(raffle.ID, tier.ID, streamer, UpdatePrizeTierInput{WinnerCount: &newCount})
	if !errors.Is(err, ErrPrizeTierInvalidCount) {
		t.Fatalf("expected ErrPrizeTierInvalidCount, got %v", err)
	}
}
