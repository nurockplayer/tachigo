# POST /users/me/wallet — 已登入 Twitch 使用者綁定 MetaMask

**狀態**：待實作

## 背景

目前 `POST /auth/web3/nonce` + `POST /auth/web3/verify` 是「用錢包**登入**」的流程。已用 Twitch 登入的使用者沒有辦法把 MetaMask 錢包綁到自己的帳號，導致 `ClaimService.Claim()` 呼叫 `resolveWalletAddress()` 時找不到 `provider='web3'` 的 AuthProvider，回傳 `ErrClaimWalletNotLinked`。

Demo 階段的 `wallet_linker` helper 已在 #243 移除，正式綁定流程需補上。

---

## 設計決策

| 問題 | 決策 |
|---|---|
| 每個 user 幾個錢包？ | 只允許一個；綁新的自動取代舊的 |
| nonce 端點 | 複用現有 `POST /auth/web3/nonce`（public），binding 本身需 JWT |
| 舊 web3 row 處理方式 | Soft delete（保留歷史），restore 而非 insert 若同 user 重綁同一地址 |
| SIWE helper 位置 | 抽到 `services/siwe.go`，package-level unexported，`auth_service` 與 `user_service` 共用 |

---

## API 規格

### 前置：取 nonce（現有端點，不動）

```
POST /api/v1/auth/web3/nonce
Content-Type: application/json

{"address": "0xAbCd..."}

→ 200 {"success":true,"data":{"nonce":"<hex64>"}}
```

### 新端點：綁定錢包

```
POST /api/v1/users/me/wallet
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "address":   "0xAbCd...",   // EIP-55 or lowercase, 後端統一 checksum
  "nonce":     "<hex64>",
  "signature": "0x<hex130>"
}

→ 200 {"success":true,"data":{"address":"0xAbCd..."}}  // checksummed
```

### Error mapping

| 情況 | HTTP | error 欄位 |
|---|---|---|
| address 格式不合法 | 400 | `invalid wallet address` |
| nonce 不存在或過期 | 401 | `invalid or expired nonce` |
| 簽名驗證失敗 | 401 | `invalid wallet signature` |
| address 已被其他 user 綁定 | 409 | `wallet already linked to another account` |
| DB 錯誤 | 500 | `internal server error` |

---

## 資料庫變更

### migration 014：將全域 unique index 改為 partial unique index

現有 `001_init.sql` 有：
```sql
UNIQUE (provider, provider_id)
```

Soft delete 後舊 row 的 `deleted_at IS NOT NULL`，insert 新 row 時會踩到這個 constraint。需新增 migration：

```sql
-- backend/migrations/014_auth_provider_partial_unique.sql

-- 移除全域 unique constraint
ALTER TABLE auth_providers DROP CONSTRAINT IF EXISTS auth_providers_provider_provider_id_key;

-- 改為 partial unique index（只約束 active row）
CREATE UNIQUE INDEX IF NOT EXISTS
    idx_auth_providers_provider_provider_id_active
ON auth_providers(provider, provider_id)
WHERE deleted_at IS NULL;
```

> PostgreSQL 支援 partial index；專案其他 migration 已有先例。

### SQLite（測試環境）

SQLite 不支援 `CREATE UNIQUE INDEX ... WHERE`。測試 helper 需在 schema 初始化時跳過這個 index，或改用 `uniqueIndex` tag 搭配 test-specific migration。具體做法在 implementation plan 階段決定，spec 僅標注此注意事項。

---

## 新增 / 修改物件清單

### 1. `backend/internal/services/siwe.go`（新增）

Package-level unexported helpers，`auth_service.go` 與 `user_service.go` 都位於 `services` package，可直接呼叫。

```
siweMessage(address, nonce string) string
verifyEthSignature(message, sigHex, expectedAddress string) bool
```

- 從 `auth_service.go` 移過來（原處改為呼叫此處的函式，不重複定義）

### 2. `backend/internal/services/user_service.go`（新增方法）

```go
type LinkWalletInput struct {
    Address   string `json:"address"   binding:"required"`
    Nonce     string `json:"nonce"     binding:"required"`
    Signature string `json:"signature" binding:"required"`
}

func (s *UserService) LinkWallet(userID uuid.UUID, input LinkWalletInput) (string, error)
```

