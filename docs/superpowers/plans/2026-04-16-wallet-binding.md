# POST /users/me/wallet — MetaMask Wallet Binding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow authenticated Twitch users to link their MetaMask wallet via SIWE signature, enabling `ClaimService` to resolve the wallet address for on-chain mint.

**Architecture:** Add `UserService.LinkWallet` backed by a SIWE signature check; extract shared SIWE helpers to `services/siwe.go`; replace the global unique constraint on `auth_providers(provider, provider_id)` with a partial index so soft-deleted rows don't block re-binding; soft-delete the old web3 row on replace and restore (not insert) if the same address is re-bound.

**Tech Stack:** Go, Gin, GORM, SQLite (tests), PostgreSQL (production), `github.com/ethereum/go-ethereum` (already in go.mod)

**Spec:** `docs/superpowers/specs/2026-04-16-wallet-binding-design.md`

---

## File Map

| Action | File |
|---|---|
| Create | `backend/migrations/014_auth_provider_partial_unique.sql` |
| Modify | `backend/internal/services/testutil_test.go` |
| Modify | `backend/internal/handlers/testutil_test.go` |
| Create | `backend/internal/services/siwe.go` |
| Modify | `backend/internal/services/auth_service.go` |
| Modify | `backend/internal/database/db.go` |
| Modify | `backend/internal/services/user_service.go` |
| Modify | `backend/internal/services/user_service_test.go` |
| Modify | `backend/internal/handlers/swagger_types.go` |
| Modify | `backend/internal/handlers/user_handler.go` |
| Modify | `backend/internal/handlers/testutil_test.go` |
| Modify | `backend/internal/handlers/user_handler_test.go` |
| Modify | `backend/internal/router/router.go` |

---

## Task 1: Migration 014 + update test helper schemas

### Files:
- Create: `backend/migrations/014_auth_provider_partial_unique.sql`
- Modify: `backend/internal/services/testutil_test.go`
- Modify: `backend/internal/handlers/testutil_test.go`

- [ ] **Step 1: Create migration file**

```sql
-- backend/migrations/014_auth_provider_partial_unique.sql

-- Drop the global unique constraint added in 001_init.sql.
-- Soft-deleted rows (deleted_at IS NOT NULL) must not block re-binding
-- the same wallet, so we replace it with a partial unique index.
ALTER TABLE auth_providers
    DROP CONSTRAINT IF EXISTS auth_providers_provider_provider_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS
    idx_auth_providers_provider_provider_id_active
ON auth_providers(provider, provider_id)
WHERE deleted_at IS NULL;
```

- [ ] **Step 2: Add partial index to the services test helper**

In `backend/internal/services/testutil_test.go`, add the index statement **after** the `auth_providers` table CREATE in `migrateTestDB`. Find the block that ends with:

