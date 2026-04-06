# Tachimint Component Contract

## 目標

定義 `tachimint` 第一版前端的 component tree 與 component contract，作為後續：

- React 切版
- state 下放 / props 設計
- UI 重構

的基準文件。

這份文件承接：

- [home-wireframe-contract.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/home-wireframe-contract.md)
- [secondary-panels-wireframe-contract.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/secondary-panels-wireframe-contract.md)
- [target-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/target-state-inventory.md)
- [design-tokens.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-tokens.md)

---

## 原則

### 1. Layout component 不應擁有業務 state

像：

- `AppShell`
- `TopStatusBar`
- `BottomTabBar`
- `PanelHeader`

這類元件應該只接收 props，不自己發 request、不自己推導 auth / heartbeat / balance。

### 2. State 應由 hook 擁有，component 只做表達

目標對齊：

- `useTwitch`
- `useWatchSession`
- `useHeartbeat`
- `useBalance`
- `useClickBoost`
- `useBits`

component 只負責：

- render
- event callback
- variant 切換

### 3. Home 與 Secondary Panels 共用基礎元件

不要每個 tab 都各自重新發明：

- header
- status banner
- card
- badge
- placeholder
- bottom nav

---

## 目標 Component Tree

```text
TachimintApp
└─ AppShell
   ├─ TopStatusBar
   ├─ StatusBannerHost
   ├─ ActivePanel
   │  ├─ HomePanel
   │  │  ├─ MiningStage
   │  │  │  ├─ MiningVisual
   │  │  │  ├─ PointsDisplay
   │  │  │  ├─ MineAction
   │  │  │  └─ GainLayer
   │  │  ├─ ResourceRow
   │  │  │  ├─ ResourceCard
   │  │  │  ├─ ResourceCard
   │  │  │  └─ HeartbeatCard
   │  │  └─ BitsStrip
   │  │     └─ BitsProductCard
   │  ├─ MissionsPanel
   │  │  ├─ PanelHeader
   │  │  ├─ FeatureCard
   │  │  ├─ MissionCard[]
   │  │  └─ FooterHint
   │  ├─ EquipmentPanel
   │  │  ├─ PanelHeader
   │  │  ├─ LoadoutCard
   │  │  ├─ EquipmentSlot[]
   │  │  ├─ ItemPreviewCard[]
   │  │  └─ FooterHint
   │  └─ ShopPanel
   │     ├─ PanelHeader
   │     ├─ FeatureCard
   │     ├─ ShopItemCard[]
   │     └─ FooterHint
   └─ BottomTabBar
```

---

## Root Layer

## `TachimintApp`

### 責任

- 組合各支 hook
- 根據 `role / auth / activeTab` 決定 render 路徑
- 把 state 映射成 UI props

### 不應負責

- 直接寫大量 JSX 結構
- 在 component tree 內直接散落 API 呼叫

### 建議輸出給 `AppShell`

- `statusBarProps`
- `bannerProps`
- `activeTab`
- `panelProps`
- `tabBarProps`

---

## `AppShell`

### 責任

- 提供固定 layout 骨架
- 排出：
  - header
  - banner slot
  - active panel
  - bottom nav

### Props 建議

- `header`
- `banner`
- `children`
- `tabBar`

### 不應負責

- 自己判斷 auth state
- 自己判斷哪個 panel 顯示什麼

---

## Home Panel Components

## `HomePanel`

### 責任

組合首頁各區塊：

- `MiningStage`
- `ResourceRow`
- `BitsStrip`

### Props 建議

- `miningStage`
- `resources`
- `bits`

### 不應負責

- 自己發 heartbeat
- 自己管理 click cooldown

---

## `MiningStage`

### 責任

承載首頁主焦點：

- capybara miner
- 主數字
- CTA
- cooldown / gain layer

### 子元件

- `MiningVisual`
- `PointsDisplay`
- `MineAction`
- `GainLayer`

### Props 建議

- `primaryValue`
- `secondaryLabel`
- `cta`
- `cooldown`
- `gain`
- `visualVariant`

### 狀態變體

- `loading`
- `ready`
- `cooldown`
- `disabled`
- `error-hint`

---

## `MiningVisual`

### 責任

- 顯示 capybara miner
- 顯示輕量背景氛圍

### 不應負責

- 顯示主數字
- 直接顯示 request 狀態文案

---

## `PointsDisplay`

### 責任

- 顯示主數字
- 顯示 label / subtext
- 支援 skeleton / error / animating

### Props 建議

- `value`
- `label`
- `subtext`
- `state`

### 狀態變體

- `loading`
- `ready`
- `animating`
- `error`

---

## `MineAction`

### 責任

- 顯示主 CTA
- 顯示 cooldown / pending / disabled 文案
- 接收 `onPress`

### Props 建議

- `state`
- `cooldownMs`
- `disabledReason`
- `onPress`

### 狀態變體

- `ready`
- `cooldown`
- `pending`
- `disabled`

### 不應負責

- 計算 cooldown
- 發 click request

---

## `GainLayer`

### 責任

- 顯示 gain 浮字
- 顯示 cooldown ring
- 顯示短期 bump / glow

### Props 建議

- `gain`
- `isAnimating`
- `cooldownMs`

### 原則

- 永遠是覆蓋層
- 不應成為主要內容承載者

---

## `ResourceRow`

### 責任

- 排列資源與狀態小卡

### 子元件

- `ResourceCard`
- `HeartbeatCard`

