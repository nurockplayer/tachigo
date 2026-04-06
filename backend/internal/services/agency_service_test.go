package services

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

func seedAgencyStreamerRelation(t *testing.T, db *gorm.DB, agencyID uuid.UUID, channelID string) {
	t.Helper()

	streamerUserID := seedStreamerUserRow(t, db, models.RoleStreamer)
	if err := db.Exec(
		`INSERT INTO streamers (id, user_id, channel_id, display_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), streamerUserID, channelID, channelID,
	).Error; err != nil {
		t.Fatalf("seed streamer: %v", err)
	}

	if err := db.Exec(
		`INSERT INTO agency_streamers (id, agency_id, channel_id, created_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		uuid.New(), agencyID, channelID,
	).Error; err != nil {
		t.Fatalf("seed agency streamer: %v", err)
	}
}

func TestAgencyOwnsChannel_OK(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	seedAgencyStreamerRelation(t, db, agencyID, "ch_owned")

	owns, err := svc.OwnsChannel(agencyID, "ch_owned")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !owns {
		t.Fatal("expected agency to own channel")
	}
}

func TestAgencyOwnsChannel_NotOwned(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	seedAgencyStreamerRelation(t, db, agencyID, "ch_owned")

	owns, err := svc.OwnsChannel(agencyID, "ch_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owns {
		t.Fatal("expected agency to not own channel")
	}
}

func TestAgencyOwnsChannel_IsolatedByAgency(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)
	ownerAgencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	otherAgencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	seedAgencyStreamerRelation(t, db, ownerAgencyID, "ch_shared")

	owns, err := svc.OwnsChannel(otherAgencyID, "ch_shared")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owns {
		t.Fatal("expected other agency to not own channel")
	}
}
