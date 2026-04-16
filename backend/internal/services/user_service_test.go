package services

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

func TestGetByID_Found(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	user, err := svc.GetByID(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != userID {
		t.Errorf("ID: want %s, got %s", userID, user.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := NewUserService(newTestDB(t))

	_, err := svc.GetByID(uuid.New())
	if err != ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

func TestUpdateProfile_Username(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	newName := "newusername"
	user, err := svc.UpdateProfile(userID, UpdateProfileInput{Username: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *user.Username != newName {
		t.Errorf("username: want %s, got %s", newName, *user.Username)
	}
}

func TestUpdateProfile_AvatarURL(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	avatar := "https://example.com/avatar.png"
	user, err := svc.UpdateProfile(userID, UpdateProfileInput{AvatarURL: &avatar})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *user.AvatarURL != avatar {
		t.Errorf("avatar_url: want %s, got %s", avatar, *user.AvatarURL)
	}
}

func TestUpdateProfile_DuplicateUsername(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	taken := "taken"
	id1 := uuid.New()
	db.Create(&models.User{ID: id1, Username: &taken, Role: models.RoleViewer})

	id2 := uuid.New()
	db.Create(&models.User{ID: id2, Role: models.RoleViewer})

	_, err := svc.UpdateProfile(id2, UpdateProfileInput{Username: &taken})
	if err != ErrUsernameExists {
		t.Errorf("want ErrUsernameExists, got %v", err)
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	svc := NewUserService(newTestDB(t))

	name := "ghost"
	_, err := svc.UpdateProfile(uuid.New(), UpdateProfileInput{Username: &name})
	if err != ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

func TestListProviders_ReturnsLinkedProviders(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderEmail, ProviderID: "user@example.com"})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderTwitch, ProviderID: "twitch-123"})

	providers, err := svc.ListProviders(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) != 2 {
		t.Errorf("want 2 providers, got %d", len(providers))
	}
}

func TestListProviders_EmptyForNewUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	providers, err := svc.ListProviders(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("want 0 providers, got %d", len(providers))
	}
}

func newTestWallet(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	addr := common.HexToAddress(crypto.PubkeyToAddress(key.PublicKey).Hex()).Hex()
	return key, addr
}

func signSIWE(t *testing.T, message string, key *ecdsa.PrivateKey) string {
	t.Helper()
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed))
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sig[64] += 27
	return "0x" + hex.EncodeToString(sig)
}

func seedWalletNonce(t *testing.T, db *gorm.DB, address, nonce string) {
	t.Helper()
	if err := db.Create(&models.Web3Nonce{
		Nonce:     nonce,
		Address:   strings.ToLower(address),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed nonce: %v", err)
	}
}

func seedExpiredWalletNonce(t *testing.T, db *gorm.DB, address, nonce string) {
	t.Helper()
	if err := db.Create(&models.Web3Nonce{
		Nonce:     nonce,
		Address:   strings.ToLower(address),
		ExpiresAt: time.Now().Add(-time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed expired nonce: %v", err)
	}
}

func TestLinkWallet_Success(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	nonce := "link-wallet-success"
	seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(strings.ToLower(addr), nonce)
	sig := signSIWE(t, msg, key)

	got, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != addr {
		t.Errorf("address: want %s, got %s", addr, got)
	}

	var ap models.AuthProvider
	if err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).First(&ap).Error; err != nil {
		t.Fatalf("auth provider not found: %v", err)
	}
	if ap.ProviderID != addr {
		t.Errorf("provider_id: want %s, got %s", addr, ap.ProviderID)
	}

	var nonceCount int64
	db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 0 {
		t.Errorf("nonce should be consumed, got %d rows", nonceCount)
	}
}

func TestLinkWallet_AddressAlreadyLinkedToOtherUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userA := uuid.New()
	userB := uuid.New()
	db.Create(&models.User{ID: userA, Role: models.RoleViewer})
	db.Create(&models.User{ID: userB, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	nonceA := "duplicate-address-a"
	seedWalletNonce(t, db, addr, nonceA)
	msgA := siweMessage(strings.ToLower(addr), nonceA)
	if _, err := svc.LinkWallet(userA, LinkWalletInput{
		Address:   addr,
		Nonce:     nonceA,
		Signature: signSIWE(t, msgA, key),
	}); err != nil {
		t.Fatalf("first link: %v", err)
	}

	nonceB := "duplicate-address-b"
	seedWalletNonce(t, db, addr, nonceB)
	msgB := siweMessage(strings.ToLower(addr), nonceB)
	_, err := svc.LinkWallet(userB, LinkWalletInput{
		Address:   addr,
		Nonce:     nonceB,
		Signature: signSIWE(t, msgB, key),
	})
	if err != ErrProviderLinked {
		t.Errorf("want ErrProviderLinked, got %v", err)
	}
}

func TestLinkWallet_ReplacesUsersExistingPrimaryWallet(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key1, addr1 := newTestWallet(t)
	nonce1 := "replace-primary-a"
	seedWalletNonce(t, db, addr1, nonce1)
	msg1 := siweMessage(strings.ToLower(addr1), nonce1)
	if _, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr1,
		Nonce:     nonce1,
		Signature: signSIWE(t, msg1, key1),
	}); err != nil {
		t.Fatalf("first link: %v", err)
	}

	key2, addr2 := newTestWallet(t)
	nonce2 := "replace-primary-b"
	seedWalletNonce(t, db, addr2, nonce2)
	msg2 := siweMessage(strings.ToLower(addr2), nonce2)
	got, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr2,
		Nonce:     nonce2,
		Signature: signSIWE(t, msg2, key2),
	})
	if err != nil {
		t.Fatalf("second link: %v", err)
	}
	if got != addr2 {
		t.Errorf("address: want %s, got %s", addr2, got)
	}

	var activeCount int64
	db.Model(&models.AuthProvider{}).
		Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).
		Count(&activeCount)
	if activeCount != 1 {
		t.Errorf("want 1 active wallet, got %d", activeCount)
	}

	var old models.AuthProvider
	if err := db.Unscoped().
		Where("user_id = ? AND provider = ? AND provider_id = ?", userID, models.ProviderWeb3, addr1).
		First(&old).Error; err != nil {
		t.Fatalf("old wallet row not found: %v", err)
	}
	if !old.DeletedAt.Valid {
		t.Error("old wallet row should be soft-deleted")
	}
}

