package demo

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

func newWalletLinkerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT,
			email TEXT UNIQUE,
			password_hash TEXT,
			avatar_url TEXT,
			role TEXT,
			is_active BOOLEAN,
			email_verified BOOLEAN,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE auth_providers (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			access_token TEXT,
			refresh_token TEXT,
			token_expires_at DATETIME,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create auth_providers: %v", err)
	}
	return db
}

func seedDemoUser(t *testing.T, db *gorm.DB, email string) uuid.UUID {
	t.Helper()

	user := &models.User{
		Email: &email,
		Role:  models.RoleViewer,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user.ID
}

func TestLinkDemoWalletCreatesWeb3ProviderByEmail(t *testing.T) {
	db := newWalletLinkerTestDB(t)
	userID := seedDemoUser(t, db, "viewer@example.com")

	linked, err := LinkDemoWallet(context.Background(), db, LinkDemoWalletInput{
		Email:         "viewer@example.com",
		WalletAddress: "0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
	})
	if err != nil {
		t.Fatalf("LinkDemoWallet: %v", err)
	}
	if linked.UserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, linked.UserID)
	}
	if linked.WalletAddress != "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045" {
		t.Fatalf("expected checksummed address, got %s", linked.WalletAddress)
	}

	var providers []models.AuthProvider
	if err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).Find(&providers).Error; err != nil {
		t.Fatalf("query providers: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 web3 provider, got %d", len(providers))
	}
	if providers[0].ProviderID != linked.WalletAddress {
		t.Fatalf("expected provider_id %s, got %s", linked.WalletAddress, providers[0].ProviderID)
	}
}

func TestLinkDemoWalletUpdatesExistingWeb3Provider(t *testing.T) {
	db := newWalletLinkerTestDB(t)
	userID := seedDemoUser(t, db, "viewer@example.com")
	oldAddress := "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
	if err := db.Create(&models.AuthProvider{
		UserID:     userID,
		Provider:   models.ProviderWeb3,
		ProviderID: oldAddress,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	linked, err := LinkDemoWallet(context.Background(), db, LinkDemoWalletInput{
		UserID:        userID.String(),
		WalletAddress: "0x000000000000000000000000000000000000dEaD",
	})
	if err != nil {
		t.Fatalf("LinkDemoWallet: %v", err)
	}
	if linked.WalletAddress != "0x000000000000000000000000000000000000dEaD" {
		t.Fatalf("expected updated wallet, got %s", linked.WalletAddress)
	}

	var providers []models.AuthProvider
	if err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).Find(&providers).Error; err != nil {
		t.Fatalf("query providers: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected existing provider to be updated, got %d rows", len(providers))
	}
	if providers[0].ProviderID != linked.WalletAddress {
		t.Fatalf("expected provider_id %s, got %s", linked.WalletAddress, providers[0].ProviderID)
	}
}

func TestLinkDemoWalletRejectsInvalidWalletAddress(t *testing.T) {
	db := newWalletLinkerTestDB(t)
	seedDemoUser(t, db, "viewer@example.com")

	_, err := LinkDemoWallet(context.Background(), db, LinkDemoWalletInput{
		Email:         "viewer@example.com",
		WalletAddress: "not-a-wallet",
	})
	if !errors.Is(err, ErrInvalidWalletAddress) {
		t.Fatalf("expected ErrInvalidWalletAddress, got %v", err)
	}
}