### Props 建議

- `items`

---

## `ResourceCard`

### 用途

顯示：

- `Spendable`
- `Cumulative`

### Props 建議

- `label`
- `value`
- `state`
- `trend`

### 狀態變體

- `loading`
- `ready`
- `error`

---

## `HeartbeatCard`

### 用途

顯示 heartbeat / session 的摘要狀態。

### Props 建議

- `state`
- `caption`

### 狀態變體

- `running`
- `error`
- `idle`

---

## `BitsStrip`

### 責任

- 顯示首頁 bits 模組
- 保持它是次要功能，不主導頁面

### Props 建議

- `state`
- `products`
- `featuredProduct`
- `onBuy`

### 狀態變體

- `unavailable`
- `idle`
- `pending`
- `success`
- `error`

---

## `BitsProductCard`

### 責任

- 顯示單一 bits 商品
- 處理 label / price / CTA 的 render

### 不應負責

- 直接發 purchase request

---

## Shared Secondary Panel Components

## `PanelHeader`

### 責任

- 顯示 panel title
- 顯示 helper text
- 保持各 tab header 一致

### Props 建議

- `title`
- `subtitle`
- `statusHint`

---

## `FeatureCard`

### 用途

承載每個 panel 最上方的重點區塊：

- featured mission
- loadout block
- featured offer

### Props 建議

- `title`
- `description`
- `meta`
- `state`
- `children`

---

## `FooterHint`

### 用途

- 補充 placeholder / future note

### Props 建議

- `text`
- `tone`

---

## Missions Components

## `MissionsPanel`

### 責任

- 組合任務頁的 header、featured mission、mission list、footer

### 子元件

- `PanelHeader`
- `FeatureCard`
- `MissionCard`
- `FooterHint`

### Props 建議

- `featuredMission`
- `missions`
- `state`

---

## `MissionCard`

### Props 建議

- `title`
- `description`
- `status`
- `progress`
- `reward`
- `locked`

### 狀態變體

- `preview`
- `locked`
- `active`
- `upcoming`

---

## Equipment Components

## `EquipmentPanel`

### 子元件

- `PanelHeader`
- `LoadoutCard`
- `EquipmentSlot`
- `ItemPreviewCard`
- `FooterHint`

### Props 建議

- `loadout`
- `slots`
- `items`
- `state`

---

## `LoadoutCard`

### 責任

- 顯示 capybara / miner mini view
- 顯示 slot summary

### Props 建議

- `visual`
- `slots`
- `state`

---

## `EquipmentSlot`

### Props 建議

- `slotName`
- `filled`
- `locked`
- `itemName`

---

## `ItemPreviewCard`

### Props 建議

- `name`
- `type`
- `rarity`
- `effectText`
- `state`

### 狀態變體

- `preview`
- `locked`
- `equipped`

---

## Shop Components

## `ShopPanel`

### 子元件

- `PanelHeader`
- `FeatureCard`
- `ShopItemCard`
- `FooterHint`

### Props 建議

- `featuredOffer`
- `items`
- `state`

---

## `ShopItemCard`

### Props 建議

- `name`
- `type`
- `description`
- `priceLabel`
- `state`
- `onPress`

### 狀態變體

- `available`
- `pending`
- `preview`
- `unavailable`
- `locked`

---

## Global Components

## `TopStatusBar`

### 責任

- 顯示當前 tab 的全局狀態摘要

### Props 建議

- `title`
- `contextLabel`
- `statusDots`
- `rightChip`

---

## `StatusBannerHost`

### 責任

- 接收優先順序最高的一條 banner
- 確保只渲染一條主 banner

### Props 建議

- `banner`

---

## `BottomTabBar`

### 責任

- 顯示四個主 tab
- 處理 active tab 切換

### Props 建議

- `tabs`
- `activeTab`
- `onTabChange`

---

## Component 與 State Ownership 對應

### `useTwitch`

主要餵給：

- `TopStatusBar`
- `StatusBannerHost`
- `BitsStrip`
- role gate / non-viewer branch

### `useWatchSession`

主要餵給：

- `StatusBannerHost`
- `MineAction`
- `HeartbeatCard`

### `useHeartbeat`

主要餵給：

- `HeartbeatCard`
- `StatusBannerHost`

### `useBalance`

主要餵給：

- `PointsDisplay`
- `ResourceCard`
- `GainLayer`

### `useClickBoost`

主要餵給：

- `MineAction`
- `GainLayer`

### `useBits`

主要餵給：

- `BitsStrip`
- `BitsProductCard`

---

## 第一階段實作切分建議

### Step 1

先做 layout 與 shared UI：

- `AppShell`
- `TopStatusBar`
- `StatusBannerHost`
- `BottomTabBar`
- `PanelHeader`
- `FeatureCard`
- `FooterHint`

### Step 2

做 Home 核心：

- `HomePanel`
- `MiningStage`
- `PointsDisplay`
- `MineAction`
- `ResourceRow`
- `BitsStrip`

### Step 3

做 secondary panels 的 placeholder 版：

- `MissionsPanel`
- `EquipmentPanel`
- `ShopPanel`

---

## 驗收清單

1. `App.tsx` 不再直接承載大段首頁 JSX
2. 每個 component 的責任可以用一句話說清楚
3. state 來源與 component 顯示關係一致
4. shared UI 不會在四個 tab 各複製一份
5. component tree 足以支撐第一階段實作，不需要邊做邊重想結構
