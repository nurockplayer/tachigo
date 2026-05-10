package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/testutil"
)

func TestSyncFromTwitchAPI_UsesEncryptedStoredAccessToken(t *testing.T) {
	db := newTestDB(t)
	secret := "oauth-token-encryption-secret"
	encryptedToken, err := newOAuthTokenCipher(secret).encrypt("streamer-access-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}

	ownerID := seedUserWithEmail(t, db, "streamer@example.com")
	viewerID := seedUserWithEmail(t, db, "viewer@example.com")
	raffleID := uuid.New()

	if err := db.Exec(`
		INSERT INTO auth_providers (id, user_id, provider, provider_id, access_token, token_expires_at, created_at, updated_at)
		VALUES (?, ?, 'twitch', 'broadcaster-1', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.New().String(), ownerID.String(), encryptedToken, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatalf("insert streamer provider: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		VALUES (?, ?, 'twitch', 'viewer-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.New().String(), viewerID.String()).Error; err != nil {
		t.Fatalf("insert viewer provider: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO raffles (id, user_id, title, status, source, created_at, updated_at)
		VALUES (?, ?, 'Encrypted Token Raffle', 'draft', 'twitch_api', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, raffleID.String(), ownerID.String()).Error; err != nil {
		t.Fatalf("insert raffle: %v", err)
	}

	svc := NewRaffleService(db, "client-id", "", nil, secret)
	svc.SetTwitchBaseURL("https://twitch.test")
	svc.SetHTTPClient(testutil.NewHTTPClient(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer streamer-access-token" {
			t.Fatalf("expected decrypted bearer token, got %q", got)
		}
		return testutil.NewStringResponse(http.StatusOK, `{"data":[{"user_id":"viewer-1","user_login":"viewer","user_name":"Viewer"}],"pagination":{}}`), nil
	}))

	result, err := svc.SyncFromTwitchAPI(context.Background(), raffleID, ownerID)
	if err != nil {
		t.Fatalf("SyncFromTwitchAPI: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("expected imported=1 skipped=0, got %#v", result)
	}

	var entry models.RaffleEntry
	if err := db.Where("raffle_id = ? AND twitch_login = ?", raffleID, "viewer").First(&entry).Error; err != nil {
		t.Fatalf("expected imported raffle entry: %v", err)
	}
}

func TestSyncFromTwitchAPI_ClearsExpiredStoredAccessToken(t *testing.T) {
	db := newTestDB(t)
	secret := "oauth-token-encryption-secret"
	encryptedToken, err := newOAuthTokenCipher(secret).encrypt("expired-token")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}

	ownerID := seedUserWithEmail(t, db, "expired-streamer@example.com")
	raffleID := uuid.New()
	providerID := uuid.New()
	if err := db.Exec(`
		INSERT INTO auth_providers (id, user_id, provider, provider_id, access_token, token_expires_at, created_at, updated_at)
		VALUES (?, ?, 'twitch', 'broadcaster-expired', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, providerID.String(), ownerID.String(), encryptedToken, time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO raffles (id, user_id, title, status, source, created_at, updated_at)
		VALUES (?, ?, 'Expired Token Raffle', 'draft', 'twitch_api', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, raffleID.String(), ownerID.String()).Error; err != nil {
		t.Fatalf("insert raffle: %v", err)
	}

	svc := NewRaffleService(db, "client-id", "", nil, secret)
	_, err = svc.SyncFromTwitchAPI(context.Background(), raffleID, ownerID)
	if !errors.Is(err, ErrTwitchTokenMissing) {
		t.Fatalf("expected ErrTwitchTokenMissing, got %v", err)
	}

	var ap models.AuthProvider
	if err := db.First(&ap, "id = ?", providerID).Error; err != nil {
		t.Fatalf("load provider: %v", err)
	}
	if ap.AccessToken != nil || ap.RefreshToken != nil || ap.TokenExpiresAt != nil {
		t.Fatalf("expected expired token material to be cleared, got access=%v refresh=%v expiry=%v", ap.AccessToken, ap.RefreshToken, ap.TokenExpiresAt)
	}
}
