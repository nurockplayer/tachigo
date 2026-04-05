# Points Handler Alignment

> **狀態：** 已完成
> **關聯 Issue：** closes #30
> **關聯 PR：** #95（2026-04-06 merge）
> **最後更新：** 2026-04-06

---

## 背景

Issue #30 要求補齊登入後可用的點數查詢 API，讓 viewer 能依頻道查自己的點數餘額與最近交易歷史，並同步更新 Swagger 文件與必要測試。

---

## 完成條件

- [x] 新增 `PointsHandler`
- [x] `GET /api/v1/users/me/points` 受 JWT 保護
- [x] `GET /api/v1/users/me/points/history` 受 JWT 保護
- [x] `history` 只回最近 50 筆，且依新到舊排序
- [x] 回應資料包含 `type`、`amount`、選填 `sku`、選填 `note`、`created_at`
- [x] 點數帳本維持 channel-scoped
- [x] Swagger 文件已更新
- [x] 服務層與 handler 層測試已補齊

---

## 對應位置

- `backend/internal/handlers/points_handler.go`
- `backend/internal/services/points_service.go`
- `backend/internal/models/points.go`
- `backend/internal/router/router.go`
- `backend/migrations/007_points_transaction_sku.sql`
- `backend/docs/swagger.yaml`
- `backend/internal/services/points_service_test.go`
- `backend/internal/handlers/points_handler_test.go`

---

## 驗證

```bash
cd backend
GOCACHE=/Users/erickwang/Desktop/tachigo/.gocache \
GOMODCACHE=/Users/erickwang/Desktop/tachigo/.gomodcache \
go test ./internal/services -run 'TestPointsService_' -count=1

GOCACHE=/Users/erickwang/Desktop/tachigo/.gocache \
GOMODCACHE=/Users/erickwang/Desktop/tachigo/.gomodcache \
go test ./internal/handlers -run 'TestPointsHandler_|TestSwaggerTypes_' -count=1

GOCACHE=/Users/erickwang/Desktop/tachigo/.gocache \
GOMODCACHE=/Users/erickwang/Desktop/tachigo/.gomodcache \
go test ./... -count=1
```
