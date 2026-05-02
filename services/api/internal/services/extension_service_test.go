package services

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/models"
)

const testExtSecretRaw = "test-extension-secret-32chars!!!"

var testExtSecretB64 = base64.StdEncoding.EncodeToString([]byte(testExtSecretRaw))

func extTestConfig() *config.Config {
	cfg := testConfig()
	cfg.OAuth.Twitch.ExtensionSecret = testExtSecretB64
	return cfg
}

func newExtSvc(t *testing.T) (*ExtensionService, *PointsService) {
	t.Helper()
	db := newTestDB(t)
	cfg := extTestConfig()
	authSvc := NewAuthService(db, cfg)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	extSvc := NewExtensionService(db, cfg, authSvc, pointsSvc)
	return extSvc, pointsSvc
}

func seedTwitchUser(t *testing.T, db *gorm.DB) (uuid.UUID, string) {
	t.Helper()
	userID := uuid.New()
	if err := db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'viewer', TRUE, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		userID,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	twitchID := "twitch-" + userID.String()[:8]
	providerID := uuid.New()
	if err := db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, 'twitch', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		providerID, userID, twitchID,
	).Error; err != nil {
		t.Fatalf("seed auth_provider: %v", err)
	}
	return userID, twitchID
}

func makeExtJWT(t *testing.T, twitchUserID, channelID string) string {
	t.Helper()
	claims := ExtensionClaims{
		UserID:                twitchUserID,
		ExtensionScopedUserID: "U" + twitchUserID,
		ChannelID:             channelID,
		Role:                  "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testExtSecretRaw))
	if err != nil {
		t.Fatalf("sign ext JWT: %v", err)
	}
	return signed
}

func makeReceiptJWT(t *testing.T, txID, sku string, amount int, txType string) string {
	t.Helper()
	claims := ReceiptClaims{}
	claims.Data.TransactionID = txID
	claims.Data.SKU = sku
	claims.Data.Amount = amount
	claims.Data.Type = txType
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testExtSecretRaw))
	if err != nil {
		t.Fatalf("sign receipt JWT: %v", err)
	}
	return signed
}

func TestCompleteTPointTransaction_Success(t *testing.T) {
	svc, pointsSvc := newExtSvc(t)
	userID, twitchID := seedTwitchUser(t, svc.db)
	channelID := "channel-42"

	extJWT := makeExtJWT(t, twitchID, channelID)
	receipt := makeReceiptJWT(t, "tx-success-001", "TPOINT100", 100, "bits")

	user, tokens, err := svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != userID {
		t.Errorf("want userID=%s, got %s", userID, user.ID)
	}
	if tokens == nil {
		t.Error("expected tokens, got nil")
	}

	bal, err := pointsSvc.GetBalance(userID, channelID)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.SpendableBalance != 100 {
		t.Errorf("want balance=100, got %d", bal.SpendableBalance)
	}

	var tx models.PointsTransaction
	if err := svc.db.Where("external_transaction_id = ?", "tx-success-001").First(&tx).Error; err != nil {
		t.Errorf("points_transaction not found by external_transaction_id: %v", err)
	}
}

func TestCompleteTPointTransaction_DuplicateTransactionID_ReturnsErrDuplicate(t *testing.T) {
	svc, pointsSvc := newExtSvc(t)
	userID, twitchID := seedTwitchUser(t, svc.db)
	channelID := "channel-42"

	extJWT := makeExtJWT(t, twitchID, channelID)
	receipt := makeReceiptJWT(t, "tx-dup-001", "TPOINT100", 100, "bits")

	if _, _, err := svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100"); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	_, _, err := svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100")
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Errorf("want ErrDuplicateTransaction, got %v", err)
	}

	bal, err := pointsSvc.GetBalance(userID, channelID)
	if err != nil {
		t.Fatalf("GetBalance after duplicate: %v", err)
	}
	if bal.SpendableBalance != 100 {
		t.Errorf("want balance=100 after duplicate, got %d", bal.SpendableBalance)
	}
}

func TestCompleteTPointTransaction_PointsWriteFailure_ReturnsOriginalError(t *testing.T) {
	svc, _ := newExtSvc(t)
	_, twitchID := seedTwitchUser(t, svc.db)
	channelID := "channel-42"
	extJWT := makeExtJWT(t, twitchID, channelID)
	receipt := makeReceiptJWT(t, "tx-db-fail-001", "TPOINT100", 100, "bits")

	if err := svc.db.Exec("DROP TABLE points_transactions").Error; err != nil {
		t.Fatalf("drop points_transactions: %v", err)
	}

	_, _, err := svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if errors.Is(err, ErrDuplicateTransaction) {
		t.Fatalf("non-duplicate DB error must not map to ErrDuplicateTransaction: %v", err)
	}
}

