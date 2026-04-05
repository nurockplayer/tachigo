# 實況主數據管理頁面

**Issue**：[#69](https://github.com/nurockplayer/tachigo/issues/69)
**Branch**：`feat/streamers-page`（從 `feat/dashboard-auth` 拉，等 Refine 遷移 PR merge 後才改為從 `develop` 拉）
**PR 目標**：`develop`
**狀態**：已完成

---

## 背景

Streamer 需要查看自己頻道的數據統計；Agency 需要查看旗下各實況主的數據；Admin 可查看全部實況主。

`develop` 的 `StreamersPage` 目前只是 `<h1>實況主管理</h1>` placeholder，`StreamerDetailPage` 根本不存在。此計畫從 `feat/dashboard-auth` branch 延伸（因為 Refine 框架在那裡），完整實作列表頁與詳細頁，並串接後端 API。

---

## 前置依賴

| 依賴 | 狀態 | 說明 |
|---|---|---|
| feat/dashboard-auth（Refine 遷移） | ✅ 已包含在本 PR | Refine、authProvider、authProvider.getPermissions() 均已就位 |
| #28（StreamerService 擴充） | ⏳ 待實作 | `ListChannels` 目前只回傳 `Streamer` model，缺統計欄位；`GetChannelStats` 缺 `unique_miners`、`avg_session_seconds`、`total_token_minted` |
| #68（挖礦倍率後端） | ✅ 已 merge 至 develop | `ChannelConfig` 已有 `Multiplier` 欄位（default 1）；`GET /channels/:id/config` 已實作 |

> **實作策略**：後端 API 尚未完成的欄位，前端一律預留欄位但顯示 `—`（dash），不 block UI。等後端 merge 後再移除佔位符。

---

## 架構決策

| 決策 | 說明 |
|---|---|
| API service 獨立成新檔案 | 新建 `src/services/channels.ts`，import `client` from `api.ts`，不修改 `api.ts` |
| 角色判斷用 Refine `usePermissions()` | authProvider 已從 JWT 讀取 role，`feat/dashboard-auth` 才有 Refine |
| Streamer 直接跳轉 | `StreamersPage` 資料載入後，若 role === `streamer` 則 navigate 到 `/streamers/:channel_id` |
| Response 解包 | 所有 API 回傳格式為 `{ success: true, data: { key: ... } }`，需正確解開 |
| 後端缺少欄位預留 | `unique_miners`、`avg_session_seconds`、`total_token_minted`、`multiplier` 等欄位後端未完成時顯示 `—` |
| `StreamerDetailPage` 新建 | develop 無此頁面，需建立並在 `App.tsx` 加路由 `/streamers/:streamerId` |

---

## 待實作 Checklist

### UI 元件

- [x] 新增 `dashboard/src/components/ui/skeleton.tsx`

### App.tsx 更新

- [x] 在 `App.tsx` 新增路由 `/streamers/:streamerId` → `<StreamerDetailPage />`（feat/dashboard-auth 已包含）

### API Service（`src/services/channels.ts`）

型別定義（對應後端現有欄位）：

```ts
// GET /api/v1/dashboard/streamers/channels
// 後端回傳: { success: true, data: { channels: Streamer[] } }
// Streamer model 目前只有基本欄位，統計欄位需等 #28
interface ChannelListItem {
  id: string
  channel_id: string
  display_name: string
  // 以下欄位等 #28 後端擴充後才有，目前前端先定義、顯示 — 佔位
  daily_seconds?: number
  unique_miners?: number
  total_token_minted?: number
}

// GET /api/v1/dashboard/channels/:id/stats
// 後端回傳: { success: true, data: { stats: BroadcastStats } }
// BroadcastStats 目前只有 4 個時間維度，缺礦工/停留/點數，需等 #28
interface ChannelStats {
  current_session_seconds: number
  daily_seconds: number
  monthly_seconds: number
  yearly_seconds: number
  // 以下欄位等 #28 後端擴充
  unique_miners?: number
  avg_session_seconds?: number
  total_token_minted?: number
}

// GET /api/v1/dashboard/channels/:id/config — #68 已實作
// PUT /api/v1/dashboard/channels/:id/config — body: { seconds_per_point?, multiplier? }
interface ChannelConfig {
  channel_id: string
  seconds_per_point: number
  multiplier?: number
}
```

函式：
- [x] `getStreamerChannels()` → `GET /api/v1/dashboard/streamers/channels`，解包 `res.data.data.channels`
- [x] `getChannelStats(channelId)` → `GET /api/v1/dashboard/channels/:id/stats`，解包 `res.data.data.stats`
- [x] `getChannelConfig(channelId)` → `GET /api/v1/dashboard/channels/:id/config`（#68 已實作），解包 `res.data.data.config`

### StreamersPage（`src/pages/StreamersPage.tsx`）

- [x] 呼叫 `getStreamerChannels()` 取代 placeholder
- [x] 載入中顯示 3 行 Skeleton（全寬，h-11）
- [x] API 失敗顯示錯誤訊息
- [x] 用 `usePermissions<string>()` 取得 role：role === `streamer` → 載入完成後 navigate 到 `/streamers/:channel_id`
- [x] 欄位：實況主名稱（`display_name`）、本日開台（`daily_seconds` 若有，否則顯示 `—`）、挖礦觀眾（`unique_miners` 若有）、總產出點數（`total_token_minted` 若有）
- [x] 點擊列導向 `/streamers/:channel_id`

### StreamerDetailPage（新建 `src/pages/StreamerDetailPage.tsx`）

- [x] 從 URL params 取得 `streamerId`（即 `channel_id`）
- [x] `Promise.all([getChannelStats(streamerId), getChannelConfig(streamerId)])` — config 若失敗則降級（顯示 `—`）
- [x] 載入中各 section 顯示 Skeleton
- [x] 時數換算：`seconds / 3600`，一位小數（`toFixed(1)`）
- [x] 平均停留換算：`Math.round(avg_session_seconds / 60)`，若無資料顯示 `—`
- [x] 每分鐘產出（`seconds_per_point` 和 `multiplier` 都有時才計算，否則顯示 `—`）
- [x] role === `streamer` → 隱藏「← 返回列表」按鈕
- [x] 「空投」「調整倍率」按鈕啟用，`onClick={() => console.log('TODO: #71')}`，等 #71 接入

---

## API 規格（後端現況）

```
GET /api/v1/dashboard/streamers/channels
  需要 JWT（RoleAdmin 或 RoleStreamer）
  ⚠️ RoleAgency 未包含在 middleware，Agency 用戶會收到 403，等 #28
  回傳: { success: true, data: { channels: Streamer[] } }
  Streamer: { id, user_id, channel_id, display_name, created_at, updated_at }
  ⚠️ 統計欄位（daily_seconds 等）需等 #28

GET /api/v1/dashboard/channels/:channel_id/stats
  需要 JWT；非 admin 需擁有該 channel
  回傳: { success: true, data: { stats: BroadcastStats } }
  BroadcastStats: { current_session_seconds, daily_seconds, monthly_seconds, yearly_seconds }
  ⚠️ unique_miners、avg_session_seconds、total_token_minted 需等 #28

GET /api/v1/dashboard/channels/:channel_id/config
  ✅ #68 已實作，回傳 { config: { channel_id, seconds_per_point, multiplier } }
  config 不存在時後端回傳預設值（seconds_per_point=60, multiplier=1），不會 404

PUT /api/v1/dashboard/channels/:channel_id/config
  ✅ body: { seconds_per_point?: number, multiplier?: number }（至少一個必填）
```

---

## 驗證方式

```bash
cd dashboard
pnpm build   # 無 TypeScript 錯誤
pnpm lint    # 無 lint 錯誤
```

手動確認：
- Admin 帳號 → `/streamers` 顯示列表，點入進詳細頁 ✅
- Streamer 帳號 → 自動跳轉至 `/streamers/:id` ✅
- Agency 帳號 → 後端 403（等 #28），前端顯示錯誤訊息 ✅（降級正確）
- 詳細頁：四個時數卡（有資料）、三個 metric 卡（後端未完成顯示 `—`）、倍率設定區塊（#68 已就緒） ✅
- 載入中有 skeleton ✅

---

## 完成條件（對應 issue #69）

- [ ] Agency / Admin 列表正常顯示（Agency 只看旗下）
  - Admin ✅；Agency ⛔ 後端 `RequireRole` 未包含 `RoleAgency`，等 #28 後端修正
- [x] Streamer 登入後直接進入自己頻道詳細頁
- [x] 詳細頁頂部右側「空投」與「調整倍率」按鈕已啟用（功能由 #71 實作）
- [x] 開台時數以分組卡片呈現四個維度
- [x] 挖礦參與人數、觀眾平均停留、總產出點數以獨立小卡呈現（後端 #28 完成前顯示 `—`）
- [x] 顯示挖礦倍率設定區塊（#68 已就緒，seconds_per_point + multiplier + 每分鐘產出計算均正確）
- [x] 資料載入中顯示 skeleton
- [x] PR 指向 `develop`（pnpm test 16 passed、pnpm build、pnpm lint 全數通過）
