package handlers_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/services"
)

const extHandlerSecretRaw = "test-extension-secret-32chars!!!"

func newExtHandlerEnv(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("foreign keys: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	secretB64 := base64.StdEncoding.EncodeToString([]byte(extHandlerSecretRaw))
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
		OAuth: config.OAuthConfig{
			Twitch: config.TwitchConfig{
				ExtensionSecret: secretB64,
			},
		},
	}

	authSvc := services.NewAuthService(db, cfg)
	watchSvc := services.NewWatchService(db)
	pointsSvc := services.NewPointsService(db, watchSvc)
	extSvc := services.NewExtensionService(db, cfg, authSvc, pointsSvc)
	extH := handlers.NewExtensionHandler(extSvc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/extension/t-point/complete", extH.TPointComplete)
	return r, db
}

func seedTwitchUserForHandler(t *testing.T, db *gorm.DB) string {
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
	return twitchID
}

func signExtJWTForHandler(t *testing.T, twitchID, channelID string) string {
	t.Helper()
	type extClaims struct {
		UserID                string `json:"user_id"`
		ExtensionScopedUserID string `json:"opaque_user_id"`
		ChannelID             string `json:"channel_id"`
		Role                  string `json:"role"`
		jwt.RegisteredClaims
	}
	claims := extClaims{
		UserID:                twitchID,
		ExtensionScopedUserID: "U" + twitchID,
		ChannelID:             channelID,
		Role:                  "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(extHandlerSecretRaw))
	if err != nil {
		t.Fatalf("sign ext JWT: %v", err)
	}
	return signed
}

func signReceiptJWTForHandler(t *testing.T, txID, sku string, amount int, txType string) string {
	t.Helper()
	type receiptData struct {
		TransactionID string `json:"transactionId"`
		SKU           string `json:"sku"`
		Amount        int    `json:"amount"`
		Type          string `json:"type"`
	}
	type receiptClaims struct {
		Data receiptData `json:"data"`
		jwt.RegisteredClaims
	}
	claims := receiptClaims{
		Data: receiptData{TransactionID: txID, SKU: sku, Amount: amount, Type: txType},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(extHandlerSecretRaw))
	if err != nil {
		t.Fatalf("sign receipt JWT: %v", err)
	}
	return signed
}

func tpointBody(t *testing.T, extJWT, receipt, sku string) *bytes.Buffer {
	t.Helper()
	b, _ := json.Marshal(map[string]string{
		"extension_jwt":       extJWT,
		"transaction_receipt": receipt,
		"sku":                 sku,
	})
	return bytes.NewBuffer(b)
}

func TestTPointComplete_DuplicateTransactionID_Returns409(t *testing.T) {
	r, db := newExtHandlerEnv(t)
	twitchID := seedTwitchUserForHandler(t, db)
	extJWT := signExtJWTForHandler(t, twitchID, "ch-42")
	receipt := signReceiptJWTForHandler(t, "tx-handler-dup", "TPOINT100", 100, "bits")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/t-point/complete",
		tpointBody(t, extJWT, receipt, "TPOINT100"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first request: want 200, got %d: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/extension/t-point/complete",
		tpointBody(t, extJWT, receipt, "TPOINT100"))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("second request: want 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestTPointComplete_InvalidReceiptAmount_Returns400(t *testing.T) {
	r, db := newExtHandlerEnv(t)
	twitchID := seedTwitchUserForHandler(t, db)
	extJWT := signExtJWTForHandler(t, twitchID, "ch-42")
	receipt := signReceiptJWTForHandler(t, "tx-bad-amount", "TPOINT100", 0, "bits")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/t-point/complete",
		tpointBody(t, extJWT, receipt, "TPOINT100"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}
