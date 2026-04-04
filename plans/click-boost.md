# 挖礦角色點擊增益（Click Boost）

> **狀態：** 待實作
> **關聯 Issue：** refs #77
> **最後更新：** 2026-04-04

---

## 背景

目前觀看點數完全被動（heartbeat 每 60 秒 +1 點），互動性不足。
此功能讓觀眾可以點擊 Extension 中的挖礦角色，觸發即時點數增益，
體驗類似忠誠點數互動，同時透過冷卻機制防止刷點。

依賴基礎設施：
- 雙帳本系統（`PointsLedger` + `PointsTransaction`）— PR #74 已合併
- Watch session 管理（`WatchSession`）— PR #52 已合併
- Extension JWT 認證中間件（`ext_auth.go`）

---

## 架構決策

| 項目 | 決策 | 理由 |
|---|---|---|
| 冷卻儲存位置 | 在 `watch_sessions` 新增 `click_cooldown_until` 欄位 | 不新增表，與現有 session 鎖定機制共用 `SELECT FOR UPDATE` |
| 預設冷卻時間 | 5 秒 / 觀眾 | 體感即時但防止 macro 暴力點擊 |
| 每次點擊獎勵 | 固定 1 點（MVP），後續可接 ChannelConfig | 先驗證流程，再做可調參數 |
| 速率拒絕行為 | 回傳 `429`，附帶 `retry_after_ms` | 前端可據此控制 UI 冷卻倒數 |
| TxSource | 新增 `"click"` 值 | 帳本可區分 watch_time / bits / click / spend 來源 |
| 前端冷卻控制 | 樂觀 UI：點擊後立即灰掉按鈕並倒數，不等 API 回應 | 降低延遲感，API 429 時重置倒數即可 |
| 視覺反饋 | 浮字（+1）+ CSS keyframe 動畫，不引入新套件 | 與現有 `isAnimating` 模式一致 |

---

## 資料庫變更

### Migration：新增 click_cooldown_until

```sql
-- migrations/XXXXXX_add_click_cooldown_to_watch_sessions.sql
ALTER TABLE watch_sessions
  ADD COLUMN click_cooldown_until TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01';
```

無需索引，每次查詢都走 session 的 `(user_id, channel_id, is_active)` 索引。

---

## 後端實作

### 1. Model 變更

**`backend/internal/models/points.go`**

```go
const (
    TxSourceWatchTime TxSource = "watch_time"
    TxSourceBits      TxSource = "bits"
    TxSourceClick     TxSource = "click"   // 新增
    TxSourceSpend     TxSource = "spend"
)
```

**`backend/internal/models/watch_session.go`**

```go
type WatchSession struct {
    // ... 現有欄位 ...
    ClickCooldownUntil time.Time `gorm:"not null;default:'1970-01-01'" json:"-"`
}
```

### 2. Service 新方法

**`backend/internal/services/watch_service.go`**

新增常數：

```go
const (
    clickCooldown      = 5 * time.Second
    clickPointsPerClick = int64(1)
)
```

新增方法 `RecordClick(userID, channelID uuid.UUID) (balanceAfter int64, err error)`：

```
1. BEGIN TRANSACTION
2. SELECT session WHERE user_id=? AND channel_id=? AND is_active=true FOR UPDATE
   → 無活躍 session：回傳 ErrNoActiveSession（前端顯示「請先開始觀看」）
3. 檢查 click_cooldown_until > NOW()
   → 若是：回傳 ErrClickOnCooldown{RetryAfterMs: ...}
4. UPDATE watch_sessions SET click_cooldown_until = NOW() + 5s
5. UPSERT points_ledgers（+1 spendable, +1 cumulative）
6. INSERT points_transactions（source="click", delta=1, session_id=session.ID）
7. COMMIT
8. 回傳 balanceAfter
```

### 3. Handler 新方法

**`backend/internal/handlers/watch_handler.go`**

