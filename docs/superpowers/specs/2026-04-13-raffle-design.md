# 訂閱者抽獎系統設計

**日期：** 2026-04-13  
**狀態：** 草稿，待 GitHub 討論  
**相關票：** 待開

---

## 背景

實況主目前的做法是月底 31 號 23:59 手動下載訂閱名單，再自己找抽獎工具操作。目標是將整個流程整合進 tachigo，讓主播不需要離開平台即可完成訂閱者抽獎，降低操作成本，提高安裝意願。

---

## 範圍邊界

**本設計包含：**
- 抽獎活動管理（Dashboard）
- 訂閱者快照（CSV 匯入 + Twitch API）
- 排程快照（月底結算用）
- 逐一抽獎流程（直播中手動操作）
- 中獎者領獎連結與收件資訊收集
- Extension 公佈中獎結果

**本設計不包含：**
- 通知管道實作（Email、Discord 各自獨立開票）
- 物流處理（交由 tachiya 負責）
- Campaign 多場抽獎層（Phase 2 補票）
- 補位自動重抽（補位由主播手動再按抽下一名）

---

## 資料模型

### raffles — 一場抽獎活動

| 欄位 | 型別 | 說明 |
|------|------|------|
| id | uuid PK | |
| streamer_id | uuid FK → streamers | |
| title | varchar(255) | 活動名稱 |
| status | enum | `draft` → `snapshot_ready` → `live` → `completed`（第一次抽獎時自動從 `snapshot_ready` 轉 `live`） |
| source | enum | `csv` / `twitch_api` |
| filter_config | jsonb | 資格條件（訂閱 tier、最低累積點數等） |
| snapshot_at | timestamptz | 實際快照時間（排程或手動觸發後填入） |
| scheduled_at | timestamptz | 排程快照時間（null = 手動） |
| claim_deadline_hours | int | 領獎期限（小時） |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### raffle_entries — 符合資格的參賽名單（快照）

| 欄位 | 型別 | 說明 |
|------|------|------|
| id | uuid PK | |
| raffle_id | uuid FK → raffles | |
| user_id | uuid FK → users | 必須有 tachigo 帳號 |
| twitch_user_id | varchar(255) | Twitch user ID（索引用） |
| display_name | varchar(255) | |
| tier | smallint | 訂閱等級 1 / 2 / 3 |
| cumulative_pts | int8 | 快照當下的累積點數 |
| is_eligible | bool | 是否通過 filter_config 過濾 |
| drawn_at | timestamptz | 被抽中的時間（null = 未抽中） |

### raffle_draws — 每次抽出的結果

| 欄位 | 型別 | 說明 |
|------|------|------|
| id | uuid PK | |
| raffle_id | uuid FK → raffles | |
| entry_id | uuid FK → raffle_entries | |
| draw_order | int | 第幾名（1, 2, 3...） |
| status | enum | `pending` / `claimed` / `expired` / `redrawn` |
| claim_token | uuid | 領獎連結用的 token（一次性） |
| claim_expires_at | timestamptz | |
| created_at | timestamptz | |

### raffle_claims — 中獎者填寫的收件資訊

| 欄位 | 型別 | 說明 |
|------|------|------|
| id | uuid PK | |
| draw_id | uuid FK → raffle_draws | |
| recipient_name | varchar(255) | |
| phone | varchar(50) | |
| address_json | jsonb | 交給 tachiya 的原始格式 |
| submitted_at | timestamptz | |

---

## API 設計

### Dashboard（實況主操作，需身份驗證）

```
POST   /raffles                          建立抽獎活動
GET    /raffles                          列出我的抽獎活動
GET    /raffles/:id                      查看單場活動詳情
POST   /raffles/:id/snapshot             手動觸發快照（CSV 上傳或 Twitch API 同步）
POST   /raffles/:id/entries/import-csv   上傳 CSV 匯入名單
POST   /raffles/:id/draws                抽出下一名
GET    /raffles/:id/draws                查看目前抽獎結果列表
POST   /raffles/:id/complete             結束活動
```

### 排程（內部 cron job）

```
每天 23:55 掃描 scheduled_at 即將到期的 raffles → 觸發快照
```

### 中獎者領獎（公開，不需登入）

```
GET    /claim/:token                     查看領獎頁資訊（確認 token 有效）
POST   /claim/:token                     提交收件資訊
```

### Extension 公佈結果（觀眾端）

```
GET    /ext/raffles/:id/result           查看目前已抽出的中獎名單
```

---

## 流程設計

### 快照流程（排程或手動）

```
觸發快照
  → source=twitch_api：呼叫 Twitch Helix API 拉訂閱名單
  → source=csv：解析上傳的 CSV
  → 比對 tachigo users（auth_providers WHERE provider='twitch'）
  → 套用 filter_config（tier、累積點數門檻）
  → 寫入 raffle_entries，is_eligible 標記
  → raffle.status → snapshot_ready，填入 snapshot_at
```

### 抽獎流程（直播中主播手動操作）

```
主播在 Dashboard 按「抽下一名」
  → 從 is_eligible=true 且 drawn_at=null 的 entries 隨機抽一筆
  → 寫入 raffle_draws（draw_order 遞增）
  → entry.drawn_at 標記時間
  → 產生 claim_token，計算 claim_expires_at
  → 觸發通知（email / Discord，各自獨立票實作）
  → Extension GET /ext/raffles/:id/result 即時顯示新中獎者
```

### 領獎流程

```
中獎者收到通知（含領獎連結 /claim/:token）
  → GET /claim/:token → 確認 token 有效且未過期
  → 填寫收件資訊 → POST /claim/:token
  → raffle_claims 建立，draw.status → claimed
  → 通知 tachiya（webhook 或 API，本系統只送資料）

若超過 claim_expires_at 未領：
  → draw.status → expired
  → 主播在 Dashboard 手動再按「抽下一名」補位
```

---

## 待討論事項（GitHub）

- [ ] filter_config jsonb 的具體 schema（tier 門檻、點數門檻的格式）
- [ ] Twitch Helix 訂閱名單 API 的 scope 需求與 token 管理
- [ ] CSV 格式規範（欄位名稱、編碼）
- [ ] claim_deadline_hours 預設值（24h？72h？）
- [ ] Extension 結果頁的 UI 設計（只顯示最新一名 vs 全部名單）
- [ ] 通知管道實作順序（Email 優先 vs Discord 優先）
- [ ] tachiya webhook 介面規格
- [ ] Phase 2：Campaign 多場抽獎層設計

---

## 未來規劃（本票不做）

- **Phase 2**：Campaign 層，同一份快照辦多場不同獎項的抽獎
- **通知**：Email 通知（獨立票）、Discord 通知（獨立票）
- **物流**：tachiya 端的收件資訊處理與出貨追蹤
