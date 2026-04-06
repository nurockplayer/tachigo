# Tachimint Implementation Checklist

## 目標

把目前 `tachimint` 設計討論文件，整理成可直接拆成 issue / phase 的實作清單。

這份文件不取代：

- [frontend-roadmap.md](/Users/tachikoma/Documents/Web3/tachigo/docs/frontend-roadmap.md)
- [target-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/target-state-inventory.md)
- [component-contract.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/component-contract.md)

而是把它們變成「接下來真的要做什麼」。

---

## 範圍原則

### 這一輪要做的

- 收斂 `Home / Mining` 的 state ownership
- 建立 `AppShell` 與四個 tab 的基本骨架
- 讓 `viewer / non-viewer` 流程明確
- 讓 auth / session / heartbeat / click / bits 都有對應 UI state
- 先把 `Missions / Equipment / Shop` 做成誠實 placeholder

### 這一輪先不要做的

- BOSS 任務真 API 串接
- 裝備系統真資料與 equip / unequip 流程
- 完整商城與複雜價格邏輯
- wallet / NFT / mint flow
- 大量高保真特效

---

## 建議拆分

## Phase A：Layout 與共用元件骨架

### A1. 建立 `AppShell`

目標：

- 把 `tachimint` 畫面從單一頁面結構拆成固定骨架

包含：

- `TopStatusBar`
- `StatusBannerHost`
- active panel slot
- `BottomTabBar`

完成標準：

- `App.tsx` 不再直接承載大段 layout JSX
- 四個 tab 有統一外框

### A2. 建立共用 UI primitives

目標：

- 先做四個 tab 都會重用的 UI 元件

建議元件：

- `PanelHeader`
- `FeatureCard`
- `StatusBadge`
- `PlaceholderCard`
- `FooterHint`

完成標準：

- `Home / Missions / Equipment / Shop` 不再各自發明卡片結構

### A3. 導入第一版 tokens

目標：

- 把 [design-tokens.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-tokens.md) 映射成實際 CSS variables 或 styling constants

完成標準：

- 至少色彩、字體、間距、圓角、狀態色有統一來源

---

## Phase B：Home 核心重構

### B1. 建立 `HomePanel`

目標：

- 把首頁拆成：
  - `MiningStage`
  - `ResourceRow`
  - `BitsStrip`

完成標準：

- 首頁主要區塊有清楚 component 邊界

### B2. 把 balance source of truth 收斂到 `useBalance`

目標：

- 不再讓 heartbeat / click 各自持有不同 balance state

完成標準：

- 主數字與 gain animation 都由同一支 hook 驅動
- click 後不再靠跨 hook 同步 patch 修 UI

### B3. 把 heartbeat 收斂成純同步 hook

目標：

- `useHeartbeat` 只負責 timer 與 refetch trigger

完成標準：

- `useHeartbeat` 不再直接持有 balance / gain / animation

### B4. 重構 click flow

目標：

- 讓 `MineAction` 只負責 render，真正的 cooldown / request / optimistic update 由 `useClickBoost` 控制

完成標準：

- 有明確 `ready / cooldown / pending / error` 狀態
- `retry_after_ms` 能正確驅動 UI

### B5. 把 bits 模組收成 `BitsStrip`

目標：

- bits 在首頁作為次要功能，不再散在主畫面各處

完成標準：

- 有 `idle / pending / success / error / unavailable` UI
- bits 狀態不會整頁 takeover

---

## Phase C：Auth / Role / Session 狀態收斂

### C1. 明確 viewer / non-viewer gate

目標：

- `broadcaster / moderator / external` 不再落進 viewer 主畫面

完成標準：

- 只有 `viewer` 會進入 `Home / Mining` 主流程
- 其他角色有獨立提示畫面

### C2. 把 `authError` 升級成真正 `authState`

目標：

- 不再只靠 header 紅點表達 auth 問題

建議狀態：

- `loading`
- `ready`
- `account_unlinked`
- `backend_unavailable`

完成標準：

- auth 問題有對應 banner / fallback

### C3. 把 `sessionStarting / sessionReady / sessionError` 顯性化

目標：

- start / end flow 不再是隱性行為

完成標準：

- `Mine` 是否可操作明確依賴 session state
- session error 有實際 UI

---

## Phase D：Secondary Panels Placeholder

### D1. 建立 `MissionsPanel`

目標：

- 先實作 preview / locked 版本

完成標準：

- featured mission + mission list + footer hint 到位
- 不誤導成真任務系統

### D2. 建立 `EquipmentPanel`

目標：

- 先實作 slot-based preview

完成標準：

- `LoadoutCard`
- `EquipmentSlot`
- `ItemPreviewCard`

都能正常顯示 placeholder 狀態

### D3. 建立 `ShopPanel`

目標：

- 先實作輕量 shop / boost panel

完成標準：

- 有 featured offer
- 有產品卡
- `preview / unavailable / available` 狀態可切換

---

## Phase E：視覺與互動收斂

### E1. 對齊 Twitch-native 視覺方向

目標：

- 確保畫面不像 launcher / wallet / SaaS dashboard

完成標準：

- `Home` 與三個 secondary panels 在同一套產品語言內

### E2. 補首頁必要動效

目標：

- 補足對狀態有幫助的最小動效

建議範圍：

- gain float
- cooldown ring
- CTA press feedback
- tab 切換過渡

完成標準：

- 動效幫助理解，不干擾閱讀

---

## 建議開票順序

### 第一批

1. `AppShell + BottomTabBar + TopStatusBar`
2. `HomePanel` 切分
3. `useBalance / useHeartbeat / useClickBoost` ownership 重構
4. `viewer / non-viewer gate`

### 第二批

5. `authState` 顯性化
6. `BitsStrip` 收斂
7. `MissionsPanel` placeholder
8. `EquipmentPanel` placeholder
9. `ShopPanel` placeholder

### 第三批

10. tokens 導入與視覺收斂
11. 最小動效補強

---

## 每張 Issue 建議模板

### 目標

- 這張票要解什麼 UI / state 問題

### 範圍

- 會改哪些 component / hook / page

### 不包含

- 明確列出這張票先不碰的功能

### 驗收標準

- 列出對應 state
- 列出是否需要 viewer / non-viewer / error flow 驗證

### 參考文件

- 直接鏈到對應的：
  - state inventory
  - wireframe contract
  - component contract

---

## 最小可上線切片

如果想用最小切片往前推，建議最先完成的是：

1. `AppShell`
2. `HomePanel`
3. `useBalance` 作為唯一 points source of truth
4. `MineAction` 正確 cooldown / pending / error
5. non-viewer gate
6. bits strip 的基本狀態

做到這裡，`tachimint` 就會從「原型頁面」進化成比較穩定的 extension UI 基礎。

---

## 驗收清單

1. 可以直接從這份文件拆出 5-10 張 issue
2. 每張 issue 都能找到對應設計文件
3. 順序上先解 state ownership，再補視覺細節
4. 不會在 placeholder 階段誤做過多後端未完成功能