```go
// POST /api/v1/extension/watch/click
func (h *WatchHandler) Click(c *gin.Context) {
    userID  := // 從 JWT 取
    var req struct {
        ChannelID string `json:"channel_id" binding:"required"`
    }
    // binding...
    balance, err := h.watchSvc.RecordClick(userID, channelID)
    switch {
    case errors.Is(err, services.ErrNoActiveSession):
        c.JSON(400, gin.H{"error": "no_active_session"})
    case errors.As(err, &cooldownErr):
        c.JSON(429, gin.H{"error": "on_cooldown", "retry_after_ms": cooldownErr.RetryAfterMs})
    case err != nil:
        c.JSON(500, gin.H{"error": "internal"})
    default:
        c.JSON(200, gin.H{"balance": balance, "delta": 1})
    }
}
```

### 4. 路由

**`backend/internal/router/router.go`**

在 `extension` 受保護路由組新增（與 heartbeat 同層）：

```go
protected.POST("/watch/click", watchHandler.Click)
```

---

## 前端實作

### 1. API 函式

**`tachimint/src/services/api.ts`**

```typescript
export async function sendClick(channelId: string): Promise<{
  balance: number
  delta: number
} | { error: string; retry_after_ms?: number }> {
  const res = await apiClient.post('/api/v1/extension/watch/click', {
    channel_id: channelId,
  })
  return res.data
}
```

### 2. useClickBoost Hook

**`tachimint/src/hooks/useClickBoost.ts`**（新增檔案）

```typescript
interface UseClickBoostResult {
  handleClick: () => void
  cooldownMs: number          // 0 = 可點擊
  isAnimating: boolean        // 浮字動畫開關
  gain: number | null         // 本次獲得點數
  balance: number | null      // 更新後餘額（供外層同步）
}
```

狀態機：
- `idle` → 點擊 → `pending`（樂觀 UI 立即啟動冷卻倒數）
- `pending` → API 回 200 → `animating`（顯示浮字 1500ms）→ `idle`
- `pending` → API 回 429 → 以 `retry_after_ms` 重設倒數

### 3. App.tsx 整合

在 Viewer 視圖中：
- 引入 `useClickBoost`，將 `balance` 同步給現有餘額顯示
- 在挖礦角色圖案上綁定 `onClick={handleClick}`
- `cooldownMs > 0` 時對圖案套用灰階 + `cursor: not-allowed`
- `isAnimating` 時顯示 `+{gain}` 浮字（絕對定位，CSS keyframe 向上淡出）

---

## 待實作 Checklist

### 後端

- [ ] Migration：`watch_sessions.click_cooldown_until`
- [ ] `models/points.go`：新增 `TxSourceClick`
- [ ] `models/watch_session.go`：新增 `ClickCooldownUntil` 欄位
- [ ] `services/watch_service.go`：實作 `RecordClick()`
- [ ] `handlers/watch_handler.go`：實作 `Click()` handler
- [ ] `router/router.go`：掛載 `POST /watch/click`
- [ ] 單元測試：`RecordClick` 正常 / 無 session / 冷卻中
- [ ] 整合測試：`POST /watch/click` 200 / 400 / 429

### 前端

- [ ] `services/api.ts`：新增 `sendClick()`
- [ ] `hooks/useClickBoost.ts`：實作 hook（含樂觀 UI 冷卻倒數）
- [ ] `App.tsx`：整合 `useClickBoost`，綁定點擊事件
- [ ] 視覺反饋：浮字動畫（`+1`）+ 冷卻灰階效果
- [ ] 冷卻倒數進度條（選做）

---

## 驗證方式

1. **正常點擊**：觀眾點擊 → 餘額 +1，浮字顯示 +1，5 秒內按鈕灰掉
2. **冷卻中**：5 秒內再點 → API 回 429，UI 倒數不重置（從剩餘時間繼續）
3. **無 session**：未開始觀看時點擊 → 回 400，UI 顯示提示
4. **帳本一致性**：`PointsTransaction` source="click" 有正確記錄，`PointsLedger` 餘額正確累加
5. **並發**：兩個 tab 同時點擊 → 只有一個成功（DB 鎖保護）
