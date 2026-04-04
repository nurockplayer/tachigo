# GitHub Issues Draft (Bilingual CN/EN)

> 使用方式：每一段 `Issue` 內容可直接貼到 GitHub New Issue。  
> Suggested label set: `feature` + (`frontend`/`backend`/`web3`/`gameplay`)。

---

## Issue 1

### Title
`[Feature] 挖礦角色點擊增益 / Mining Character Click Boost`

### Body

## 背景與目標 / Background & Goal
目前觀看點數主要來自 heartbeat 累積，互動性較弱。此功能讓看台觀眾可透過點擊挖礦角色，提供額外點數增益，體驗類似忠誠點數互動。

EN:
Current watch points are mostly passive. This feature introduces click-based interaction so viewers can actively boost mining output and earn extra points.

## 功能需求 / Requirements
- 新增「挖礦角色點擊事件」API 與前端事件上報
- 點擊提供額外點數（可配置倍率/固定值）
- 每位觀眾加入冷卻與速率限制（anti-spam）
- 即時顯示點擊增益回饋（特效 + 數值浮字）

EN:
- Add click event API and frontend event reporting.
- Grant configurable bonus points from clicks.
- Add per-viewer cooldown/rate limit anti-spam.
- Show immediate visual feedback for click rewards.

## 驗收條件 / Acceptance Criteria
- [ ] 點擊事件可被記錄，且正確計入點數帳本
- [ ] 單一觀眾超頻點擊不會突破速率限制
- [ ] UI 可即時顯示本次點擊收益
- [ ] 事件有基礎監控指標（QPS、拒絕率、成功率）

## 備註 / Notes
依賴既有 watch points/ledger 模型；建議先以 off-chain 點數實作。

---

## Issue 2

### Title
`[Feature] 安琪拉之門社群史詩任務 / Angela Gate Community Epic Mission`

### Body

## 背景與目標 / Background & Goal
建立全看台共同進度條，觀眾點擊與觀看行為可共同推進「安琪拉之門」任務。任務完成後觸發頻道專屬 NFT 解鎖條件。

EN:
Introduce a channel-wide epic mission where all viewers contribute to a shared progress bar (“Angela Gate”). Completion unlocks channel-exclusive NFT eligibility.

## 功能需求 / Requirements
- 任務進度模型（目標值、目前值、活動期間）
- 進度來源整合：觀看點數、點擊事件、可選加成事件
- 任務狀態機：未開始/進行中/完成/已結算
- 任務完成事件（供 NFT 流程與前端公告使用）

EN:
- Mission progress model (target/current/time window).
- Aggregate contributions from watch points and click events.
- Mission state machine lifecycle.
- Completion event for NFT and UI announcement flows.

## 驗收條件 / Acceptance Criteria
- [ ] 多位觀眾可同時推進同一任務且數據一致
- [ ] 任務完成會產生可追蹤事件記錄
- [ ] 任務畫面可顯示剩餘進度與預估完成時間（可選）
- [ ] 可由主播/管理端設定任務檔期與目標

## 備註 / Notes
需先決定是否賽季制（seasonal reset）與完成後重置策略。

---

## Issue 3

### Title
`[Feature] 時裝與裝備加成系統 / Costume and Equipment Buff System`

### Body

## 背景與目標 / Background & Goal
建立角色成長循環，讓觀眾可透過時裝/裝備改變外觀並獲得點數加成，提升長期參與動機。

EN:
Add progression through costumes and equipment that change avatar visuals and provide gameplay buffs.

## 功能需求 / Requirements
- 物品資料模型（稀有度、槽位、加成類型、數值）
- 角色裝備欄位（可穿戴、可替換、加成生效）
- 時裝外觀同步到挖礦角色動畫
- 與點數計算串接（最終倍率/加值）

EN:
- Item model (rarity, slot, buff type/value).
- Equip/unequip logic and active buff application.
- Costume visuals synced to mining character animation.
- Integrate buffs into points calculation pipeline.

## 驗收條件 / Acceptance Criteria
- [ ] 裝備變更後，角色外觀立即更新
- [ ] 加成可正確反映在點數收益結果
- [ ] 同類型加成疊加規則有明確定義並可測試
- [ ] 至少一組 starter 物品可完整跑通流程