func TestCompleteTPointTransaction_InvalidReceipt_Errors(t *testing.T) {
	svc, _ := newExtSvc(t)
	_, twitchID := seedTwitchUser(t, svc.db)
	channelID := "channel-42"

	cases := []struct {
		name    string
		receipt func() string
		sku     string
		wantErr error
	}{
		{
			name:    "amount zero",
			receipt: func() string { return makeReceiptJWT(t, "tx-v1", "TPOINT100", 0, "bits") },
			sku:     "TPOINT100",
			wantErr: ErrInvalidReceiptAmount,
		},
		{
			name:    "amount negative",
			receipt: func() string { return makeReceiptJWT(t, "tx-v2", "TPOINT100", -50, "bits") },
			sku:     "TPOINT100",
			wantErr: ErrInvalidReceiptAmount,
		},
		{
			name:    "wrong type",
			receipt: func() string { return makeReceiptJWT(t, "tx-v3", "TPOINT100", 100, "subscription") },
			sku:     "TPOINT100",
			wantErr: ErrInvalidReceiptType,
		},
		{
			name:    "sku mismatch",
			receipt: func() string { return makeReceiptJWT(t, "tx-v4", "OTHER_SKU", 100, "bits") },
			sku:     "TPOINT100",
			wantErr: ErrInvalidReceipt,
		},
		{
			name:    "empty transactionId",
			receipt: func() string { return makeReceiptJWT(t, "", "TPOINT100", 100, "bits") },
			sku:     "TPOINT100",
			wantErr: ErrInvalidReceipt,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			extJWT := makeExtJWT(t, twitchID, channelID)
			_, _, err := svc.CompleteTPointTransaction(extJWT, tc.receipt(), tc.sku)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCompleteTPointTransaction_TokenIssueFailure_PointsCreditedOnce_RetryErrDuplicate(t *testing.T) {
	svc, pointsSvc := newExtSvc(t)
	userID, twitchID := seedTwitchUser(t, svc.db)
	channelID := "channel-retry"
	extJWT := makeExtJWT(t, twitchID, channelID)
	receipt := makeReceiptJWT(t, "tx-retry-001", "TPOINT100", 100, "bits")

	// Drop refresh_tokens so issueTokenPair fails after points are written.
	if err := svc.db.Exec("DROP TABLE refresh_tokens").Error; err != nil {
		t.Fatalf("drop refresh_tokens: %v", err)
	}

	_, _, err := svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100")
	if err == nil {
		t.Fatal("want error when refresh_tokens is gone, got nil")
	}

	// Points must be credited despite token issuance failure.
	bal, err := pointsSvc.GetBalance(userID, channelID)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.SpendableBalance != 100 {
		t.Errorf("want balance=100 after token failure, got %d", bal.SpendableBalance)
	}

	// Retry with same receipt: AddPointsWithMeta fails before reaching issueTokenPair,
	// so ErrDuplicateTransaction is returned — the documented retry contract.
	_, _, err = svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100")
	if !errors.Is(err, ErrDuplicateTransaction) {
		t.Errorf("want ErrDuplicateTransaction on retry, got %v", err)
	}
}

func TestCompleteTPointTransaction_PointsWriteFailure_NoOrphanRefreshToken(t *testing.T) {
	svc, _ := newExtSvc(t)
	_, twitchID := seedTwitchUser(t, svc.db)
	extJWT := makeExtJWT(t, twitchID, "channel-42")
	receipt := makeReceiptJWT(t, "tx-orphan-001", "TPOINT100", 100, "bits")

	// newTestDB provides per-test DB isolation: DROP TABLE here only affects this test.
	// RefreshToken has no DeletedAt, so Count() is a direct row count.
	var countBefore int64
	svc.db.Model(&models.RefreshToken{}).Count(&countBefore)

	if err := svc.db.Exec("DROP TABLE points_transactions").Error; err != nil {
		t.Fatalf("drop points_transactions: %v", err)
	}

	_, _, err := svc.CompleteTPointTransaction(extJWT, receipt, "TPOINT100")
	if err == nil {
		t.Fatal("want error, got nil")
	}

	var countAfter int64
	svc.db.Model(&models.RefreshToken{}).Count(&countAfter)
	if countAfter != countBefore {
		t.Errorf("points write failure must not create refresh tokens: got %d new record(s)", countAfter-countBefore)
	}
}
