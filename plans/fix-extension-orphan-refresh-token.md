# Fix Extension Orphan Refresh Token Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent `CompleteTPointTransaction` from creating orphan refresh token records when points write fails, and fix `issueTokenPair`'s silent DB error.

**Architecture:** The fix moves token issuance to after `AddPointsWithMeta` succeeds. A private `lookupExtensionUser` helper is extracted from `LoginWithExtension` to share user-lookup logic without coupling it to token issuance. `issueTokenPair` is fixed to propagate the `RefreshToken` DB write error instead of silently discarding it. `DeleteExpiredRefreshTokens` is added to `AuthService` as the documented cleanup mechanism for any tokens that escaped before this fix (or from other flows).

**Tech Stack:** Go, GORM, PostgreSQL, `github.com/golang-jwt/jwt/v5`, in-package unit tests (`package services`)

---

## 問題診斷

`CompleteTPointTransaction` 的現行呼叫順序：

```
VerifyExtJWT + VerifyReceiptJWT
  → LoginWithExtension          ← issueTokenPair 在此呼叫，refresh_token 寫入 DB
    → AddPointsWithMeta         ← 如果這裡失敗
      handler 回 error，client 拿不到 token
      但 DB 裡已有 orphan refresh_token record
```

修正後順序：

```
VerifyExtJWT + VerifyReceiptJWT
  → lookupExtensionUser         ← 只找 user，不發 token
    → AddPointsWithMeta         ← 先寫 points
      → issueTokenPair          ← points 成功才發 token
```

## File Map

| 操作 | 檔案 | 說明 |
|------|------|------|
| Modify | `services/api/internal/services/extension_service.go` | 新增 `lookupExtensionUser`，重構 `CompleteTPointTransaction` |
| Modify | `services/api/internal/services/auth_service.go` | 修 `issueTokenPair` error 傳遞；新增 `DeleteExpiredRefreshTokens` |
| Modify (tests) | `services/api/internal/services/extension_service_test.go` | orphan token regression test |
| Modify (tests) | `services/api/internal/services/auth_service_test.go` | `issueTokenPair` error propagation test；`DeleteExpiredRefreshTokens` test |

---

### Task 1: 寫 regression test（預期 RED）

**Files:**
- Modify: `services/api/internal/services/extension_service_test.go`

- [ ] **Step 1: 在 extension_service_test.go 末尾新增 failing test**

`newTestDB(t)` 為每個 test 建立獨立的 in-memory SQLite DB，`DROP TABLE` 不會影響其他 test。`models.RefreshToken` 沒有 `gorm.DeletedAt`，`Count()` 不受 soft-delete scope 污染。

```go
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
```

- [ ] **Step 2: 確認測試 RED**

```bash
docker compose run --no-deps --rm app \
  go test ./internal/services/ -run TestCompleteTPointTransaction_PointsWriteFailure_NoOrphanRefreshToken -v
```

預期輸出包含：`FAIL` 且 `points write failure must not create refresh tokens`

---

### Task 2: 重構 `extension_service.go`

**Files:**
- Modify: `services/api/internal/services/extension_service.go`

- [ ] **Step 1: 在 `LoginWithExtension` 上方新增 `lookupExtensionUser`**

將以下程式碼插入 `extension_service.go` 的 `LoginWithExtension` function 之前（第 115 行附近）：

```go
// lookupExtensionUser resolves a Twitch identity to a tachigo User.
// It does not issue tokens; call issueTokenPair separately.
func (s *ExtensionService) lookupExtensionUser(claims *ExtensionClaims) (*models.User, error) {
	if claims.UserID == "" {
		return nil, ErrInvalidExtJWT
	}
	var provider models.AuthProvider
	err := s.db.Where("provider = ? AND provider_id = ?", models.ProviderTwitch, claims.UserID).
		First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := s.db.First(&user, provider.UserID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
```

- [ ] **Step 2: 更新 `LoginWithExtension` 使用 `lookupExtensionUser`**

將 `LoginWithExtension` body（第 116-148 行）替換為：

```go
func (s *ExtensionService) LoginWithExtension(extJWT string) (*models.User, *TokenPair, error) {
	claims, err := s.VerifyExtJWT(extJWT)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.lookupExtensionUser(claims)
	if err != nil {
		return nil, nil, err
	}
	tokens, err := s.authSvc.issueTokenPair(user)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}
```

- [ ] **Step 3: 重構 `CompleteTPointTransaction`**

將 `CompleteTPointTransaction` body（第 153-206 行）替換為：