## 備註 / Notes
MVP 建議先使用 off-chain inventory，後續再討論 NFT 化。

---

## Issue 4

### Title
`[Feature] 主播/經紀公司全體 PUFF 技能 / Streamer & Agency Global PUFF Skill`

### Body

## 背景與目標 / Background & Goal
提供主播與經紀公司可主動施放的全體加成技能 `PUFF`，提升直播檔期內活動峰值與社群參與。

EN:
Enable streamers/agencies to cast a global “PUFF” buff that temporarily boosts viewer point gains.

## 功能需求 / Requirements
- `PUFF` 技能配置（倍率、持續時間、冷卻時間）
- 施放權限（streamer / agency admin）
- 施放廣播與倒數狀態同步
- 與觀眾加成系統的疊加規則

EN:
- Configurable PUFF buff (multiplier, duration, cooldown).
- Role-based cast permission.
- Broadcast active status and countdown.
- Deterministic stacking rules with equipment/click buffs.

## 驗收條件 / Acceptance Criteria
- [ ] 只有有權限者可施放 PUFF
- [ ] PUFF 生效期間點數增益正確
- [ ] 冷卻中不可重複施放
- [ ] 直播間可看到 PUFF 生效提示與剩餘時間

## 備註 / Notes
需要先定義 buff 計算優先序（加法後乘法或全乘法）。

---

## Issue 5

### Title
`[Feature] 放置手遊化前端與獨立介面 / Idle-Game Frontend with Independent Panels`

### Body

## 背景與目標 / Background & Goal
把目前前端升級為類放置手遊介面，至少提供可切換的「裝備」「任務」「商城」三個獨立介面，保持清晰資訊架構。

EN:
Rework frontend into an idle-game style interface with independent Equipment, Mission, and Shop panels.

## 功能需求 / Requirements
- 首頁主視覺：挖礦角色、資源條、互動按鈕
- 獨立介面：裝備頁、任務頁、商城頁
- 導航按鈕/分頁切換與狀態保留
- 響應式設計（桌面與 Twitch 面板尺寸）

EN:
- Main idle-game HUD with miner avatar and resources.
- Independent Equipment/Mission/Shop screens.
- Navigation and state-preserving panel switching.
- Responsive behavior for desktop and Twitch panel constraints.

## 驗收條件 / Acceptance Criteria
- [ ] 三個介面可由按鈕獨立開啟且互不覆蓋資料邏輯
- [ ] 切頁後關鍵狀態（裝備選擇/任務追蹤）不遺失
- [ ] 手機窄寬度與 Twitch 面板都可用
- [ ] UI 元件命名與樣式 token 一致

## 備註 / Notes
視覺可參考楓之谷式 UI 結構，但避免直接拷貝素材。

---

## Issue 6

### Title
`[Feature] 頻道專屬 NFT 解鎖流程 / Channel-exclusive NFT Unlock Flow`

### Body

## 背景與目標 / Background & Goal
當安琪拉之門史詩任務達標後，開啟頻道專屬 NFT 解鎖流程，作為社群共創里程碑獎勵。

EN:
After Angela Gate mission completion, enable a channel-exclusive NFT unlock and claim flow as a community milestone reward.

## 功能需求 / Requirements
- 任務完成 -> NFT 可鑄造狀態同步
- 頻道/活動維度的 NFT 合約或 collection 規劃
- 觀眾資格判定（是否參與、門檻）
- 前端顯示可領取/已領取狀態

EN:
- Mission completion toggles NFT claimability.
- Channel/event scoped contract or collection strategy.
- Eligibility checks (participation thresholds).
- Frontend claim status indicators.

## 驗收條件 / Acceptance Criteria
- [ ] 任務完成後可觸發 NFT 解鎖事件
- [ ] 符合資格觀眾可完成領取流程
- [ ] 不符合資格者收到明確原因
- [ ] 交易失敗可重試並保留狀態一致性

## 備註 / Notes
此議題可在 Phase 2+ 交付，MVP 先完成 off-chain 模擬流程亦可。

---

## Dependency Map (建議)

1. Issue 1 -> Issue 2 -> Issue 6
2. Issue 1 -> Issue 3 -> Issue 4
3. Issue 1 + Issue 2 + Issue 3 + Issue 4 -> Issue 5
