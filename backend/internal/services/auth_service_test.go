package services

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
)

// ─── Register ────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())

	user, tokens, err := svc.Register(RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || tokens == nil {
		t.Fatal("expected user and tokens, got nil")
	}
	if *user.Email != "test@example.com" {
		t.Errorf("email: want test@example.com, got %s", *user.Email)
	}
	if *user.Username != "testuser" {
		t.Errorf("username: want testuser, got %s", *user.Username)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("expected non-empty access and refresh tokens")
	}
	if user.Role != models.RoleViewer {
		t.Errorf("role: want viewer, got %s", user.Role)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	svc.Register(RegisterInput{Username: "user1", Email: "dup@example.com", Password: "password123"})

	_, _, err := svc.Register(RegisterInput{Username: "user2", Email: "dup@example.com", Password: "password123"})
	if err != ErrEmailExists {
		t.Errorf("want ErrEmailExists, got %v", err)
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	svc.Register(RegisterInput{Username: "sameuser", Email: "first@example.com", Password: "password123"})

	_, _, err := svc.Register(RegisterInput{Username: "sameuser", Email: "second@example.com", Password: "password123"})
	if err != ErrUsernameExists {
		t.Errorf("want ErrUsernameExists, got %v", err)
	}
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	svc.Register(RegisterInput{Username: "loginuser", Email: "login@example.com", Password: "mypassword"})

	user, tokens, err := svc.Login(LoginInput{Email: "login@example.com", Password: "mypassword"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || tokens == nil {
		t.Fatal("expected user and tokens")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	svc.Register(RegisterInput{Username: "user", Email: "user@example.com", Password: "correctpass"})

	_, _, err := svc.Login(LoginInput{Email: "user@example.com", Password: "wrongpass"})
	if err != ErrInvalidCredentials {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())

	_, _, err := svc.Login(LoginInput{Email: "nobody@example.com", Password: "pass"})
	if err != ErrInvalidCredentials {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	_, tokens, _ := svc.Register(RegisterInput{Username: "ruser", Email: "r@example.com", Password: "password123"})

	newTokens, err := svc.Refresh(tokens.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestRefresh_RotatesToken(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	_, tokens, _ := svc.Register(RegisterInput{Username: "rotuser", Email: "rot@example.com", Password: "password123"})

	svc.Refresh(tokens.RefreshToken)

	_, err := svc.Refresh(tokens.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("want ErrInvalidToken after rotation, got %v", err)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())

	_, err := svc.Refresh("totally-invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("want ErrInvalidToken, got %v", err)
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	_, tokens, _ := svc.Register(RegisterInput{Username: "expuser", Email: "exp@example.com", Password: "password123"})

	hash := hashToken(tokens.RefreshToken)
	db.Model(&models.RefreshToken{}).Where("token_hash = ?", hash).Update("expires_at", time.Now().Add(-time.Hour))

	_, err := svc.Refresh(tokens.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("want ErrInvalidToken, got %v", err)
	}
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestLogout_InvalidatesRefreshToken(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	_, tokens, _ := svc.Register(RegisterInput{Username: "luser", Email: "l@example.com", Password: "password123"})

	if err := svc.Logout(tokens.RefreshToken); err != nil {
		t.Fatalf("logout error: %v", err)
	}

	_, err := svc.Refresh(tokens.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("want ErrInvalidToken after logout, got %v", err)
	}
}

// ─── ValidateAccessToken ─────────────────────────────────────────────────────

func TestValidateAccessToken_Valid(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	user, tokens, _ := svc.Register(RegisterInput{Username: "tuser", Email: "t@example.com", Password: "password123"})

	claims, err := svc.ValidateAccessToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != user.ID.String() {
		t.Errorf("userID: want %s, got %s", user.ID, claims.UserID)
	}
	if claims.Role != models.RoleViewer {
		t.Errorf("role: want viewer, got %s", claims.Role)
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())

	_, err := svc.ValidateAccessToken("not.a.valid.jwt")
	if err != ErrInvalidToken {
		t.Errorf("want ErrInvalidToken, got %v", err)
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := NewAuthService(newTestDB(t), testConfig())
	_, tokens, _ := svc1.Register(RegisterInput{Username: "s1user", Email: "s1@example.com", Password: "password123"})

	cfg2 := testConfig()
	cfg2.JWT.AccessSecret = "completely-different-secret-value!!"
	svc2 := NewAuthService(newTestDB(t), cfg2)

	_, err := svc2.ValidateAccessToken(tokens.AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("want ErrInvalidToken with wrong secret, got %v", err)
	}
}

// ─── Web3Nonce ───────────────────────────────────────────────────────────────

func TestWeb3Nonce_Success(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())

	nonce, err := svc.Web3Nonce("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nonce == "" {
		t.Error("expected non-empty nonce")
	}
}

func TestWeb3Nonce_ReplacesExisting(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	address := "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"

	nonce1, _ := svc.Web3Nonce(address)
	nonce2, _ := svc.Web3Nonce(address)

	if nonce1 == nonce2 {
		t.Error("expected different nonces on repeated calls")
	}

	var count int64
	db.Model(&models.Web3Nonce{}).Where("address = ?", strings.ToLower(address)).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 nonce record, got %d", count)
	}
}

// ─── UnlinkProvider ──────────────────────────────────────────────────────────

func TestUnlinkProvider_LastProvider_NoPassword(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderWeb3, ProviderID: "0xSomeAddress"})

	err := svc.UnlinkProvider(userID, models.ProviderWeb3)
	if err != ErrLastProvider {
		t.Errorf("want ErrLastProvider, got %v", err)
	}
}

func TestUnlinkProvider_HasPassword_CanUnlinkOnlyProvider(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	user, _, _ := svc.Register(RegisterInput{Username: "unlinkuser", Email: "unlink@example.com", Password: "password123"})

	// Has 1 email provider + a password → can unlink
	err := svc.UnlinkProvider(user.ID, models.ProviderEmail)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnlinkProvider_MultipleProviders(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderTwitch, ProviderID: "twitch-123"})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderGoogle, ProviderID: "google-456"})

	err := svc.UnlinkProvider(userID, models.ProviderTwitch)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var count int64
	db.Model(&models.AuthProvider{}).Where("user_id = ?", userID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 remaining provider, got %d", count)
	}
}

// ─── crypto helpers ───────────────────────────────────────────────────────────

func TestHashToken_Deterministic(t *testing.T) {
	h1 := hashToken("some-token")
	h2 := hashToken("some-token")
	if h1 != h2 {
		t.Error("hashToken should be deterministic")
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	if hashToken("token-a") == hashToken("token-b") {
		t.Error("different inputs should produce different hashes")
	}
}

func TestVerifyEthSignature_InvalidHex(t *testing.T) {
	if verifyEthSignature("msg", "not-hex", "0xaddress") {
		t.Error("expected false for invalid hex signature")
	}
}

func TestVerifyEthSignature_WrongLength(t *testing.T) {
	if verifyEthSignature("msg", "deadbeef", "0xaddress") {
		t.Error("expected false for wrong-length signature")
	}
}
