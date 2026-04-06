# 前端 Roadmap

## 目標

為 `tachigo` 建立一份可執行的前端演進路線，避免後續開發同時混雜：

- `dashboard` 的管理後台需求
- `tachimint` 的 Twitch Extension / GameFi 體驗需求
- 共用設計規則、元件與資料狀態管理方式

這份 roadmap 的目的不是一次決定所有視覺細節，而是先回答：

1. 先做哪一個前端產品
2. 每個階段要交付什麼
3. 什麼算完成
4. 哪些事情現在先不要做

---

## 產品切面

### 1. Dashboard

使用者：

- streamer
- agency
- admin

核心目的：

- 查看頻道資料與統計
- 管理頻道設定
- 後續承接 agency / event / 營運型流程

前端特性：

- 任務導向
- 資訊密度高
- 權限與錯誤狀態必須清楚
- 穩定性與可維護性優先於花俏視覺

### 2. Tachimint

使用者：

- viewer
- broadcaster / moderator / external 只需極簡提示畫面

核心目的：

- 讓觀眾在 Twitch Extension 內完成 watch-to-earn / idle-game 體驗
- 建立 click、heartbeat、bits、任務與成長感的前端骨架

前端特性：

- Twitch-native first，GameFi flavor second
- 狀態感與回饋感優先
- 需要嚴格對齊 Twitch / JWT / watch session lifecycle
- 高風險點在資料同步、session 狀態與 UI state ownership

### 3. 共用前端基礎

核心目的：

- 讓不同前端產品共享一致的設計語言與工程規則

包含內容：

- design tokens
- 共用狀態呈現模式
- API contract 對齊方式
- 錯誤處理與 loading/empty/forbidden 規則

---

## 總體策略

### 原則 1：先穩定狀態，再強化視覺

`tachigo` 目前更容易出錯的地方，不是「畫面不夠漂亮」，而是：

- auth flow 與 route guard 不一致
- balance / session / heartbeat 有多份 state
- 前端 request contract 與後端實作不一致
- role-based UI 邊界不清楚

因此 roadmap 採：

1. 先收斂 state inventory 與 API contract
2. 再完成可用 UI 骨架
3. 最後做視覺升級與 GameFi 氛圍

### 原則 2：Dashboard 與 Extension 分開規劃

這兩個產品的使用情境完全不同：

- `dashboard` 要的是穩定管理流
- `tachimint` 要的是互動與節奏

不能共用同一套頁面思維，也不該共用同一份 roadmap phase。

### 原則 3：每個畫面先有 state inventory

每個頁面在進入視覺稿前，都應先列出：

- loading
- success
- empty
- partial data
- forbidden
- error
- retry / cooldown / unavailable

如果 state inventory 不完整，就不要先做高保真設計。

---

## 分階段規劃

## Phase 1：共用基礎與規則收斂

### 目標

建立所有前端工作共用的設計與工程基準。

### 產出物

- 前端設計原則文件
- `tachimint` 視覺原則文件
- state inventory 模板
- API contract checklist
- design tokens 初版
- 共用 UI primitives 規格

### 建議元件

- `PageHeader`
- `SectionCard`
- `StatCard`
- `StatusBadge`
- `EmptyState`
- `LoadingState`
- `ErrorState`
- `ForbiddenState`
- `ConfirmDialog`

### 驗收標準

- 新頁面不再各自發明 loading / error / empty 樣式
- PR review 可以用同一組 state / contract 標準檢查
- `dashboard` 與 `tachimint` 都能引用同一份基礎規則

### 現階段不要做

- 大量高保真設計稿
- 先做完整設計系統網站
- 過度抽象的 component library

---

## Phase 2：Dashboard MVP

### 目標

讓 `dashboard` 先成為可穩定操作的管理後台。

### 優先頁面

- login / auth restore
- channels list
- channel stats
- channel config
- 基本 settings

### 核心工作

- token persistence / refresh flow
- route guard 與 `authProvider` 收斂
- role-based UI gating
- list / form / detail 頁面結構固定化
- 與後端 dashboard API contract 對齊

### 工程重點

- 單一 auth state source of truth
- 明確區分 `401`、`403`、`404`、`500`
- 建立表單成功 / 失敗 / 驗證錯誤模式
- 明確處理 streamer / agency / admin 差異

### 驗收標準

- 重整頁面後 auth 不會掉
- 主要 dashboard 頁面可完整走通
- 不同角色看不到不該看的操作
- 空資料與 forbidden 狀態都有清楚 UI

