# Ocean Mining 角色系統 — Phase 1 實作計劃

狀態：規劃中（待 Codex review）

來源討論：[discussion #710](https://github.com/nurockplayer/tachigo/discussions/710) — 海洋挖礦角色系統設計

---

## 背景

discussion #710 是 `[discussion]` 設計票，把現有的 capybara 被動挖礦 HUD 擴充成
「螃蟹 🦀 → 海豚 🐬 / 海龜 🐢 → 鯨魚 🐋」四角色挖礦系統。本計劃只涵蓋設計書的
**Phase 1 範圍**（設計書第 9 節），把它拆成可獨立 merge 的小 PR，並草擬對應的
`[backend]` / `[frontend]` GitHub issue。

`[discussion]` 票不直接實作；本計劃的產出是「把討論結論轉成一組 feature issue」。

---

## Repo 現況校正（重要）

設計書「關鍵檔案／模組」段落引用的路徑與實際 repo **不符**，實作時以下表為準：

| 設計書寫的 | 實際 repo |
|---|---|
| `tachimint/` sidepanel | **`apps/extension/`**（`tachimint/` 目前是空資料夾） |
| `tachimint/src/components/MarioHUD/` | `apps/extension/src/app/components/MarioHUD.tsx`（單檔 24KB capybara HUD） |
| `backend/internal/models/`、`services/` | `services/api/internal/models/`、`services/api/internal/services/` |
| `migrations/015_*.sql` | 本 repo 用 GORM AutoMigrate，migration 慣例見 `services/api/cmd/server/main.go` |

**後端現況**（`services/api/internal/`）：已有 watch session / click / heartbeat、
每頻道 points ledger（`PointsLedger.cumulative_total` + `spendable_balance`）、
claim → $TACHI balance、coupon redeem。**完全沒有**角色、XP、熟悉度、排行榜、任務。

相關現有檔案：
- `models/points.go`、`models/watch_session.go`、`models/channel_config.go`
- `services/watch_service.go`（StartSession / Heartbeat / RecordClick / EndSession）
- `services/points_service.go`（AddWatchTime / AddHeartbeatTime / AddPointsWithMeta）
- `router/router.go`（`/extension/watch/{start,heartbeat,click,end,balance}`）
- 前端 `apps/extension/src/app/components/MarioHUD.tsx`、`hooks/useTwitch.ts`

---

## 架構決策

1. **角色資料是 per-user 全域，不是 per-channel。**
   `PointsLedger` 是 per-channel，但「解鎖即永久擁有」「角色 XP 各自獨立累積」
   「進化永久保留」都是跨頻道的。→ 新表 `user_characters` 以 `(user_id, character)`
   為單位；`active_character` 與切換 cooldown 也是 per-user 全域單一狀態。

2. **熟悉度是 per (user × streamer)。** 只影響 T-Points 產出，不影響 XP 產出
   （設計書第 7 節「水土不服」）。→ 新表 `streamer_familiarity` 記錄跨 session
   累積觀看秒數。

3. **Contract 變更先行。** 依 CLAUDE.md「PR 不得依賴未 merge 的 PR」，所有後端
   contract（schema、API、heartbeat payload 欄位）必須先 merge 進 `develop`，
   前端 PR 才能開。本計劃的 PR 順序已據此排好。

4. **heartbeat payload 需擴充。** 海豚 S1（聊天則數）與海龜 S1（連續觀看秒數）
   的 buff 計算需要前端把 `chat_count`、`continuous_seconds` 帶進 heartbeat。
   這是 contract 變更，放在後端 PR-B4。

5. **Phase 1 鯨魚只做 UI 剪影**，不做 buff / 解鎖邏輯（設計書第 9 節）。
   S2/S3 buff 只預留 hook，UI 顯示「未解鎖」。

6. **美術資產是外部依賴。** 3 角色 × 3 stage 的 Q 版立繪 + 鯨魚剪影需設計提供；
   程式 PR 先用 placeholder，資產到位後另開小 PR 替換。此依賴需在 issue 標註。

---

## 待 review / 待確認問題

- [ ] 美術資產（角色立繪、剪影、進化特效）由誰、何時提供？Phase 1 是否接受 placeholder 上線？
- [ ] content script 注入 `*://*.twitch.tv/*` 屬於 Manifest 權限擴張，需確認 extension 審查 / 上架影響。
- [ ] 進化門檻、解鎖價格、buff 數值皆為設計書「待校準」值；Phase 1 直接採建議預設，OK？
- [ ] 海龜「中斷 5 分鐘歸零」由後端依 `last_heartbeat_at` 判定 — 與現有 watch session 既有的中斷邏輯是否衝突需確認。
- [ ] `apps/extension` 與 `extensions/tachigo-demo-sidepanel` 程式幾乎重複；本計劃只動 `apps/extension`，demo 是否同步不在範圍內。

---

## PR 拆分總表

順序即依賴順序；後端 contract 全部 merge 後前端才開。每張預估 < 400 行，
超過者於該 issue 註明可再拆。

| # | PR | 類型 | 依賴 | 預估行數 |
|---|---|---|---|---|
| B1 | 角色 / 熟悉度 schema + model | `[backend]` | — | ~250（含 migration） |
| B2 | 熟悉度 service + 整合進 T-Points 產出 | `[backend]` | B1 | ~300 |
| B3 | character service：解鎖 / 切換 / 進化 | `[backend]` | B1 | ~350 |
| B4 | heartbeat/click contract 擴充 + XP 累積 | `[backend]` | B1 | ~300 |
| B5 | S1 buff 計算（螃蟹 / 海豚 / 海龜） | `[backend]` | B3, B4 | ~350 |
| B6 | character handler + routes + swagger | `[backend]` | B3, B5 | ~300 |
| F0 | characters.ts domain module（純函式） | `[frontend]` | — | ~250 |
| F1 | content script chatDetector + manifest | `[frontend]` | — | ~150 |
| F2 | ActiveCharacterDisplay（替換 capybara） | `[frontend]` | B6, F0 | ~350 |
| F3 | CharacterMenu 抽屜 + 切換 cooldown UI | `[frontend]` | B6, F0 | ~350 |
| F4 | BuffList + FamiliarityIndicator + EvolutionPrompt + coach marks | `[frontend]` | B6, F0, F2 | ~380 |

F0 / F1 無後端依賴，可與後端並行開工。F2–F4 需等 B6 merge。
合計 11 PR，與設計書「Phase 1 約 2-3 週」一致。

---

## GitHub Issue 草稿

> 以下為各 PR 對應 issue 的 body 草稿。標題前綴、label、完成條件格式依
> CLAUDE.md「GitHub Issue 慣例」。target branch 一律 `develop`。

### B1 — `[backend] 角色系統 — user_characters / streamer_familiarity schema`

**背景**
角色挖礦系統需要 per-user 的角色擁有狀態與 per (user×streamer) 的熟悉度記帳。
現有 `PointsLedger` 是 per-channel，不適合存全域角色資料。

**任務**
- [ ] 新增 model `models/user_character.go`：`user_id`、`character`（enum: crab/dolphin/turtle/whale/capybara）、`unlocked`、`stage`(1-3)、`xp`、`unlocked_at`
- [ ] 新增 per-user 全域狀態欄位：`active_character`、`switch_cooldown_until`（放新 model `models/user_character_state.go` 或 `user.go`，二擇一於 PR 說明）
- [ ] 新增 model `models/streamer_familiarity.go`：`user_id`、`channel_id`、`cumulative_watch_seconds`、`last_watched_at`
- [ ] 在 `services/api/cmd/server/main.go` AutoMigrate 清單註冊新 model + 必要的 unique index `(user_id, character)`、`(user_id, channel_id)`
- [ ] model 單元測試（BeforeCreate UUID v7、預設值）

**介面／規格**
- `Character` 型別為 `string` enum；新使用者預設擁有 crab（unlocked、stage 1、active）
- capybara 為 dev 專屬角色，enum 保留但不在主流程解鎖

**參考**
`models/points.go`（GORM 慣例、UUID v7 BeforeCreate）、`models/watch_session.go`

**完成條件**
- [ ] AutoMigrate 在乾淨 DB 成功建表
- [ ] `docker compose run --no-deps --rm app go test ./...` 通過

**本票明確不做**：service / handler / router / 前端 / buff 邏輯 / migration 以外的行為。

---

### B2 — `[backend] 熟悉度 service — 水土不服曲線 + T-Points 產出整合`

**背景**
防炸魚核心機制（設計書第 7 節）：T-Points 產出乘上 per-streamer 熟悉度，
強角色到新台只能拿 10% 產出。

**任務**
- [ ] 新增 `services/familiarity_service.go`：依 `cumulative_watch_seconds` 算熟悉度
  - 0→60min：10%→50% 線性；60→300min：50%→100% 線性
  - 30 天未觀看後每天 -1%，下限 10%
- [ ] heartbeat tick 累加 `cumulative_watch_seconds` 並更新 `last_watched_at`
- [ ] 在 `points_service.go` T-Points 產出路徑套用 `familiarity`（XP 路徑不套）
- [ ] 單元測試：曲線插值邊界、衰減、新台 10% 驗證

**介面／規格**
```text
T-Points 產出 = 基礎活動值 × familiarity(streamer) × (1 + buffs)
XP 產出       = 基礎活動值 × (1 + buffs)   // 不套熟悉度
```

**參考**
`services/points_service.go`（AddWatchTime / addPointsWithMetaAt）、`services/watch_service.go`

**完成條件**
- [ ] 新台熟悉度測試顯示 10%；累積 5hr 顯示 100%
- [ ] 既有 points / watch 測試不回歸
- [ ] 後端測試全通過

**本票明確不做**：角色 buff、XP（B4）、handler、前端。

---

### B3 — `[backend] character service — 解鎖 / 切換 / 進化`

**背景**
角色解鎖（扣 `spendable_balance`）、切換（30 分鐘 cooldown + 海豚例外）、
進化（XP 門檻 → stage）的核心邏輯。

**任務**
- [ ] 新增 `services/character_service.go`
- [ ] `Unlock`：海豚需 `spendable_balance ≥ 50`、海龜 ≥ 1500；扣款後永久擁有；鯨魚回傳「特殊管道」錯誤不可解鎖
- [ ] `Switch`：切換後 30 分鐘 cooldown；**唯一例外**——切「進入」海豚不受 cooldown 限制，但進入後原 cooldown 繼續倒數、結束前不可從海豚切出
- [ ] `Evolve`：S1→2 需 1000 XP、S2→3 需 10000 XP（Phase 1 只啟用 S1→2）；使用者主動觸發
- [ ] 單元測試覆蓋 cooldown 海豚例外的情境範例（設計書第 4 節 5 步驟）

**介面／規格**
- 解鎖以 `spendable_balance` 是否足夠為準，扣的也是 `spendable_balance`
- 切換 cooldown 是 per-user 全域單一狀態

**參考**
`services/spend_service.go`（扣款 / balance 檢查慣例）、設計書第 4、5、6 節

**完成條件**
- [ ] cooldown 海豚例外的 5 步情境測試全綠
- [ ] 餘額不足、重複解鎖、鯨魚解鎖皆有對應錯誤
- [ ] 後端測試全通過

**本票明確不做**：buff 計算、XP 累積（B4）、handler、前端。

---

### B4 — `[backend] heartbeat/click contract 擴充 + 角色 XP 累積`

**背景**
海豚／海龜 S1 buff 需要前端回傳聊天則數與連續觀看秒數；active 角色每 tick 累積 XP。

**任務**
- [ ] heartbeat request 擴充：`chat_count`、`continuous_seconds`（向後相容，缺值視為 0）
- [ ] heartbeat / click tick 對 **active 角色** 累積 XP（離線角色 XP = 0）
- [ ] 海龜中斷判定：後端 5 分鐘未收到 heartbeat → `continuous_seconds` 視為歸零
- [ ] 更新 swagger annotation；若改路由則 `swag init`
- [ ] 單元測試 + 既有 heartbeat/click 測試相容

**介面／規格**
```jsonc
// POST /extension/watch/heartbeat
{ "channel_id": "...", "chat_count": 0, "continuous_seconds": 0 }
```

**參考**
`handlers/watch_handler.go`（`watchBody`）、`services/watch_service.go`（Heartbeat / RecordClick）

**完成條件**
- [ ] 舊 client（無新欄位）heartbeat 仍正常
- [ ] active 角色 XP 隨 tick 增加，切換後新角色才累積
- [ ] swagger docs 同 PR 更新
- [ ] 後端測試全通過

**本票明確不做**：buff 倍率計算（B5）、解鎖／進化（B3）、前端。

---

### B5 — `[backend] S1 buff 計算 — 螃蟹 / 海豚 / 海龜`

**背景**
3 隻常規角色的 stage 1 buff（波動鉗擊 / 回聲波紋 / 龜養生息），套用到 T-Points/XP 產出。

**任務**
- [ ] 螃蟹 S1：每 60s tick 10 次有效點擊上限，1.5×；超過上限不累積
- [ ] 海豚 S1：1 則 +40% / 2 則 +60% / 3 則以上 +80%，S1 上限 +100%（品質加成 Phase 2）
- [ ] 海龜 S1：連續觀看 30min +10% / 60min +20% / 90min +35%，中斷 >5min 歸零
- [ ] buff 疊加採加法：`multiplier = 1.0 + Σbuff`
- [ ] S2/S3 預留 hook（介面留空、回傳 0），不實作
- [ ] 單元測試覆蓋各 buff 邊界與上限

**介面／規格**
`CharacterBuff(character, stage, tickContext) float64` — 純函式，tickContext 含 clicks/chat/continuousSeconds

**參考**
設計書第 3 節 buff 矩陣、`channel_config.go`（既有 multiplier 慣例）

**完成條件**
- [ ] 各角色 S1 上限值（螃蟹 1.5×、海豚 +100%、海龜 1.35×）測試正確
- [ ] 後端測試全通過

**本票明確不做**：鯨魚 buff、S2/S3 實作、handler、前端。

---

### B6 — `[backend] character handler + routes + swagger`

**背景**
把 character / familiarity 狀態與操作開成 extension 用的 API。

**任務**
- [ ] 新增 `handlers/character_handler.go`
- [ ] `GET /extension/characters` — 回傳擁有角色、stage、XP、active、cooldown 剩餘、熟悉度
- [ ] `POST /extension/characters/unlock`、`POST /extension/characters/switch`、`POST /extension/characters/evolve`
- [ ] router 註冊（extension 已驗證群組內）
- [ ] swagger annotation + `swag init`（docs.go / swagger.json / swagger.yaml 同 PR）
- [ ] handler 層測試

**參考**
`handlers/watch_handler.go`、`router/router.go` extension 群組、CLAUDE.md「Swagger Docs 更新規則」

**完成條件**
- [ ] 4 個 endpoint 可正常回應、錯誤碼正確
- [ ] swagger docs 已更新並含於 PR
- [ ] 後端測試全通過

**本票明確不做**：前端、buff 邏輯變更。

---

### F0 — `[frontend] characters.ts domain module（角色定義 + buff/熟悉度曲線）`

**背景**
前端的純邏輯層：角色定義、buff 計算、熟悉度曲線。無 UI、無 API、可單元測試。

**任務**
- [ ] 新增 `apps/extension/src/app/features/characters/characters.ts`
- [ ] 角色定義（4 角 + capybara dev）、解鎖價格、進化門檻常數
- [ ] buff / 熟悉度曲線純函式（與後端 B2/B5 對齊，用於 UI 預覽顯示）
- [ ] vitest 單元測試

**參考**
設計書第 3、5、6、7 節；`apps/extension/src/types/`

**完成條件**
- [ ] 曲線函式測試與後端值一致
- [ ] `pnpm test` 通過

**本票明確不做**：任何 UI 元件、API 串接、content script。

---

### F1 — `[frontend] content script — Twitch 聊天偵測 + manifest`

**背景**
偵測使用者在 twitch.tv 送出聊天，透過 `chrome.runtime` 通知 sidepanel（海豚 buff 用）。

**任務**
- [ ] 新增 `apps/extension/src/content/chatDetector.ts`：監聽 chat send button / Enter
- [ ] manifest 新增 `content_scripts`（match `*://*.twitch.tv/*`）與 `tabs` 權限
- [ ] sidepanel 端接收 `CHAT_SENT` 訊息並計數
- [ ] selector 失效時 chatCount 自動為 0，不影響其他功能（設計書已註明）

**參考**
設計書「聊天訊息偵測設計」、`apps/extension` 既有 manifest / `extensions/tachigo-demo-sidepanel/src/extension/content.ts`

**完成條件**
- [ ] 在 twitch.tv 送訊息能被 sidepanel 計數
- [ ] selector 失效不報錯、不影響挖礦
- [ ] manifest 權限變更於 PR body 說明

**本票明確不做**：海豚 buff 數值計算（後端 B5）、UI。

---

### F2 — `[frontend] ActiveCharacterDisplay — 替換 capybara HUD`

**背景**
把現有 `MarioHUD.tsx` 的 capybara 容器換成 Active Character 顯示區；capybara 改 dev 專屬。

**任務**
- [ ] 新增 `features/characters/ActiveCharacterDisplay.tsx`：立繪 + 名字 + stage + XP + 進化進度條
- [ ] 串接 `GET /extension/characters`
- [ ] capybara 改為僅 dev build 出現的隱藏選項
- [ ] 角色立繪先用 placeholder（美術資產另開 PR）
- [ ] 元件測試

**參考**
`apps/extension/src/app/components/MarioHUD.tsx`、設計書第 8 節

**完成條件**
- [ ] sidepanel 顯示當前 active 角色與 XP/進化進度
- [ ] capybara 在 production build 不出現、dev build 可選
- [ ] `pnpm test` 通過

**本票明確不做**：角色選單、buff list、進化動畫（F3/F4）。

---

### F3 — `[frontend] CharacterMenu 抽屜 + 切換 cooldown UI`

**背景**
角色選單抽屜：已解鎖角色切換、鎖住角色顯示解鎖條件、cooldown 倒數。

**任務**
- [ ] 新增 `features/characters/CharacterMenu.tsx`
- [ ] 已解鎖：縮圖 + stage + 切換按鈕（顯示 cooldown 剩餘）
- [ ] 鎖住：剪影 + 解鎖條件（缺多少 T-Points）；鯨魚顯示「需透過特別管道解鎖」
- [ ] 切往海豚按鈕永遠可點；進入海豚後切出按鈕顯示倒數 disabled
- [ ] 串接 unlock / switch API
- [ ] 元件測試覆蓋海豚 cooldown 例外 UI 行為

**參考**
設計書第 4、8 節、F0 `characters.ts`

**完成條件**
- [ ] cooldown 海豚例外的 UI 行為符合設計書 5 步情境
- [ ] 解鎖扣款後角色永久可用
- [ ] `pnpm test` 通過

**本票明確不做**：進化動畫、buff list、coach marks（F4）。

---

### F4 — `[frontend] BuffList / FamiliarityIndicator / EvolutionPrompt / coach marks`

**背景**
Phase 1 前端收尾：當前 buff 清單、熟悉度徽章、進化提示與最簡動畫、新手引導。

**任務**
- [ ] `BuffList.tsx`：當前生效 / 未達成 buff 清單（✓ / ✗）
- [ ] `FamiliarityIndicator.tsx`：當前實況主熟悉度徽章
- [ ] `EvolutionPrompt.tsx`：XP 足夠時角色頭上「！」+ 進化按鈕 + 最簡視覺升級
- [ ] 首次開啟 1-2 頁 coach mark，可關閉、不強制
- [ ] 元件測試

**參考**
設計書第 5、8 節

**完成條件**
- [ ] buff list 正確反映後端回傳的生效狀態
- [ ] 進化按鈕觸發後角色升 stage 並保留
- [ ] `pnpm test` 通過

**本票明確不做**：S2/S3 buff、進化粒子特效（Phase 2）、排行榜、任務 UI。

---

## 不在 Phase 1（設計書 Phase 2/3，本計劃明確不做）

排行榜系統、競技獎章、里程碑稱號、實況主限時任務、爆擊 → $TACHI、
鯨魚 buff（S1/S2/S3）、鯊魚角色、進化分支、PvP 食物鏈。
鯨魚 Phase 1 僅止於選單剪影 +「需透過特別管道解鎖」文案。

---

## 端到端驗證（Phase 1 完成後）

對照設計書第 10 節驗證清單，重點：
1. 新使用者：螃蟹 active、其他角色鎖住顯示解鎖條件
2. 累積 50 / 1500 T-Points → 海豚 / 海龜解鎖按鈕亮、扣 `spendable_balance` 後永久擁有
3. 螃蟹→海豚無 cooldown；海豚→螃蟹 cooldown 中被擋
4. cooldown 海豚例外 5 步情境
5. 聊天 / 點擊 / 連續觀看觸發對應 S1 buff
6. 防炸魚：新台熟悉度 10%，累積 1hr → 50%
7. 螃蟹累積 1000 XP → 進化提示 → 升 S2 並保留
8. dev build 可選 capybara、production 不出現