func TestLinkWallet_InvalidAddress(t *testing.T) {
	svc := NewUserService(newTestDB(t))

	_, err := svc.LinkWallet(uuid.New(), LinkWalletInput{
		Address:   "not-an-address",
		Nonce:     "invalid-address",
		Signature: "0xdeadbeef",
	})
	if err != ErrInvalidWalletAddress {
		t.Errorf("want ErrInvalidWalletAddress, got %v", err)
	}
}

func TestLinkWallet_NonceNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})
	_, addr := newTestWallet(t)

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr,
		Nonce:     "missing-nonce",
		Signature: "0x" + strings.Repeat("ab", 65),
	})
	if err != ErrInvalidNonce {
		t.Errorf("want ErrInvalidNonce, got %v", err)
	}
}

func TestLinkWallet_ExpiredNonce(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})
	_, addr := newTestWallet(t)
	nonce := "expired-nonce"
	seedExpiredWalletNonce(t, db, addr, nonce)

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: "0x" + strings.Repeat("ab", 65),
	})
	if err != ErrInvalidNonce {
		t.Errorf("want ErrInvalidNonce, got %v", err)
	}
}

func TestLinkWallet_InvalidSignature(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	_, addr := newTestWallet(t)
	wrongKey, _ := newTestWallet(t)
	nonce := "invalid-signature"
	seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(strings.ToLower(addr), nonce)

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: signSIWE(t, msg, wrongKey),
	})
	if err != ErrInvalidSignature {
		t.Errorf("want ErrInvalidSignature, got %v", err)
	}

	var nonceCount int64
	db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 1 {
		t.Errorf("nonce should remain after invalid signature, got %d rows", nonceCount)
	}
}

func TestLinkWallet_ReplayConsumedNonceRejected(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	nonce := "replay-consumed-nonce"
	seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(strings.ToLower(addr), nonce)
	input := LinkWalletInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: signSIWE(t, msg, key),
	}

	if _, err := svc.LinkWallet(userID, input); err != nil {
		t.Fatalf("first link: %v", err)
	}
	_, err := svc.LinkWallet(userID, input)
	if err != ErrInvalidNonce {
		t.Errorf("want ErrInvalidNonce, got %v", err)
	}
}

func TestLinkWallet_RestoresSoftDeletedSameWallet(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	deletedAt := time.Now().Add(-time.Hour)
	if err := db.Create(&models.AuthProvider{
		UserID:     userID,
		Provider:   models.ProviderWeb3,
		ProviderID: addr,
		DeletedAt:  gorm.DeletedAt{Time: deletedAt, Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed soft-deleted auth provider: %v", err)
	}

	nonce := "restore-same-wallet"
	seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(strings.ToLower(addr), nonce)
	got, err := svc.LinkWallet(userID, LinkWalletInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: signSIWE(t, msg, key),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != addr {
		t.Errorf("address: want %s, got %s", addr, got)
	}

	var rows []models.AuthProvider
	if err := db.Unscoped().
		Where("user_id = ? AND provider = ? AND provider_id = ?", userID, models.ProviderWeb3, addr).
		Find(&rows).Error; err != nil {
		t.Fatalf("find auth providers: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want restored existing row only, got %d rows", len(rows))
	}
	if rows[0].DeletedAt.Valid {
		t.Error("restored wallet should be active")
	}
}

func TestLinkWallet_IgnoresSoftDeletedWalletLinkedToOtherUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	oldUser := uuid.New()
	newUser := uuid.New()
	db.Create(&models.User{ID: oldUser, Role: models.RoleViewer})
	db.Create(&models.User{ID: newUser, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	if err := db.Create(&models.AuthProvider{
		UserID:     oldUser,
		Provider:   models.ProviderWeb3,
		ProviderID: addr,
		DeletedAt:  gorm.DeletedAt{Time: time.Now().Add(-time.Hour), Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed soft-deleted auth provider: %v", err)
	}

	nonce := "soft-deleted-other-user"
	seedWalletNonce(t, db, addr, nonce)
	msg := siweMessage(strings.ToLower(addr), nonce)
	got, err := svc.LinkWallet(newUser, LinkWalletInput{
		Address:   addr,
		Nonce:     nonce,
		Signature: signSIWE(t, msg, key),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != addr {
		t.Errorf("address: want %s, got %s", addr, got)
	}

	var activeCount int64
	db.Model(&models.AuthProvider{}).
		Where("provider = ? AND provider_id = ?", models.ProviderWeb3, addr).
		Count(&activeCount)
	if activeCount != 1 {
		t.Errorf("want 1 active wallet, got %d", activeCount)
	}
}