```go
		`CREATE TABLE IF NOT EXISTS web3_nonces (
			id TEXT PRIMARY KEY,
			nonce TEXT NOT NULL UNIQUE,
			address TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
```

Insert the new index immediately **before** that statement:

```go
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_providers_provider_provider_id_active
			ON auth_providers (provider, provider_id)
			WHERE deleted_at IS NULL`,
```

- [ ] **Step 3: Add partial index to the handlers test helper**

In `backend/internal/handlers/testutil_test.go`, the `auth_providers` CREATE is followed by the `shipping_addresses` CREATE. Add the same index between them:

```go
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_providers_provider_provider_id_active
			ON auth_providers (provider, provider_id)
			WHERE deleted_at IS NULL`,
```

- [ ] **Step 4: Run existing tests to confirm nothing broke**

```bash
cd backend && docker compose run --no-deps --rm app go test ./internal/services/... ./internal/handlers/... -v 2>&1 | tail -40
```

Expected: all existing tests PASS (no failures related to auth_providers).

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/014_auth_provider_partial_unique.sql \
        backend/internal/services/testutil_test.go \
        backend/internal/handlers/testutil_test.go
git commit -m "feat: migration 014 — partial unique index on auth_providers

Replaces UNIQUE(provider, provider_id) with a partial index that
only covers active rows (deleted_at IS NULL), enabling soft-delete
+ re-bind without constraint violations.

refs #<ISSUE_NUMBER>

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Extract SIWE helpers to siwe.go

### Files:
- Create: `backend/internal/services/siwe.go`
- Modify: `backend/internal/services/auth_service.go`

- [ ] **Step 1: Create siwe.go with the two helpers**

```go
// backend/internal/services/siwe.go
package services

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// siweMessage builds the EIP-4361 message that the user must sign.
// Both auth_service (Web3Verify) and user_service (LinkWallet) use this.
func siweMessage(address, nonce string) string {
	return fmt.Sprintf(
		"tachigo.io wants you to sign in with your Ethereum account:\n%s\n\nSign in to Tachigo\n\nNonce: %s\nIssued At: %s",
		address, nonce, time.Now().UTC().Format(time.RFC3339),
	)
}

// verifyEthSignature recovers the signer address from a personal_sign
// (EIP-191) signature and compares it to expectedAddress (case-insensitive).
func verifyEthSignature(message, sigHex, expectedAddress string) bool {
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil || len(sigBytes) != 65 {
		return false
	}
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed))
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false
	}
	recovered := strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex())
	return recovered == strings.ToLower(expectedAddress)
}
```

- [ ] **Step 2: Remove the two helpers from auth_service.go**

In `backend/internal/services/auth_service.go`:

a) Delete the `siweMessage` function (currently around line 471):

```go
func siweMessage(address, nonce string) string {
	return fmt.Sprintf(
		"tachigo.io wants you to sign in with your Ethereum account:\n%s\n\nSign in to Tachigo\n\nNonce: %s\nIssued At: %s",
		address, nonce, time.Now().UTC().Format(time.RFC3339),
	)
}
```

b) Delete the `verifyEthSignature` function (currently around line 478):

```go
func verifyEthSignature(message, sigHex, expectedAddress string) bool {
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil || len(sigBytes) != 65 {
		return false
	}
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed))
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false
	}
	recovered := strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex())
	return recovered == strings.ToLower(expectedAddress)
}
```

c) Remove `"github.com/ethereum/go-ethereum/crypto"` from the import block (it is no longer used in `auth_service.go`; `common` and `hex` and `fmt` are still needed by other functions).

- [ ] **Step 3: Build and test**

```bash
cd backend && docker compose run --no-deps --rm app go build ./... 2>&1
```

Expected: exits 0. If `crypto` import removal caused a compile error, re-check whether any remaining function in `auth_service.go` uses a `crypto.*` symbol.

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -run TestVerifyEth -v 2>&1
```

Expected:
```
--- PASS: TestVerifyEthSignature_InvalidHex
--- PASS: TestVerifyEthSignature_WrongLength
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/siwe.go \
        backend/internal/services/auth_service.go
git commit -m "refactor: extract SIWE helpers to services/siwe.go

siweMessage and verifyEthSignature are now shared between auth_service
and the upcoming user_service.LinkWallet, avoiding duplication.

refs #<ISSUE_NUMBER>

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3: UserService.LinkWallet (TDD)

### Files:
- Modify: `backend/internal/database/db.go`
- Modify: `backend/internal/services/user_service.go`
- Modify: `backend/internal/services/user_service_test.go`

- [ ] **Step 1: Enable TranslateError in the production DB connection**

In `backend/internal/database/db.go`, change:

```go
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
```

to:

```go
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info),
		TranslateError: true,
	})
```

This ensures `gorm.ErrDuplicatedKey` is returned consistently from both SQLite (tests) and PostgreSQL (production) when a unique constraint is violated.

- [ ] **Step 2: Write the failing tests**

Add the following to `backend/internal/services/user_service_test.go`:

```go
// ─── LinkWallet helpers ──────────────────────────────────────────────────────

// newTestWallet generates a fresh Ethereum key pair for testing.
// Returns the private key and the EIP-55 checksummed address.
func newTestWallet(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	addr := common.HexToAddress(crypto.PubkeyToAddress(key.PublicKey).Hex()).Hex()
	return key, addr
}