**LinkWallet 內部流程（transaction 外 → transaction 內）：**

```
[transaction 外]
1. common.IsHexAddress(input.Address) → false → ErrInvalidWalletAddress
2. checksumAddr = common.HexToAddress(input.Address).Hex()
3. lookupAddr  = strings.ToLower(input.Address)
4. db.Where("nonce=? AND address=?", input.Nonce, lookupAddr).First(&nonceRecord)
   → not found or expired → ErrInvalidNonce
5. msg = siweMessage(lookupAddr, input.Nonce)
   verifyEthSignature(msg, input.Signature, lookupAddr) → false → ErrInvalidSignature

[BEGIN TRANSACTION]
6. db.Where("nonce=? AND address=?", ...).Delete(&Web3Nonce{})  // consume nonce
7. db.Unscoped().
       Where("provider='web3' AND provider_id=? AND deleted_at IS NULL AND user_id != ?",
             checksumAddr, userID).
       Count(&count)
   → count > 0 → ErrProviderLinked

8. db.Where("user_id=? AND provider='web3' AND deleted_at IS NULL", userID).
       Update("deleted_at", now)  // soft delete 目前 active web3 row（若有）

9. db.Unscoped().
       Where("user_id=? AND provider='web3' AND provider_id=?", userID, checksumAddr).
       First(&ap)
   → 找到 soft-deleted row → db.Unscoped().Model(&ap).Update("deleted_at", nil)
   → 找不到              → db.Create(&AuthProvider{provider:'web3', provider_id: checksumAddr})

[COMMIT]
10. return checksumAddr, nil
```

**新增 sentinel errors（在 `user_service.go` 宣告）：**

```go
ErrInvalidWalletAddress = errors.New("invalid wallet address")
```

沿用現有：`ErrInvalidNonce`、`ErrInvalidSignature`、`ErrProviderLinked`（已在 `auth_service.go`，同 package 可直接用）

### 3. `backend/internal/handlers/user_handler.go`（新增方法）

```go
// LinkWallet godoc
// @Summary      Bind a MetaMask wallet address to the current user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body services.LinkWalletInput true "address + nonce + signature"
// @Success      200  {object}  Response{data=WalletResponse}
// @Failure      400  {object}  Response
// @Failure      401  {object}  Response
// @Failure      409  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/wallet [post]
func (h *UserHandler) LinkWallet(c *gin.Context)
```

- `MustClaims(c)` 取 userID
- 呼叫 `userSvc.LinkWallet(userID, input)`
- error switch：
  - `ErrInvalidWalletAddress` → `badRequest`
  - `ErrInvalidNonce` → `unauthorized`
  - `ErrInvalidSignature` → `unauthorized`
  - `ErrProviderLinked` → `conflict("wallet already linked to another account")`
  - default → `internal`

### 4. `backend/internal/router/router.go`（修改）

在 `protected` group 新增：

```go
protected.POST("users/me/wallet", userH.LinkWallet)
```

---

## 測試規格

### `services/user_service_test.go`（新增 test cases）

| Case | 預期結果 |
|---|---|
| 合法 address + nonce + 簽名，首次綁定 | 200，AuthProvider insert，checksumAddr 回傳 |
| 合法，已有 active web3 row → 取代 | 舊 row soft deleted，新 row insert |
| 同 user 重綁同一地址（soft-deleted row 存在） | restore deleted_at = NULL，不 insert 新 row |
| nonce 不存在 | `ErrInvalidNonce` |
| nonce 已過期 | `ErrInvalidNonce` |
| 簽名錯誤 | `ErrInvalidSignature` |
| address 已被其他 user 綁定 | `ErrProviderLinked` |
| address 格式不合法 | `ErrInvalidWalletAddress` |

### `handlers/user_handler_test.go`（新增 test cases）

HTTP 層測試，驗證 status code 與 response body，不重複 service 邏輯。

---

## 本票明確不做

- `GET /users/me/wallet`（查詢已綁錢包）
- 解綁走現有 `DELETE /auth/providers/web3`，不改動
- 前端（tachimint / dashboard）串接
- 不修改 `ClaimService.resolveWalletAddress()`
- 不改 `POST /auth/web3/nonce` 或 `POST /auth/web3/verify`
