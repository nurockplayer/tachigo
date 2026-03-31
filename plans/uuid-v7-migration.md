# UUID v7 Migration

> **狀態：** 待實作
> **關聯 Issue：** refs #61
> **最後更新：** 2026-04-01

---

## 背景

所有 model 的 PK 目前用 `uuid.New()`（UUID v4，完全隨機），導致 B-tree index 隨機插入、page split。改用 `uuid.New7()`（UUID v7，時序）可解決此問題。詳見 [docs/uuid-v7.md](../docs/uuid-v7.md)。

---

## 待實作項目

### Model BeforeCreate hooks（改 uuid.New7()）

- [ ] `backend/internal/models/user.go`
- [ ] `backend/internal/models/auth_provider.go`
- [ ] `backend/internal/models/address.go`
- [ ] `backend/internal/models/refresh_token.go`
- [ ] `backend/internal/models/email_auth.go`（EmailVerification、PasswordReset）
- [ ] `backend/internal/models/points.go`（PointsLedger、PointsTransaction）
- [ ] `backend/internal/models/watch_session.go`

### Service 層直接賦值（改 uuid.New7()）

- [ ] `backend/internal/services/watch_service.go:62` — `ID: uuid.New()` for WatchSession
- [ ] `backend/internal/services/extension_service.go:125` — `ID: uuid.New()` for User

### 不需更動

- 測試檔案中的 `uuid.New()` — 測試 ID 無須時序性

---

## 驗證方式

```bash
docker compose run --no-deps --rm app go test ./...
```

確認生成的 UUID 為 v7 格式（開頭 8 個 hex chars 反映時間，不是純隨機）。