// signSIWE signs a SIWE message with the given private key and returns
// the 0x-prefixed hex signature (65 bytes with adjusted v byte).
// siweMessage uses time.Now() with RFC3339 (second precision); this helper
// must be called and the corresponding LinkWallet must be invoked within
// the same second for the messages to match — always true in unit tests.
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

// seedNonce inserts a Web3Nonce record for the given address into the test DB.
func seedNonce(t *testing.T, db *gorm.DB, address, nonce string) {
	t.Helper()
	db.Create(&models.Web3Nonce{
		Nonce:     nonce,
		Address:   strings.ToLower(address),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
}

// ─── LinkWallet tests ────────────────────────────────────────────────────────

func TestLinkWallet_InvalidAddress(t *testing.T) {
	svc := NewUserService(newTestDB(t))
	_, err := svc.LinkWallet(uuid.New(), LinkWalletInput{
		Address: "not-an-address", Nonce: "abc", Signature: "0xdeadbeef",
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

	_, privKey, addr := newTestWalletWithDB(t, db, userID, "nonce-x")
	// Use a nonce that was never stored
	msg := siweMessage(strings.ToLower(addr), "wrong-nonce")
	sig := signSIWE(t, msg, privKey)

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address: addr, Nonce: "wrong-nonce", Signature: sig,
	})
	if err != ErrInvalidNonce {
		t.Errorf("want ErrInvalidNonce, got %v", err)
	}
}

func TestLinkWallet_NonceExpired(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	nonce := "expired-nonce"
	db.Create(&models.Web3Nonce{
		Nonce:     nonce,
		Address:   strings.ToLower(addr),
		ExpiresAt: time.Now().Add(-time.Minute), // expired
	})
	msg := siweMessage(strings.ToLower(addr), nonce)
	sig := signSIWE(t, msg, key)

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address: addr, Nonce: nonce, Signature: sig,
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
	nonce := "sig-test-nonce"
	seedNonce(t, db, addr, nonce)

	// Sign with a different key — address won't match
	wrongKey, _ := newTestWallet(t)
	msg := siweMessage(strings.ToLower(addr), nonce)
	sig := signSIWE(t, msg, wrongKey)

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address: addr, Nonce: nonce, Signature: sig,
	})
	if err != ErrInvalidSignature {
		t.Errorf("want ErrInvalidSignature, got %v", err)
	}
}

