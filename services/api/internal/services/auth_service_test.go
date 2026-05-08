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

	nonce, issuedAt, err := svc.Web3Nonce("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nonce == "" {
		t.Error("expected non-empty nonce")
	}
	if issuedAt.IsZero() {
		t.Error("expected non-zero issuedAt")
	}
}

func TestWeb3Nonce_ReplacesExisting(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	address := "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"

	nonce1, _, err := svc.Web3Nonce(address)
	if err != nil {
		t.Fatalf("first Web3Nonce call failed: %v", err)
	}
	nonce2, _, err := svc.Web3Nonce(address)
	if err != nil {
		t.Fatalf("second Web3Nonce call failed: %v", err)
	}

	if nonce1 == nonce2 {
		t.Error("expected different nonces on repeated calls")
	}

	var count int64
	db.Model(&models.Web3Nonce{}).Where("address = ?", strings.ToLower(address)).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 nonce record, got %d", count)
	}
}

func TestWeb3Verify_SuccessConsumesNonceAndIssuesTokens(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	key, addr := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-success"
	nonceRecord := seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, key)

	user, tokens, err := svc.Web3Verify(Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || tokens == nil {
		t.Fatal("expected user and tokens")
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected non-empty token pair")
	}

	var provider models.AuthProvider
	if err := db.Where("user_id = ? AND provider = ?", user.ID, models.ProviderWeb3).First(&provider).Error; err != nil {
		t.Fatalf("web3 provider not found: %v", err)
	}
	if provider.ProviderID != addr {
		t.Fatalf("provider_id: want %s, got %s", addr, provider.ProviderID)
	}

	var nonceCount int64
	db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 0 {
		t.Fatalf("nonce should be consumed, got %d rows", nonceCount)
	}
}

func TestWeb3Verify_NonceDeleteFailureReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	key, addr := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-delete-failure"
	nonceRecord := seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, key)

	if err := db.Exec(`
		CREATE TRIGGER fail_web3_nonce_delete
		BEFORE DELETE ON web3_nonces
		BEGIN
			SELECT RAISE(ABORT, 'forced web3 nonce delete failure');
		END;
	`).Error; err != nil {
		t.Fatalf("create nonce delete trigger: %v", err)
	}

	user, tokens, err := svc.Web3Verify(Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err == nil {
		t.Fatalf("want nonce delete error, got nil (user=%#v tokens=%#v)", user, tokens)
	}
	if !strings.Contains(err.Error(), "forced web3 nonce delete failure") {
		t.Fatalf("want forced delete error, got %v", err)
	}

	var providerCount int64
	db.Model(&models.AuthProvider{}).Where("provider = ?", models.ProviderWeb3).Count(&providerCount)
	if providerCount != 0 {
		t.Fatalf("provider should not be created after nonce delete failure, got %d rows", providerCount)
	}

	var tokenCount int64
	db.Model(&models.RefreshToken{}).Count(&tokenCount)
	if tokenCount != 0 {
		t.Fatalf("refresh token should not be created after nonce delete failure, got %d rows", tokenCount)
	}
}

func TestWeb3Verify_ProviderCreateFailureReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	key, addr := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-provider-failure"
	nonceRecord := seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, key)

	if err := db.Exec(`
		CREATE TRIGGER fail_auth_provider_insert
		BEFORE INSERT ON auth_providers
		BEGIN
			SELECT RAISE(ABORT, 'forced auth provider create failure');
		END;
	`).Error; err != nil {
		t.Fatalf("create auth provider trigger: %v", err)
	}

	user, tokens, err := svc.Web3Verify(Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err == nil {
		t.Fatalf("want provider create error, got nil (user=%#v tokens=%#v)", user, tokens)
	}
	if !strings.Contains(err.Error(), "forced auth provider create failure") {
		t.Fatalf("want forced provider create error, got %v", err)
	}

	var providerCount int64
	db.Model(&models.AuthProvider{}).Where("provider = ?", models.ProviderWeb3).Count(&providerCount)
	if providerCount != 0 {
		t.Fatalf("provider should not be created after provider create failure, got %d rows", providerCount)
	}

	var tokenCount int64
	db.Model(&models.RefreshToken{}).Count(&tokenCount)
	if tokenCount != 0 {
		t.Fatalf("refresh token should not be created after provider create failure, got %d rows", tokenCount)
	}
}

