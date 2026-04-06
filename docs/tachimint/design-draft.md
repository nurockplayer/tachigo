# Tachimint Design Draft

## 目標

提供 `tachimint` 第一版前端設計草案，作為後續：

- Figma wireframe
- component 設計
- 前端實作切版

的共同基準。

這份草案不是最終高保真設計稿，而是偏向：

- 結構先行
- 視覺方向明確
- state 可落地
- placeholder 誠實

---

## 設計前提

這份草案建立在以下原則上：

- `Twitch-native first`
- `GameFi flavor second`
- 小面積 panel 可讀性優先
- 真實 API 與 placeholder 必須分清楚

參考文件：

- [visual-principles.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/visual-principles.md)
- [home-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/home-state-inventory.md)
- [target-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/target-state-inventory.md)

---

## 版型總覽

### Form Factor

目標尺寸優先考慮：

- `360 x 720`
- `318 x 500`

整體應是一個直式 Twitch Extension panel，而不是手機 app mockup。

### 畫面骨架

建議固定為四段：

1. Top status bar
2. Home / panel content
3. Secondary utility strip
4. Bottom navigation

說明：

- Top status bar 負責身份、狀態、資源摘要
- Home / panel content 放主要互動
- Secondary utility strip 放補充資訊或輕量 CTA
- Bottom nav 切換主面板

---

## 首頁 Home / Mining

### 目標

使用者一進來必須在幾秒內看懂：

- 現在是不是 viewer
- auth / session 是否正常
- 可不可以挖
- 現在有多少 points
- 是否有 bits boost 可以買

### 結構草案

#### 1. Header

內容：

- `Tachimint` 小標
- 頻道或目前所在 context 摘要
- 右上角 status cluster

status cluster 可包含：

- auth degraded dot
- bits availability dot
- session / heartbeat 小狀態

風格：

- 高度低
- 緊湊
- 不做大型 hero

#### 2. Main Mining Stage

內容：

- capybara miner 主視覺
- 主 points 數字
- 次要 points 說明
- 主 CTA：`Mine`
- cooldown / gain feedback

設計要點：

- capybara 是辨識點，但不是插畫展板
- points 數字要比角色更容易掃到
- CTA 永遠在首屏中央區域

#### 3. Resource Row

建議在主舞台下方放一列 2~3 個小卡：

- `Spendable`
- `Cumulative`
- `Heartbeat`

用途：

- 把核心數值拆出來
- 避免所有資訊都壓在主舞台上

#### 4. Bits Strip

形式：

- 單列卡片或橫向小模組

內容：

- 最主要的 bits boost 商品
- pending / unavailable / success 狀態

目的：

- 讓 bits 是「可見的次要功能」
- 但不搶首頁主要焦點

#### 5. Bottom Nav

固定四個 tab：

- `Home`
- `Missions`
- `Equipment`
- `Shop`

風格：

- 簡潔 icon + label
- active tab 清楚
- 不做過重遊戲快捷欄

---

## Home 狀態草圖

### A. Context Loading

畫面：

- 中央 loading
- 簡短文案：`Connecting to Twitch…`

不要：

- 顯示完整主 UI skeleton 太久

### B. Viewer Ready

畫面：

- 完整 Home 結構
- `Mine` 可操作
- points 與 bits 區塊可見

### C. Non-viewer View

適用：

- broadcaster
- moderator
- external

畫面：

- 簡潔提示卡
- 說明此 extension 主要提供 viewer 使用

不要：

- 落回 viewer 主畫面

### D. Auth Degraded

畫面：

- Header 有 degraded indicator
- Home 內有一塊明確 banner / inline notice

原則：

- 不再只靠紅點
- 要讓使用者知道是暫時性問題還是未綁定問題

### E. Session Starting

畫面：

- `Mine` disabled
- 顯示 `Preparing session…`

### F. Click Cooldown

畫面：

- `Mine` 按鈕 disabled
- 明確顯示秒數
- 可用 ring 或文字倒數

### G. Heartbeat Error

畫面：

- Home 保持可讀
- 顯示一條 degraded banner 或 small card

不要：

- 完全無提示

### H. Bits Pending / Success / Error

bits strip 應有 3 種狀態切換：

- `pending`
- `success`
- `error`

且不應把整個首頁切成交易完成畫面。

---

## Missions Panel 草案

### 目標

在 API 尚未完全接好前，先讓這個 panel 有清楚結構與誠實 placeholder。

### 結構

1. panel title
2. active mission card
3. upcoming / locked mission list

### 視覺方向

- 比 Home 更資訊型
- 仍保留遊戲化卡片語言
- 但不要變成完整 quest journal

### Placeholder 規則

若尚未接真 API：

- 標示 `Planned`
- 標示 `Locked`
- 標示 `Coming soon`

不要假裝 countdown、reward、progress 都是真的。

---

## Equipment Panel 草案

### 目標

先建立「裝備感」而不是先做完整 inventory 系統。

### 結構

1. character / miner mini view
2. equipment slots
3. item preview cards

### 視覺方向

- 可借用 RPG slot 語法
- 但要壓低厚重 fantasy 味

比較接近：

- 輕量 avatar upgrade panel

而不是：

- Diablo 背包
- MMO 紙娃娃系統

### Placeholder 規則

- 直接標 `Not connected`
- 或 `Equipment preview`

---

## Shop Panel 草案

### 目標

讓商城是清楚、可信的補充功能，不變成首頁第二主角。

### 結構

1. featured boost
2. product list
3. unavailable / coming soon items

### 視覺方向

- 比 missions / equipment 更接近 Twitch extension 交易模組
- 保持乾淨、可掃描

### 不要做成

- Web3 NFT marketplace
- 金融商品列表
- 重型 fantasy 商人 UI

---

## 元件清單

這份草案建議至少先定以下元件：

- `TopStatusBar`
- `StatusDot`
- `StatusBanner`
- `MiningStage`
- `PointsDisplay`
- `MineButton`
- `CooldownRing`
- `GainFloat`
- `ResourceMiniCard`
- `BitsStrip`
- `BitsProductCard`
- `PanelHeader`
- `PlaceholderCard`
- `BottomTabBar`

---

## 視覺層級

### 首要層級

- main points
- main CTA
- session / cooldown 狀態

### 次要層級

- cumulative / spendable breakdown
- bits boost
- panel nav

### 第三級

- 裝飾性背景
- 額外小圖示
- flavor 文案

原則：

- 遊戲 flavor 永遠不能蓋過首要層級

---

## 文案草案

### 可直接拿來試的短文案

- `Mine`
- `Cooldown`
- `Preparing session`
- `Bits boost`
- `Mission preview`
- `Equipment preview`
- `Shop preview`
- `Viewer only`
- `Temporarily unavailable`

### 文案原則

- 儘量短
- 先可理解，再談 flavor
- extension 內避免長段敘事

---

## 設計驗收清單

1. 首頁一屏內看得懂核心互動
2. 非 viewer 不會誤進主挖礦畫面
3. auth / heartbeat / cooldown 狀態有可見 UI
4. missions / equipment / shop 的 placeholder 不會誤導
5. 整體像 Twitch extension，不像獨立產品
6. 遊戲感存在，但沒有壓過資訊可讀性

---

## 後續建議

1. 補一份 `design-tokens.md`
2. 依這份草案畫 `Home / Mining` wireframe
3. 再做一次 `Home` 的 component contract 拆解