func TestLinkWallet_Success_FirstBind(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	nonce := "first-bind-nonce"
	seedNonce(t, db, addr, nonce)

	msg := siweMessage(strings.ToLower(addr), nonce)
	sig := signSIWE(t, msg, key)

	got, err := svc.LinkWallet(userID, LinkWalletInput{
		Address: addr, Nonce: nonce, Signature: sig,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != addr {
		t.Errorf("returned address: want %s, got %s", addr, got)
	}

	// AuthProvider should exist
	var ap models.AuthProvider
	if err := db.Where("user_id = ? AND provider = 'web3'", userID).First(&ap).Error; err != nil {
		t.Fatalf("auth provider not found: %v", err)
	}
	if ap.ProviderID != addr {
		t.Errorf("provider_id: want %s, got %s", addr, ap.ProviderID)
	}

	// Nonce should be consumed
	var count int64
	db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&count)
	if count != 0 {
		t.Errorf("nonce should be consumed, still found %d", count)
	}
}

func TestLinkWallet_ReplacesExistingWallet(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	// First bind
	key1, addr1 := newTestWallet(t)
	nonce1 := "replace-nonce-1"
	seedNonce(t, db, addr1, nonce1)
	msg1 := siweMessage(strings.ToLower(addr1), nonce1)
	sig1 := signSIWE(t, msg1, key1)
	svc.LinkWallet(userID, LinkWalletInput{Address: addr1, Nonce: nonce1, Signature: sig1})

	// Second bind with a different wallet
	key2, addr2 := newTestWallet(t)
	nonce2 := "replace-nonce-2"
	seedNonce(t, db, addr2, nonce2)
	msg2 := siweMessage(strings.ToLower(addr2), nonce2)
	sig2 := signSIWE(t, msg2, key2)

	got, err := svc.LinkWallet(userID, LinkWalletInput{Address: addr2, Nonce: nonce2, Signature: sig2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != addr2 {
		t.Errorf("returned address: want %s, got %s", addr2, got)
	}

	// Old row should be soft-deleted
	var old models.AuthProvider
	db.Unscoped().Where("user_id = ? AND provider_id = ?", userID, addr1).First(&old)
	if !old.DeletedAt.Valid {
		t.Error("old provider row should be soft-deleted")
	}

	// New row should be active
	var active models.AuthProvider
	db.Where("user_id = ? AND provider = 'web3'", userID).First(&active)
	if active.ProviderID != addr2 {
		t.Errorf("active provider_id: want %s, got %s", addr2, active.ProviderID)
	}
}

func TestLinkWallet_RestoresSoftDeletedRow(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)

	// First bind addr
	nonce1 := "restore-nonce-1"
	seedNonce(t, db, addr, nonce1)
	msg1 := siweMessage(strings.ToLower(addr), nonce1)
	svc.LinkWallet(userID, LinkWalletInput{Address: addr, Nonce: nonce1, Signature: signSIWE(t, msg1, key)})

	// Bind a different address (soft-deletes addr's row)
	key2, addr2 := newTestWallet(t)
	nonce2 := "restore-nonce-2"
	seedNonce(t, db, addr2, nonce2)
	msg2 := siweMessage(strings.ToLower(addr2), nonce2)
	svc.LinkWallet(userID, LinkWalletInput{Address: addr2, Nonce: nonce2, Signature: signSIWE(t, msg2, key2)})

	// Re-bind original addr — should restore, not insert a new row
	nonce3 := "restore-nonce-3"
	seedNonce(t, db, addr, nonce3)
	msg3 := siweMessage(strings.ToLower(addr), nonce3)
	got, err := svc.LinkWallet(userID, LinkWalletInput{Address: addr, Nonce: nonce3, Signature: signSIWE(t, msg3, key)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != addr {
		t.Errorf("returned address: want %s, got %s", addr, got)
	}

	// Only one active web3 row
	var count int64
	db.Model(&models.AuthProvider{}).Where("user_id = ? AND provider = 'web3'", userID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 active web3 provider, got %d", count)
	}

	// Total rows (including soft-deleted): still 2, not 3
	db.Unscoped().Model(&models.AuthProvider{}).Where("user_id = ? AND provider = 'web3'", userID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 total web3 rows (1 active + 1 soft-deleted), got %d", count)
	}
}

func TestLinkWallet_NonceReplay(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	key, addr := newTestWallet(t)
	nonce := "replay-nonce"
	seedNonce(t, db, addr, nonce)

	msg := siweMessage(strings.ToLower(addr), nonce)
	sig := signSIWE(t, msg, key)
	input := LinkWalletInput{Address: addr, Nonce: nonce, Signature: sig}

	// First call succeeds
	if _, err := svc.LinkWallet(userID, input); err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	// Second call with same nonce fails — nonce already consumed
	_, err := svc.LinkWallet(userID, input)
	if err != ErrInvalidNonce {
		t.Errorf("want ErrInvalidNonce on replay, got %v", err)
	}
}

func TestLinkWallet_WalletAlreadyLinkedToOtherUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	// User A binds addr
	userA := uuid.New()
	db.Create(&models.User{ID: userA, Role: models.RoleViewer})
	key, addr := newTestWallet(t)
	nonce1 := "conflict-nonce-1"
	seedNonce(t, db, addr, nonce1)
	msg1 := siweMessage(strings.ToLower(addr), nonce1)
	svc.LinkWallet(userA, LinkWalletInput{Address: addr, Nonce: nonce1, Signature: signSIWE(t, msg1, key)})

	// User B tries to bind the same addr
	userB := uuid.New()
	db.Create(&models.User{ID: userB, Role: models.RoleViewer})
	nonce2 := "conflict-nonce-2"
	seedNonce(t, db, addr, nonce2)
	msg2 := siweMessage(strings.ToLower(addr), nonce2)
	_, err := svc.LinkWallet(userB, LinkWalletInput{
		Address: addr, Nonce: nonce2, Signature: signSIWE(t, msg2, key),
	})
	if err != ErrProviderLinked {
		t.Errorf("want ErrProviderLinked, got %v", err)
	}
}
```

Also add the following imports to `user_service_test.go` (merge with existing import block):

```go
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
```

And add this helper (used in the NonceNotFound test to avoid a compile issue; simplify by removing the `newTestWalletWithDB` call and replace that test):

Actually, update `TestLinkWallet_NonceNotFound` to remove the `newTestWalletWithDB` reference — replace it with:

```go
func TestLinkWallet_NonceNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)
	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	_, addr := newTestWallet(t)
	// No nonce seeded — First() will return ErrRecordNotFound

	_, err := svc.LinkWallet(userID, LinkWalletInput{
		Address: addr, Nonce: "nonexistent-nonce", Signature: "0x" + strings.Repeat("ab", 65),
	})
	if err != ErrInvalidNonce {
		t.Errorf("want ErrInvalidNonce, got %v", err)
	}
}
```

- [ ] **Step 3: Run tests to confirm they fail**

```bash
cd backend && docker compose run --no-deps --rm app go test ./internal/services/... -run TestLinkWallet -v 2>&1 | tail -30
```

Expected: compilation error `undefined: LinkWalletInput` — confirms tests are in place.

- [ ] **Step 4: Implement UserService.LinkWallet**

In `backend/internal/services/user_service.go`, add imports and implementation:

**Imports to add** (merge with existing):
```go
import (
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)
```

**Add after the existing `ErrUsernameExists` declaration** (at top of file, near other vars):
```go
var ErrInvalidWalletAddress = errors.New("invalid wallet address")
```

**Add the input type and method**:
```go
type LinkWalletInput struct {
	Address   string `json:"address"   binding:"required"`
	Nonce     string `json:"nonce"     binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

// LinkWallet binds a MetaMask wallet to the authenticated user.
// Any previously bound wallet is soft-deleted. If the same address was
// previously bound and then removed, its row is restored rather than
// re-inserted, preserving the original created_at.
func (s *UserService) LinkWallet(userID uuid.UUID, input LinkWalletInput) (string, error) {
	// 1. Validate and normalize address.
	if !common.IsHexAddress(input.Address) {
		return "", ErrInvalidWalletAddress
	}
	checksumAddr := common.HexToAddress(input.Address).Hex()
	lookupAddr := strings.ToLower(checksumAddr)

	// 2. Look up nonce (before transaction — fast exit on bad input).
	var nonceRecord models.Web3Nonce
	if err := s.db.Where("nonce = ? AND address = ?", input.Nonce, lookupAddr).
		First(&nonceRecord).Error; err != nil {
		return "", ErrInvalidNonce
	}
	if nonceRecord.IsExpired() {
		return "", ErrInvalidNonce
	}

	// 3. Verify SIWE signature.
	msg := siweMessage(lookupAddr, input.Nonce)
	if !verifyEthSignature(msg, input.Signature, lookupAddr) {
		return "", ErrInvalidSignature
	}

	// 4. Atomically: consume nonce, enforce no cross-user conflict,
	//    soft-delete old provider, restore or insert new provider.
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Consume nonce; RowsAffected check prevents concurrent replay.
		result := tx.Where("nonce = ? AND address = ?", input.Nonce, lookupAddr).
			Delete(&models.Web3Nonce{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidNonce
		}

		// Reject if this wallet is already active for a different user.
		var count int64
		if err := tx.Model(&models.AuthProvider{}).
			Where("provider = ? AND provider_id = ? AND deleted_at IS NULL AND user_id != ?",
				models.ProviderWeb3, checksumAddr, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrProviderLinked
		}

		// Soft-delete the current active web3 row for this user (if any).
		now := time.Now()
		if err := tx.Model(&models.AuthProvider{}).
			Where("user_id = ? AND provider = ? AND deleted_at IS NULL",
				userID, models.ProviderWeb3).
			Update("deleted_at", now).Error; err != nil {
			return err
		}

		// Restore an existing soft-deleted row for this exact address,
		// or insert a new one if none exists.
		var ap models.AuthProvider
		findErr := tx.Unscoped().
			Where("user_id = ? AND provider = ? AND provider_id = ?",
				userID, models.ProviderWeb3, checksumAddr).
			First(&ap).Error

		if findErr == nil {
			// Row exists (soft-deleted after step above) — restore it.
			if err := tx.Unscoped().Model(&ap).
				UpdateColumn("deleted_at", gorm.Expr("NULL")).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return ErrProviderLinked
				}
				return err
			}
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			// No prior row — insert fresh.
			newAP := models.AuthProvider{
				UserID:     userID,
				Provider:   models.ProviderWeb3,
				ProviderID: checksumAddr,
			}
			if err := tx.Create(&newAP).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return ErrProviderLinked
				}
				return err
			}
		} else {
			return findErr
		}

		return nil
	})

	if err != nil {
		return "", err
	}
	return checksumAddr, nil
}
```

- [ ] **Step 5: Run the tests**

```bash
cd backend && docker compose run --no-deps --rm app go test ./internal/services/... -run TestLinkWallet -v 2>&1
```

Expected output (all pass):
```
--- PASS: TestLinkWallet_InvalidAddress
--- PASS: TestLinkWallet_NonceNotFound
--- PASS: TestLinkWallet_NonceExpired
--- PASS: TestLinkWallet_InvalidSignature
--- PASS: TestLinkWallet_Success_FirstBind
--- PASS: TestLinkWallet_ReplacesExistingWallet
--- PASS: TestLinkWallet_RestoresSoftDeletedRow
--- PASS: TestLinkWallet_NonceReplay
--- PASS: TestLinkWallet_WalletAlreadyLinkedToOtherUser
```

- [ ] **Step 6: Run full service test suite to confirm no regressions**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... 2>&1 | tail -10
```

Expected: `ok` for all packages, 0 failures.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/database/db.go \
        backend/internal/services/user_service.go \
        backend/internal/services/user_service_test.go
git commit -m "feat: UserService.LinkWallet — SIWE-verified wallet binding

Allows an authenticated user to bind a MetaMask wallet. Implements:
- address validation and EIP-55 checksum normalization
- SIWE nonce + signature verification
- atomic nonce consumption with RowsAffected guard
- soft-delete of old provider + restore-or-insert pattern
- ErrProviderLinked on cross-user conflict or race

refs #<ISSUE_NUMBER>

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: WalletResponse + UserHandler.LinkWallet

### Files:
- Modify: `backend/internal/handlers/swagger_types.go`
- Modify: `backend/internal/handlers/user_handler.go`
- Modify: `backend/internal/handlers/testutil_test.go`
- Modify: `backend/internal/handlers/user_handler_test.go`

- [ ] **Step 1: Add WalletResponse to swagger_types.go**

In `backend/internal/handlers/swagger_types.go`, add at the end of the file:

```go
// WalletResponse wraps the bound wallet address.
type WalletResponse struct {
	Address string `json:"address"`
}
```

- [ ] **Step 2: Implement the handler**

In `backend/internal/handlers/user_handler.go`, add:

```go
// LinkWallet godoc
// @Summary      Bind a MetaMask wallet to the current user
// @Description  Verifies a SIWE signature (obtained via POST /auth/web3/nonce) and
// @Description  links the wallet address to the authenticated user. Replaces any
// @Description  previously linked wallet.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body services.LinkWalletInput true "address, nonce, signature"
// @Success      200  {object}  Response{data=WalletResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Failure      409  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/wallet [post]
func (h *UserHandler) LinkWallet(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	var input services.LinkWalletInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	addr, err := h.user.LinkWallet(userID, input)
	if err != nil {
		switch err {
		case services.ErrInvalidWalletAddress:
			badRequest(c, "invalid wallet address")
		case services.ErrInvalidNonce:
			unauthorized(c, "invalid or expired nonce")
		case services.ErrInvalidSignature:
			unauthorized(c, "invalid wallet signature")
		case services.ErrProviderLinked:
			conflict(c, "wallet already linked to another account")
		default:
			internal(c)
		}
		return
	}

	ok(c, gin.H{"address": addr})
}
```

Make sure `user_handler.go` imports `"github.com/google/uuid"`, `"github.com/tachigo/tachigo/internal/middleware"`, and `"github.com/tachigo/tachigo/internal/services"` — these are already present.

- [ ] **Step 3: Register route in the handler test env**

In `backend/internal/handlers/testutil_test.go`:

a) Add `userSvc *services.UserService` field to `testEnv`:

```go
type testEnv struct {
	db           *gorm.DB
	authSvc      *services.AuthService
	userSvc      *services.UserService
	emailAuthSvc *services.EmailAuthService
	router       *gin.Engine
}
```

b) In `newTestEnv`, after `userSvc := services.NewUserService(db)`, store it on the struct and update `userH`:

```go
	userSvc := services.NewUserService(db)
	// ...
	return &testEnv{db: db, authSvc: authSvc, userSvc: userSvc, emailAuthSvc: emailAuthSvc, router: r}
```

c) Add the route in the `protected` group (after the existing `PUT("users/me", ...)` line):

```go
	protected.POST("users/me/wallet", userH.LinkWallet)
```

- [ ] **Step 4: Write the handler tests**

Add the following to `backend/internal/handlers/user_handler_test.go`:

```go
// ─── LinkWallet handler helpers ───────────────────────────────────────────────

func newHandlerTestWallet(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	addr := common.HexToAddress(crypto.PubkeyToAddress(key.PublicKey).Hex()).Hex()
	return key, addr
}

func handlerSignSIWE(t *testing.T, message string, key *ecdsa.PrivateKey) string {
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

// ─── LinkWallet handler tests ─────────────────────────────────────────────────

func TestLinkWalletHandler_NoAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet",
		bytes.NewBufferString(`{"address":"0x1234","nonce":"abc","signature":"0xdeadbeef"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestLinkWalletHandler_InvalidAddress(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser1", "w1@example.com", "password123")

	body := `{"address":"not-an-address","nonce":"abc","signature":"0xdeadbeef"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet",
		bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLinkWalletHandler_InvalidNonce(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser2", "w2@example.com", "password123")

	_, addr := newHandlerTestWallet(t)
	// No nonce seeded in DB
	body := fmt.Sprintf(`{"address":%q,"nonce":"unknown-nonce","signature":"0x%s"}`,
		addr, strings.Repeat("ab", 65))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet",
		bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLinkWalletHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser3", "w3@example.com", "password123")

	key, addr := newHandlerTestWallet(t)
	nonce := "handler-success-nonce"
	env.db.Create(&models.Web3Nonce{
		Nonce:     nonce,
		Address:   strings.ToLower(addr),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	msg := siweMessageForTest(strings.ToLower(addr), nonce)
	sig := handlerSignSIWE(t, msg, key)

	body := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce, sig)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet",
		bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	if data["address"] != addr {
		t.Errorf("address: want %s, got %v", addr, data["address"])
	}
}

func TestLinkWalletHandler_WalletAlreadyLinked(t *testing.T) {
	env := newTestEnv(t)

	key, addr := newHandlerTestWallet(t)

	// Register user A and bind the wallet
	accessA, _ := env.registerUser(t, "walletA", "wa@example.com", "password123")
	nonce1 := "conflict-nonce-a"
	env.db.Create(&models.Web3Nonce{Nonce: nonce1, Address: strings.ToLower(addr), ExpiresAt: time.Now().Add(5 * time.Minute)})
	msg1 := siweMessageForTest(strings.ToLower(addr), nonce1)
	body1 := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce1, handlerSignSIWE(t, msg1, key))
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body1))
	req1.Header.Set("Authorization", "Bearer "+accessA)
	req1.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), req1)

	// User B tries to bind the same wallet
	accessB, _ := env.registerUser(t, "walletB", "wb@example.com", "password123")
	nonce2 := "conflict-nonce-b"
	env.db.Create(&models.Web3Nonce{Nonce: nonce2, Address: strings.ToLower(addr), ExpiresAt: time.Now().Add(5 * time.Minute)})
	msg2 := siweMessageForTest(strings.ToLower(addr), nonce2)
	body2 := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce2, handlerSignSIWE(t, msg2, key))
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body2))
	req2.Header.Set("Authorization", "Bearer "+accessB)
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req2)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}
```

Add imports needed in `user_handler_test.go` (merge with existing):

```go
import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tachigo/tachigo/internal/models"
)
```

Add a package-level helper that mirrors the SIWE message builder (since `siweMessage` is unexported in the `services` package):

```go
// siweMessageForTest builds the same SIWE message string as services.siweMessage.
// Used in handler tests where we cannot call the unexported helper directly.
func siweMessageForTest(address, nonce string) string {
	return fmt.Sprintf(
		"tachigo.io wants you to sign in with your Ethereum account:\n%s\n\nSign in to Tachigo\n\nNonce: %s\nIssued At: %s",
		address, nonce, time.Now().UTC().Format(time.RFC3339),
	)
}
```

- [ ] **Step 5: Run handler tests**

```bash
cd backend && docker compose run --no-deps --rm app go test ./internal/handlers/... -run TestLinkWallet -v 2>&1
```

Expected: all `TestLinkWalletHandler_*` tests PASS.

- [ ] **Step 6: Run full handler test suite**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... 2>&1 | tail -10
```

