# Raffle Prize Tiers 設計文件

**日期：** 2026-05-21
**狀態：** 待實作
**關聯 Issue：** #234

---

## 背景

現有抽獎系統每場抽獎只有一條抽獎序列，無法在同一批名單下辦多個獎項。本功能在現有 Raffle 上新增一層 Prize Tier，讓 Streamer 可以在同一場抽獎裡依序抽出一等獎、二等獎等多個獎項，已中獎者自動排除於後續獎項之外。

---

## 設計決策

| 項目 | 決策 |
|---|---|
| 排除規則 | 已中任一獎項者，自動排除於後續所有獎項 |
| 獎項設定時機 | 可隨時新增（邊抽邊加），不需事先全部定好 |
| 抽獎操作 | Streamer 手動逐獎項觸發，每次抽一人 |
| 獎項欄位 | 名稱、獎品描述、抽幾人 |
| Discord 通知 | 整個 Raffle 共用一個 webhook，通知內容附上獎項名稱與描述 |
| 前端範圍 | 後端 API + Dashboard UI 一起實作 |

---

## 資料結構

### 新增：`raffle_prize_tiers` 表

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID | 主鍵 |
| `raffle_id` | UUID | 所屬抽獎（FK → raffles.id） |
| `name` | VARCHAR | 獎項名稱（例：一等獎） |
| `prize_description` | TEXT | 獎品描述（例：Switch 主機） |
| `winner_count` | INT | 此獎項要抽幾人 |
| `drawn_count` | INT | 已抽幾人（系統自動維護） |
| `position` | INT | 排序（1 = 最優先） |
| `created_at` | TIMESTAMP | 建立時間 |
| `updated_at` | TIMESTAMP | 更新時間 |

索引：`(raffle_id, position)`

### 現有：`raffle_draws` 表（新增欄位）

| 欄位 | 型別 | 說明 |
|---|---|---|
| `prize_tier_id` | UUID (nullable) | 關聯的獎項（NULL 表示舊資料或無獎項模式） |

`prize_tier_id` nullable，確保向後相容。

---

## API 端點

所有端點需 JWT 認證 + RoleStreamer，前綴 `/api/v1/dashboard`。

### 獎項管理

| 方法 | 路徑 | 說明 |
|---|---|---|
| `POST` | `/raffles/{id}/prize-tiers` | 新增獎項 |
| `GET` | `/raffles/{id}/prize-tiers` | 列出所有獎項 |
| `PATCH` | `/raffles/{id}/prize-tiers/{tier_id}` | 修改獎項（只能改尚未抽完的） |
| `DELETE` | `/raffles/{id}/prize-tiers/{tier_id}` | 刪除獎項（只能刪 drawn_count = 0 的） |

### 抽獎

| 方法 | 路徑 | 說明 |
|---|---|---|
| `POST` | `/raffles/{id}/prize-tiers/{tier_id}/draws` | 觸發此獎項抽一人 |

#### POST prize-tiers Request Body
```json
{
  "name": "一等獎",
  "prize_description": "Switch 主機",
  "winner_count": 1
}
```

#### POST prize-tiers/{tier_id}/draws 邏輯
1. 確認 tier 屬於此 raffle，且 raffle 狀態為 `active`
2. 確認 `drawn_count < winner_count`，否則回傳 `409 Conflict`
3. 查出此 raffle 所有已中獎的 `entry_id`（跨所有 tier）
4. 從 `raffle_entries` 隨機抽一筆，排除上述已中獎者
5. 建立 `RaffleDraw`（帶 `prize_tier_id`），`drawn_count + 1`
6. 發送 Discord webhook（附獎項名稱與描述）、發中獎通知信

---

## 業務邏輯

### 排除機制

```
已中獎的 entry_id = 
  SELECT entry_id FROM raffle_draws 
  WHERE raffle_id = :raffle_id
```

DrawNext 從 `raffle_entries` 抽取時，排除上述所有 entry_id，無論中的是哪個 tier。

### 防呆規則

| 情境 | 回應 |
|---|---|
| `drawn_count >= winner_count` | `409` 此獎項已抽完 |
| Raffle 狀態非 `active` | `422` 抽獎未啟動 |
| 名單已全部中獎，無人可抽 | `409` 名單已耗盡 |
| 刪除已有中獎記錄的 tier | `422` 無法刪除已抽過的獎項 |
| 修改已抽完的 `winner_count` 至低於 `drawn_count` | `422` 不可低於已抽人數 |

---

## 前端改動

### `RaffleDetailPage`（現有頁面，新增區塊）

**新增「獎項」區塊**，位於名單與中獎記錄之間：

```
獎項
├── 一等獎｜Switch 主機｜已抽 1 / 1 人   [抽一人（灰色，已完成）]
├── 二等獎｜貼圖包｜已抽 0 / 3 人        [抽一人]
└── [+ 新增獎項]
```

- 「新增獎項」：彈出表單，填名稱、獎品描述、人數
- 「抽一人」：呼叫 `POST prize-tiers/{tier_id}/draws`，即時更新 `drawn_count`
- 已抽完的獎項按鈕 disable
- 有中獎記錄的獎項不顯示「刪除」

**中獎記錄區塊（現有，微調）**

每筆記錄新增「獎項」欄位，顯示 `prize_tier.name`（若 tier_id 為 null 則顯示「-」）。

### 新增 API 客戶端方法（`services/raffles.ts`）

- `createPrizeTier(raffleId, data)`
- `listPrizeTiers(raffleId)`
- `updatePrizeTier(raffleId, tierId, data)`
- `deletePrizeTier(raffleId, tierId)`
- `drawFromTier(raffleId, tierId)`

---

## 不做（本票明確排除）

- 每個獎項各自設定 Discord webhook
- 一鍵自動抽完所有獎項
- Campaign / 活動頂層概念
- Extension 畫面顯示獎項分層
- 中獎者領獎流程的獎項資訊顯示（/claim 頁面不改）

---

## Migration

新增一個 Atlas migration：
1. 建立 `raffle_prize_tiers` 表
2. `raffle_draws` 加 `prize_tier_id` 欄位（nullable UUID，FK → raffle_prize_tiers.id）
