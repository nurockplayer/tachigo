package services

import (
	"errors"
	"strings"
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

func TestAgencyStreamerDuplicateRejected(t *testing.T) {
	db := newTestDB(t)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	channelID := "ch_duplicate"

	first := models.AgencyStreamer{
		AgencyID:  agencyID,
		ChannelID: channelID,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	second := models.AgencyStreamer{
		AgencyID:  agencyID,
		ChannelID: channelID,
	}
	err := db.Create(&second).Error
	if err == nil {
		t.Fatal("expected unique violation on duplicate insert")
	}
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected duplicated key error, got %v", err)
	}
}

func TestAgencyStreamerCascadeDelete(t *testing.T) {
	db := newTestDB(t)
	agencyID := seedStreamerUserRow(t, db, models.RoleAgency)
	channelID := "ch_cascade"

	relation := models.AgencyStreamer{
		AgencyID:  agencyID,
		ChannelID: channelID,
	}
	if err := db.Create(&relation).Error; err != nil {
		t.Fatalf("insert agency streamer failed: %v", err)
	}

	if err := db.Unscoped().Delete(&models.User{ID: agencyID}).Error; err != nil {
		t.Fatalf("delete parent agency failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.AgencyStreamer{}).
		Where("agency_id = ? AND channel_id = ?", agencyID, channelID).
		Count(&count).Error; err != nil {
		t.Fatalf("count agency streamer failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected agency streamer row to be deleted, got %d", count)
	}
}

func TestAgencyService_Create_DuplicateName(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	if _, err := svc.Create("agency_x", "agency_x_1@example.com"); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.Create("agency_x", "agency_x_2@example.com")
	if !errors.Is(err, ErrAgencyNameTaken) {
		t.Fatalf("expected ErrAgencyNameTaken, got %v", err)
	}
}

func TestAgencyService_Create_NameTooLong(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	_, err := svc.Create(strings.Repeat("a", 51), "agency_long@example.com")
	if !errors.Is(err, ErrAgencyNameTooLong) {
		t.Fatalf("expected ErrAgencyNameTooLong, got %v", err)
	}
}

func TestAgencyService_Create_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	if _, err := svc.Create("agency_a", "shared@example.com"); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err := svc.Create("agency_b", "shared@example.com")
	if !errors.Is(err, ErrAgencyEmailTaken) {
		t.Fatalf("expected ErrAgencyEmailTaken, got %v", err)
	}
}

func TestAgencyService_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	_, _, err := svc.GetByID(uuid.New())
	if !errors.Is(err, ErrAgencyNotFound) {
		t.Fatalf("expected ErrAgencyNotFound, got %v", err)
	}
}

func TestAgencyService_GetByID_Found_OnboardingIncomplete(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	id := uuid.New()
	name := "test-agency-get"
	email := "ta-get@example.com"
	if err := db.Exec(
		`INSERT INTO users (id, username, email, role, is_active, email_verified, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, 'agency', 1, 1, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id, name, email,
	).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	user, complete, err := svc.GetByID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != id {
		t.Fatalf("expected id %v, got %v", id, user.ID)
	}
	if complete {
		t.Fatal("expected onboarding_complete=false when password_hash IS NULL")
	}
}

func TestAgencyService_GetByID_Found_OnboardingComplete(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	id := uuid.New()
	name := "done-agency-get"
	email := "done-get@example.com"
	if err := db.Exec(
		`INSERT INTO users (id, username, email, role, is_active, email_verified, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, 'agency', 1, 1, 'hashed', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id, name, email,
	).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, complete, err := svc.GetByID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !complete {
		t.Fatal("expected onboarding_complete=true when password_hash IS NOT NULL")
	}
}

func TestAgencyService_GetByID_WrongRole(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	id := uuid.New()
	if err := db.Exec(
		`INSERT INTO users (id, username, email, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, 'viewer', 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id, "not-agency", "not-agency@example.com",
	).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, _, err := svc.GetByID(id)
	if !errors.Is(err, ErrAgencyNotFound) {
		t.Fatalf("expected ErrAgencyNotFound for non-agency role, got %v", err)
	}
}

func TestAgencyService_Create_Success(t *testing.T) {
	db := newTestDB(t)
	svc := NewAgencyService(db)

	user, err := svc.Create("test_agency", "test@example.com")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Fatal("expected non-nil user ID")
	}
	if user.Username == nil || *user.Username != "test_agency" {
		t.Fatalf("expected username 'test_agency', got %v", user.Username)
	}
	if user.Role != models.RoleAgency {
		t.Fatalf("expected role agency, got %v", user.Role)
	}

	// Verify auth_providers row was created atomically with the user.
	var apCount int64
	db.Model(&models.AuthProvider{}).
		Where("user_id = ? AND provider = ?", user.ID, models.ProviderEmail).
		Count(&apCount)
	if apCount != 1 {
		t.Fatalf("expected 1 auth_provider(email) row, got %d", apCount)
	}
}