```go
func (s *ExtensionService) CompleteTPointTransaction(extJWT, receipt, sku string) (*models.User, *TokenPair, error) {
	extClaims, err := s.VerifyExtJWT(extJWT)
	if err != nil {
		return nil, nil, err
	}

	receiptClaims, err := s.VerifyReceiptJWT(receipt)
	if err != nil {
		return nil, nil, err
	}

	if receiptClaims.Data.SKU != sku {
		return nil, nil, ErrInvalidReceipt
	}
	if receiptClaims.Data.Type != "bits" {
		return nil, nil, ErrInvalidReceiptType
	}
	if receiptClaims.Data.Amount <= 0 {
		return nil, nil, ErrInvalidReceiptAmount
	}
	if receiptClaims.Data.TransactionID == "" {
		return nil, nil, ErrInvalidReceipt
	}
	if len([]rune(sku)) > 255 || len([]rune(receiptClaims.Data.TransactionID)) > 255 {
		return nil, nil, ErrInvalidReceipt
	}

	// Resolve user before touching any write path.
	user, err := s.lookupExtensionUser(extClaims)
	if err != nil {
		return nil, nil, err
	}

	// Write points first; tokens are issued only on success to avoid orphan
	// refresh token records when the points write fails.
	txID := receiptClaims.Data.TransactionID
	err = s.pointsSvc.AddPointsWithMeta(
		user.ID,
		extClaims.ChannelID,
		models.TxSourceTPoint,
		int64(receiptClaims.Data.Amount),
		PointsCreditMeta{
			SKU:                   &sku,
			ExternalTransactionID: &txID,
		},
	)
	if err != nil {
		if isDuplicateExternalTransactionError(err) {
			// ErrDuplicateTransaction also covers the retry case where points were
			// credited in a prior call but token issuance failed. The client should
			// call LoginWithExtension separately to obtain a token.
			return nil, nil, ErrDuplicateTransaction
		}
		return nil, nil, err
	}

	// Points are now committed. If issueTokenPair fails here, points remain credited
	// and the client will receive an error. On retry, AddPointsWithMeta returns
	// ErrDuplicateTransaction — the client should call LoginWithExtension to get tokens.
	tokens, err := s.authSvc.issueTokenPair(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}
```

- [ ] **Step 4: 跑全套 extension 測試，確認 GREEN**

```bash
docker compose run --no-deps --rm app \
  go test ./internal/services/ -run TestCompleteTPoint -v
```

預期：所有 `TestCompleteTPoint*` tests PASS，包含剛加的 orphan test。

- [ ] **Step 5: Commit**

```bash
git add services/api/internal/services/extension_service.go \
        services/api/internal/services/extension_service_test.go
git commit -m "fix: issue tokens after points write in CompleteTPointTransaction

Moves issueTokenPair to after AddPointsWithMeta so a points write
failure cannot leave orphan refresh token records in the DB.
Extracts lookupExtensionUser helper to share user-lookup logic
without coupling it to token issuance.

refs #452"
```

---

### Task 3: 新增 `DeleteExpiredRefreshTokens` 清理機制

**Files:**
- Modify: `services/api/internal/services/auth_service.go`
- Modify: `services/api/internal/services/auth_service_test.go`

- [ ] **Step 1: 寫失敗測試**

在 `auth_service_test.go` 末尾新增：

```go
func TestDeleteExpiredRefreshTokens_RemovesExpiredOnly(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())

	// Register a user to get valid tokens.
	_, err := svc.Register("del_exp@example.com", "username_del_exp", "Password1!")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	tokens, err := svc.Login("del_exp@example.com", "Password1!")
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
```

- [ ] **Step 2: 確認測試 RED**

```bash
docker compose run --no-deps --rm app \
  go test ./internal/services/ -run TestDeleteExpiredRefreshTokens -v
```

預期：FAIL — `svc.DeleteExpiredRefreshTokens undefined`

- [ ] **Step 3: 在 `auth_service.go` 新增方法**

在 `Logout` function 之後新增：

```go
// DeleteExpiredRefreshTokens removes all expired refresh token records.
// Returns the number of rows deleted.
func (s *AuthService) DeleteExpiredRefreshTokens() (int64, error) {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{})
	return result.RowsAffected, result.Error
}
```

- [ ] **Step 4: 確認測試 GREEN**

```bash
docker compose run --no-deps --rm app \
  go test ./internal/services/ -run TestDeleteExpiredRefreshTokens -v
```

預期：PASS

- [ ] **Step 5: 跑全套 services 測試**

```bash
docker compose run --no-deps --rm app go test ./internal/services/ -v 2>&1 | tail -20
```

預期：所有測試 PASS，無 FAIL。

- [ ] **Step 6: Commit**

```bash
git add services/api/internal/services/auth_service.go \
        services/api/internal/services/auth_service_test.go
git commit -m "feat: add DeleteExpiredRefreshTokens to AuthService

Provides a documented, testable cleanup path for expired refresh
token records. Orphans created before the CompleteTPointTransaction
fix and any other future stranded tokens will be pruned by this method.

refs #452"
```

