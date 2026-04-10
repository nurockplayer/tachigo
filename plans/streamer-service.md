# StreamerService — 實況主管理與數據統計

> **狀態：** 進行中
> **關聯 Issue：** refs #28
> **最後更新：** 2026-04-04

---

## 背景

#27（PointsService 雙帳本）已完成，`broadcast_time_stats` 與 `broadcast_time_logs` 的寫入邏輯（`AddBroadcastTime`）和查詢邏輯（`GetBroadcastStats`）已就位。#68（ChannelConfig / dashboard routes）已建立 `/api/v1/dashboard/` route group，並以 `RequireRole(Admin, Streamer)` 保護。

本次任務目標：

1. 建立 `streamers` 表作為「用戶—頻道」管理映射，補充 `auth_providers` 的隱式關係
2. 實作 `StreamerService`：封裝頻道管理與統計查詢
3. 新增 Dashboard API：讓實況主可查詢自己的頻道播出統計與觀眾數據
4. 前述 `PointsService.GetBroadcastStats` 已有完整實作，`StreamerHandler` 呼叫即可

設計細節見 [docs/architecture.md](../docs/architecture.md)。

---

## 架構決策

| 項目 | 決策 | 理由 |
|---|---|---|
| Migration 編號 | `005_streamers.sql` | `004_channel_config.sql` 與 `004_rbac_roles.sql` 均已存在，下一個為 005 |
| `streamers` 表設計 | `(user_id, channel_id)` unique pair，channel_id 為 Twitch channel ID string | 一位 user 可管理多個頻道；與 auth_providers.provider_id 對應 |
| GetStats 的 streamerID 查找 | 由 `channelID → auth_providers WHERE provider='twitch' AND provider_id=channelID` 取得 streamerID | 與 `PointsService.AddBroadcastTime` 相同模式，保持一致 |
| Dashboard 路由前綴 | `/api/v1/dashboard/` | 與現有 `PUT /dashboard/channels/:channel_id/config` 一致 |
| 經紀公司權限邊界 | Agency 可查詢旗下頻道，但不可查詢其他 agency 的頻道（MVP 以 role 區分，不做細粒度 ownership check） | 降低實作複雜度 |
| `GetBroadcastStats` | 直接呼叫 `PointsService.GetBroadcastStats(streamerID, channelID)` | 邏輯已在 #27 完整實作，無需重複 |

---

## 待實作 Checklist

### 1. Migration

- [ ] `backend/migrations/005_streamers.sql`

```sql
CREATE TABLE IF NOT EXISTS streamers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id  VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, channel_id)
);

CREATE INDEX idx_streamers_channel_id ON streamers(channel_id);
```

### 2. Model

- [ ] `backend/internal/models/streamer.go`

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

type Streamer struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
    ChannelID   string    `gorm:"type:varchar(255);not null;index"`
    DisplayName string    `gorm:"type:varchar(255)"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

`BeforeCreate` hook 參考其他 models，呼叫 `newUUID()` 產生 UUID。

### 3. StreamerService

- [ ] `backend/internal/services/streamer_service.go`

```go
type StreamerService struct {
    db        *gorm.DB
    pointsSvc *PointsService
}

func NewStreamerService(db *gorm.DB, pointsSvc *PointsService) *StreamerService

// Register 將 user 與 channelID 綁定為 streamer（upsert）。
// channelID 必須與 auth_providers.provider_id 對應的 Twitch channel 一致。
func (s *StreamerService) Register(userID uuid.UUID, channelID, displayName string) (*models.Streamer, error)

// ListChannels 回傳 user 管理的所有頻道。
func (s *StreamerService) ListChannels(userID uuid.UUID) ([]models.Streamer, error)

// GetChannelStats 查詢指定頻道的播出統計。
// 內部從 auth_providers 解析 streamerID，再呼叫 PointsService.GetBroadcastStats。
func (s *StreamerService) GetChannelStats(channelID string) (*BroadcastStats, error)
```

`GetChannelStats` 查找流程：
1. `SELECT user_id FROM auth_providers WHERE provider='twitch' AND provider_id=channelID`
2. 若找不到 → 回傳 `ErrStreamerNotFound`
3. 呼叫 `s.pointsSvc.GetBroadcastStats(streamerID, channelID)`

### 4. Handler

- [ ] `backend/internal/handlers/streamer_handler.go`

```go
type StreamerHandler struct {
    streamerSvc *services.StreamerService
}

func NewStreamerHandler(svc *services.StreamerService) *StreamerHandler

// POST /dashboard/streamers/register
// Body: { "channel_id": "...", "display_name": "..." }
func (h *StreamerHandler) Register(c *gin.Context)

// GET /dashboard/streamers/channels
// 回傳當前登入 user 管理的所有頻道
func (h *StreamerHandler) ListChannels(c *gin.Context)

// GET /dashboard/channels/:channel_id/stats
// 回傳頻道播出統計（current session / daily / monthly / yearly）
func (h *StreamerHandler) GetChannelStats(c *gin.Context)
```

### 5. Router

- [ ] `backend/internal/router/router.go`
  - `New()` 加入 `streamerSvc *services.StreamerService` 參數
  - 在 `dashboard` group 新增路由：

```go
streamerH := handlers.NewStreamerHandler(streamerSvc)

dashboard.POST("/streamers/register", streamerH.Register)
dashboard.GET("/streamers/channels", streamerH.ListChannels)
dashboard.GET("/channels/:channel_id/stats", streamerH.GetChannelStats)
```

### 6. Main.go

- [ ] `backend/cmd/server/main.go`
  - `AutoMigrate` 加入 `&models.Streamer{}`
  - 初始化：`streamerSvc := services.NewStreamerService(db, pointsSvc)`
  - 傳入 `router.New()`

### 7. Test DB Util

- [ ] `backend/internal/services/testutil_test.go` — 在 `migrateTestDB` 加入 `streamers` 表 DDL：

```sql
CREATE TABLE IF NOT EXISTS streamers (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    channel_id TEXT NOT NULL,
    display_name TEXT,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_streamers_user_channel
    ON streamers (user_id, channel_id);
```

### 8. 測試

- [ ] `backend/internal/services/streamer_service_test.go`

| 測試案例 | 描述 |
|---|---|
| `TestRegister_OK` | 正常建立 streamer，可重複呼叫（upsert） |
| `TestRegister_UpdateDisplayName` | 同一 (user, channel) 再次 Register 更新 display_name |
| `TestListChannels_Empty` | 尚未綁定頻道時回傳空 slice |
| `TestListChannels_MultipleChannels` | 一個 user 綁定多個頻道 |
| `TestGetChannelStats_NoStreamer` | channelID 無對應 auth_providers 時回傳 ErrStreamerNotFound |
| `TestGetChannelStats_OK` | 正常查詢，委派給 PointsService.GetBroadcastStats |

---

## 驗證方式

```bash
docker compose run --no-deps --rm app go test ./...
```

手動流程：

1. 以 Streamer 帳號登入取得 JWT
2. `POST /api/v1/dashboard/streamers/register` body `{"channel_id": "<twitch_channel_id>", "display_name": "測試實況主"}`
3. `GET /api/v1/dashboard/streamers/channels` → 確認回傳剛綁定的頻道
4. 以 Viewer 帳號觸發 heartbeat，累積播出時間
5. `GET /api/v1/dashboard/channels/<channel_id>/stats` → 確認 `daily_seconds` 有數值
6. 以非 Streamer/Admin 帳號呼叫任一 `/dashboard/` endpoint → 確認回傳 403