func TestWeb3Verify_RefreshTokenCreateFailureReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	key, addr := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-token-failure"
	nonceRecord := seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, key)

	if err := db.Exec("DROP TABLE refresh_tokens").Error; err != nil {
		t.Fatalf("drop refresh_tokens: %v", err)
	}

	user, tokens, err := svc.Web3Verify(Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err == nil {
		t.Fatalf("want refresh token create error, got nil (user=%#v tokens=%#v)", user, tokens)
	}
	if tokens != nil {
		t.Fatalf("tokens should be nil when refresh token create fails, got %#v", tokens)
	}
}

func TestWeb3Verify_ReplayConsumedNonceReturnsInvalidNonce(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	key, addr := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-replay"
	nonceRecord := seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, key)
	input := Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	}

	if _, _, err := svc.Web3Verify(input); err != nil {
		t.Fatalf("first verify failed: %v", err)
	}
	_, _, err := svc.Web3Verify(input)
	if err != ErrInvalidNonce {
		t.Fatalf("want ErrInvalidNonce on replay, got %v", err)
	}
}

func TestWeb3Verify_InvalidSignatureKeepsNonce(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	_, addr := newTestWallet(t)
	wrongKey, _ := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-bad-signature"
	nonceRecord := seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, wrongKey)

	_, _, err := svc.Web3Verify(Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err != ErrInvalidSignature {
		t.Fatalf("want ErrInvalidSignature, got %v", err)
	}

	var nonceCount int64
	db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 1 {
		t.Fatalf("invalid signature should keep nonce for retry, got %d rows", nonceCount)
	}
}

func TestWeb3Verify_ExpiredNonceReturnsInvalidNonceAndDeletesRecord(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	key, addr := newTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "web3-verify-expired"
	seedExpiredWalletNonce(t, db, addr, nonce)

	var nonceRecord models.Web3Nonce
	if err := db.Where("nonce = ?", nonce).First(&nonceRecord).Error; err != nil {
		t.Fatalf("find seeded nonce: %v", err)
	}
	msg := siweMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := signSIWE(t, msg, key)

	_, _, err := svc.Web3Verify(Web3VerifyInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err != ErrInvalidNonce {
		t.Fatalf("want ErrInvalidNonce, got %v", err)
	}

	var nonceCount int64
	db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 0 {
		t.Fatalf("expired nonce should be deleted, got %d rows", nonceCount)
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

func TestDeleteExpiredRefreshTokens_RemovesExpiredOnly(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())

	// Register a user to get valid tokens.
	_, _, err := svc.Register(RegisterInput{
		Username: "username_del_exp",
		Email:    "del_exp@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, tokens, err := svc.Login(LoginInput{
		Email:    "del_exp@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Manually insert an already-expired refresh token for the same user.
	hash := hashToken(tokens.RefreshToken + "-expired-sentinel")
	var stored models.RefreshToken
	if err := svc.db.Where("token_hash = ?", hashToken(tokens.RefreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("find stored token: %v", err)
	}
	if err := svc.db.Create(&models.RefreshToken{
		UserID:    stored.UserID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(-time.Minute), // already expired
	}).Error; err != nil {
		t.Fatalf("insert expired token: %v", err)
	}

	deleted, err := svc.DeleteExpiredRefreshTokens()
	if err != nil {
		t.Fatalf("DeleteExpiredRefreshTokens: %v", err)
	}
	if deleted != 1 {
		t.Errorf("want 1 deleted, got %d", deleted)
	}

	// Valid token must still exist.
	var count int64
	svc.db.Model(&models.RefreshToken{}).Where("token_hash = ?", hashToken(tokens.RefreshToken)).Count(&count)
	if count != 1 {
		t.Errorf("valid token should still exist, got count=%d", count)
	}
}

func TestLogin_RefreshTokensTableGone_ReturnsError(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	if _, _, err := svc.Register(RegisterInput{
		Username: "username_rt_gone",
		Email:    "rt_gone@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Drop refresh_tokens after Register (which already wrote one) to isolate Login.
	if err := svc.db.Exec("DROP TABLE refresh_tokens").Error; err != nil {
		t.Fatalf("drop refresh_tokens: %v", err)
	}
	// Before fix: Login silently ignores the db.Create error and returns tokens.
	// After fix:  Login propagates the error and returns nil tokens.
	_, _, err := svc.Login(LoginInput{
		Email:    "rt_gone@example.com",
		Password: "Password1!",
	})
	if err == nil {
		t.Fatal("want error when refresh_tokens table is gone, got nil")
	}
}