---

### Task 4: 修 `issueTokenPair` 的 `db.Create` 靜默 error

**Files:**
- Modify: `services/api/internal/services/auth_service.go`
- Modify: `services/api/internal/services/auth_service_test.go`

目前 `auth_service.go:351` 的 `s.db.Create(&models.RefreshToken{...})` error 被靜默丟棄。修復後順序中，`issueTokenPair` 在 points 寫入成功後才被呼叫；若此時 `db.Create` 靜默失敗，client 拿到一個在 DB 裡不存在的 refresh token，第一次 `Refresh` 就會 `ErrInvalidToken`。

- [ ] **Step 1: 寫失敗測試（預期 RED）**

在 `auth_service_test.go` 末尾新增：

```go
func TestLogin_RefreshTokensTableGone_ReturnsError(t *testing.T) {
	svc := NewAuthService(newTestDB(t), testConfig())
	if _, err := svc.Register("rt_gone@example.com", "username_rt_gone", "Password1!"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Drop refresh_tokens after Register (which already wrote one) to isolate Login.
	if err := svc.db.Exec("DROP TABLE refresh_tokens").Error; err != nil {
		t.Fatalf("drop refresh_tokens: %v", err)
	}
	// Before fix: Login silently ignores the db.Create error and returns tokens.
	// After fix:  Login propagates the error and returns nil tokens.
	_, err := svc.Login("rt_gone@example.com", "Password1!")
	if err == nil {
		t.Fatal("want error when refresh_tokens table is gone, got nil")
	}
}
```

- [ ] **Step 2: 確認測試 RED**

```bash
docker compose run --no-deps --rm app \
  go test ./internal/services/ -run TestLogin_RefreshTokensTableGone_ReturnsError -v
```

預期：FAIL — `Login` 目前靜默忽略 error，回傳 `nil` error

- [ ] **Step 3: 修 `issueTokenPair` 的 `db.Create` error 傳遞**

在 `auth_service.go` 找到下列程式碼（約第 351 行）：

```go
s.db.Create(&models.RefreshToken{
    UserID:    user.ID,
    TokenHash: hashToken(rawRefresh),
    ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshTTL),
})
```

替換為：

```go
if err := s.db.Create(&models.RefreshToken{
    UserID:    user.ID,
    TokenHash: hashToken(rawRefresh),
    ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshTTL),
}).Error; err != nil {
    return nil, err
}
```

- [ ] **Step 4: 確認測試 GREEN**

```bash
docker compose run --no-deps --rm app \
  go test ./internal/services/ -run TestLogin_RefreshTokensTableGone_ReturnsError -v
```

預期：PASS

- [ ] **Step 5: 跑全套 services 測試確認無 regression**

```bash
docker compose run --no-deps --rm app go test ./internal/services/ -v 2>&1 | tail -20
```

預期：所有測試 PASS，無 FAIL。

- [ ] **Step 6: Commit**

```bash
git add services/api/internal/services/auth_service.go \
        services/api/internal/services/auth_service_test.go
git commit -m "fix: propagate RefreshToken db.Create error in issueTokenPair

Previously the error from db.Create was silently discarded, allowing
callers to receive a token whose DB record did not exist. On Refresh
the token would immediately fail with ErrInvalidToken.

refs #452"
```

---

## 完成條件確認

| 條件 | 確認方式 |
|------|---------|
| points write failure 不產生 orphan refresh token | `TestCompleteTPointTransaction_PointsWriteFailure_NoOrphanRefreshToken` PASS |
| `issueTokenPair` DB error 正確往上傳 | `TestLogin_RefreshTokensTableGone_ReturnsError` PASS |
| API response contract 不變 | `TestCompleteTPointTransaction_Success` 仍 PASS，tokens 仍正常回傳 |
| 已有清理機制 | `DeleteExpiredRefreshTokens` 實作並有 test 覆蓋 |
| duplicate transaction 行為不變 | `TestCompleteTPointTransaction_DuplicateTransactionID_ReturnsErrDuplicate` PASS |
| retry 語意有文件 | `CompleteTPointTransaction` 的 `issueTokenPair` 呼叫處有 code comment 說明 retry 行為 |

## 本計畫明確不做

- 不新增定期排程 cleanup job（TTL 30 天的 token 自然過期，`DeleteExpiredRefreshTokens` 已提供可呼叫的清理介面）
- 不處理 T-point receipt idempotency（#448）
- 不新增 SKU catalog / multiplier 邏輯
- 不用 DB transaction 包住 `AddPointsWithMeta` + `issueTokenPair`（points-success/token-failure 的 partial failure 以 code comment + `ErrDuplicateTransaction` 語意覆蓋，不需 transaction rollback）