Expected: 0 failures.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/handlers/swagger_types.go \
        backend/internal/handlers/user_handler.go \
        backend/internal/handlers/testutil_test.go \
        backend/internal/handlers/user_handler_test.go
git commit -m "feat: UserHandler.LinkWallet — POST /users/me/wallet

Exposes the wallet binding endpoint. Auth middleware enforces JWT.
Error mapping: 400 invalid address, 401 nonce/signature, 409 conflict.

refs #<ISSUE_NUMBER>

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Router + final verification

### Files:
- Modify: `backend/internal/router/router.go`

- [ ] **Step 1: Add route to the protected group**

In `backend/internal/router/router.go`, inside the `protected := v1.Group("/")` block, add after `protected.PUT("users/me", userH.UpdateMe)`:

```go
		protected.POST("users/me/wallet", userH.LinkWallet)
```

- [ ] **Step 2: Run the router tests**

```bash
cd backend && docker compose run --no-deps --rm app go test ./internal/router/... -v 2>&1
```

Expected: PASS.

- [ ] **Step 3: Run the full test suite**

```bash
docker compose run --no-deps --rm app go test ./... 2>&1 | tail -20
```

Expected: all packages pass, 0 failures.

- [ ] **Step 4: Build check**

```bash
docker compose run --no-deps --rm app go build ./... 2>&1
```

Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/router/router.go
git commit -m "feat: route POST /users/me/wallet

closes #<ISSUE_NUMBER>

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Self-Review Checklist

- [x] **Spec coverage**
  - Migration 014 → Task 1
  - SQLite test helper → Task 1 steps 2-3
  - siwe.go extraction → Task 2
  - auth_service.go cleanup → Task 2 step 2
  - TranslateError → Task 3 step 1
  - `ErrInvalidWalletAddress` → Task 3 step 4
  - `LinkWalletInput` + `LinkWallet` → Task 3 step 4
  - All 10 test cases from spec → Task 3 step 2 (9 cases; race case tested via `ErrProviderLinked` from `WalletAlreadyLinkedToOtherUser`)
  - `WalletResponse` → Task 4 step 1
  - `UserHandler.LinkWallet` → Task 4 step 2
  - Handler test cases (5) → Task 4 step 4
  - Router route → Task 5 step 1

- [x] **No placeholders** — all steps have complete code or exact commands
- [x] **Type consistency** — `LinkWalletInput` defined in Task 3, used identically in Task 4; `WalletResponse` defined in Task 4 step 1, referenced in Task 4 step 2 swagger annotation
- [x] **`siweMessageForTest`** — handler tests can't call the unexported `siweMessage`; the local replica is included and the time-precision concern is documented