### 現階段不要做

- 複雜資料視覺化 dashboard
- 過度客製的 table infra
- 高複雜營運流程頁面

---

## Phase 3：Tachimint MVP

### 目標

把 `tachimint` 從「功能分散的 extension 原型」整理成穩定的 viewer app shell。

### 優先面板

- Home
- Missions
- Equipment
- Shop

### 核心工作

- 整理 hook ownership：
  - `useTwitch`
  - `useWatchSession`
  - `useHeartbeat`
  - `useBalance`
  - `useClickBoost`
  - `useBits`
- 明確 viewer / non-viewer 流程
- 修正 session start / heartbeat / end lifecycle
- 讓 balance、cooldown、bits 狀態來源單一化
- 建立直式 app shell 與 bottom nav

### UI 重點

- loading / auth failed / backend unavailable
- session starting
- balance ready / animating
- click ready / cooldown / gain / error
- bits idle / pending / success / error
- missions / equipment / shop 先用 placeholder，但結構正確

### 驗收標準

- viewer 流程能從 auth → session → heartbeat → balance → click 正常走通
- broadcaster / moderator / external 不會誤走 viewer 流程
- click 與 balance 不再各持一份不同步 state
- bits flow 至少有明確 pending / success / error UI

### 現階段不要做

- 真正的 BOSS 任務後端串接
- wallet connect / NFT claim flow
- 大量動畫 prototype

---

## Phase 4：Tachimint 視覺與互動升級

### 目標

在 MVP 穩定後，把 `tachimint` 提升成有辨識度、但仍像 Twitch Extension 的互動體驗。

### 核心工作

- 強化 Home 主舞台
- gain 動畫與 cooldown ring
- resource bar 視覺階層
- missions / equipment / shop 的一致視覺語言
- compact Twitch panel 響應壓縮規則

### 設計重點

- 直式 extension panel，而不是手機 app mockup
- 優先讓顏色、密度、語氣與元件感受貼近 Twitch 生態中合理存在的 extension
- 高資訊密度，但仍可快速掃描
- 有遊戲感，但只作為點綴，不做成獨立遊戲 launcher
- 不做成錢包 popup
- 不做成外部 SaaS dashboard

### 驗收標準

- Home 畫面具備明確主視覺焦點
- 所有互動關鍵狀態都有對應視覺反饋
- 360x720 與 318x500 都能閱讀與操作
- 第一眼仍應被理解為 Twitch 內的 extension，而不是獨立 app

### 現階段不要做

- 過度複雜的動效系統
- 先做完整角色養成與裝備 inventory

---

## Phase 5：進階能力接入

### 目標

把先前留為 placeholder 的功能，逐步接成真實產品能力。

### 可能範圍

- agency 專屬 dashboard 流程
- Angela Gate / BOSS 任務 API 串接
- equipment / buff / NFT teaser 到真實流程
- events / campaign / season UX

### 驗收標準

- placeholder 被真資料替換時，不需要重做整個畫面骨架
- 權限與資料流能沿用前面 phase 的既有模式

---

## 優先順序建議

### 建議順序

1. `Phase 1` 共用規則
2. `Phase 2` Dashboard MVP
3. `Phase 3` Tachimint MVP
4. `Phase 4` Tachimint 視覺升級
5. `Phase 5` 進階能力接入

### 為什麼這樣排

- `dashboard` 比較適合先把 auth、權限、資料契約打穩
- `tachimint` 的互動性高，若在 state ownership 混亂時就直接疊視覺，返工成本會更高
- 共用規則先建立後，後面兩條產品線的 review 成本都會下降

---

## 近期可執行清單

### 建議先做

1. 補一份 `dashboard` state inventory 文件
2. 補一份 `tachimint` state inventory 文件
3. 補一份 `tachimint` 視覺原則文件
4. 定義前端共用 design tokens 初版
5. 確認 `dashboard` 是否正式採 `Refine.dev`
6. 把 `tachimint` 六支 hook 的 ownership 文件化

### 建議暫緩

- 做完整高保真 Figma 大全套
- 做完整 NFT / wallet UX
- 提前擴張到大量營運頁面

---

## 成功標準

這份 roadmap 若執行順利，應該帶來以下結果：

- 前端任務能更清楚分辨是 `dashboard` 還是 `tachimint`
- PR review 有共同標準，不再每次重講一次 scope
- `dashboard` 與 `tachimint` 都能先把資料與權限邏輯走穩
- 視覺升級建立在穩定的 state 與 API contract 上，而不是反覆返工
