# Tachimint Visual Principles

## 目標

定義 `tachimint` 的前端視覺方向，讓後續 Figma、元件設計與實作都能沿用同一組原則。

這份文件回答的不是：

- API 怎麼串
- state 怎麼切
- hook 怎麼拆

而是回答：

- `tachimint` 應該看起來像什麼
- 不應該看起來像什麼
- 在 Twitch Extension 的限制下，哪些視覺決策是優先原則

這份文件承接：

- [frontend-roadmap.md](/Users/tachikoma/Documents/Web3/tachigo/docs/frontend-roadmap.md)
- [home-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/home-state-inventory.md)
- [target-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/target-state-inventory.md)

---

## 核心原則

### 1. Twitch-native first

`tachimint` 必須讓使用者第一眼就覺得：

- 這是 Twitch 內合理存在的 extension
- 不是外部網站嵌進來
- 不是另一個獨立產品

因此整體 form factor、資訊密度、語氣與互動方式，都要優先貼近 Twitch 生態。

### 2. GameFi flavor second

`tachimint` 可以有遊戲感，但遊戲感只能是加味，不應主導整體結構。

可以保留：

- capybara miner 主視覺
- gain feedback
- missions / equipment / shop
- 輕量 fantasy / mining 氛圍

但不應讓畫面看起來像：

- RPG 背包 UI
- MMORPG HUD
- 獨立遊戲 launcher

### 3. Small-panel readability is a feature

Twitch Extension 不是大畫面 dashboard，也不是手機 app。

設計時要先假設：

- 空間窄
- 高度有限
- 使用者注意力短
- 需要在幾秒內看懂目前狀態

因此可讀性本身是第一級需求，不是最後才補的細節。

---

## 不應偏移成

### 不要做成 Twitch 官方介面的複製品

目標是「相容」而不是「仿製」。

不需要硬拷貝 Twitch 官方元件，但要保留：

- 深色基底
- 清楚層級
- 合理的紫色使用
- extension 內可接受的資訊密度

### 不要做成獨立遊戲 launcher

避免：

- 過重的世界觀背景
- 充滿特效的首頁
- 大型金屬框、厚重裝飾邊框、滿版 fantasy 材質
- 讓任務 / 裝備 / 商城像另一個完整遊戲系統

### 不要做成錢包 popup

避免：

- 過度像 MetaMask 或 wallet approval modal 的卡片結構
- 強烈金融產品語氣
- 以資產列表與交易狀態作為主要視覺節奏

### 不要做成 SaaS dashboard

避免：

- 純後台式 stat cards 排列
- 太白、太淺、太乾的資料介面
- 一眼看起來像 admin panel

---

## 視覺語氣

### 情緒

應該是：

- 神秘但清楚
- 遊戲化但不幼稚
- Twitch 原生感強
- 節奏感明確

不應該是：

- 中世紀 cosplay
- 科幻戰艦控制台
- Web3 錢包面板

### 文案語氣

應偏向：

- 短句
- 明確
- 行動導向
- 像 extension 提示，而不是 RPG 敘事文本

例如：

- `Mining`
- `Cooldown`
- `Watch to earn`
- `Bits boost`

而不是過長的世界觀敘述。

---

## 色彩方向

### 基底

建議以深色為主，但不要做成純黑。

方向應偏向：

- charcoal
- deep violet
- muted navy

目的：

- 與 Twitch 深色環境相容
- 讓紫色與高亮狀態色有空間發揮

### 強調色

優先順序：

1. Twitch-compatible purple
2. 少量 mint / cyan 作為系統或成功狀態
3. 少量 gold / amber 作為成長或獎勵點綴

注意：

- 紫色應該是系統語言的一部分，不要整片發光紫
- 金色只能點綴，不要整套 dark fantasy 金邊卡片

### 狀態色

至少需要清楚區分：

- success
- pending
- error
- unavailable
- cooldown

這些狀態色要優先服務可讀性，而不是追求華麗。

---

## 排版與密度

### 版型

首頁應該是一個直式 extension panel，不是手機 mockup。

建議結構：

1. top status / header
2. primary mining stage
3. secondary status blocks
4. bottom nav

### 密度

原則：

- 資訊密度可以高
- 但視線路徑必須短
- 一屏內要先看懂「能不能挖、現在多少、下一步做什麼」

避免：

- 首頁同時塞太多裝飾與數值
- secondary feature 和 primary CTA 搶焦點

### 字體

優先使用易讀的無襯線字體，不走重 fantasy display font。

可以在標題或數字局部使用較有辨識度的字重或字形，但不要讓整個 extension 變成海報。

---

## 元件方向

### Header

應該承載：

- 頻道 / viewer context
- auth / degraded 狀態
- 主要資源摘要

應該是緊湊、實用的，而不是大型 hero header。

### Mining Stage

首頁主舞台要有角色或主視覺焦點，但必須服務操作。

應該包含：

- capybara miner
- 目前挖礦主狀態
- 主要 CTA
- gain / cooldown feedback

不應該包含：

- 過重背景劇情
- 需要捲動才看得到的主操作

### Stat / Resource Cards

應該偏小、明確、可掃描。

用途：

- 顯示 spendable / cumulative
- 顯示 heartbeat / session 狀態
- 顯示 bits / boost 摘要

設計上應該像 extension info module，不要像 dashboard analytics card。

### Bottom Navigation

固定底部 tab bar 是合理方向，但必須保持：

- 可單手理解
- icon + 短標籤
- 當前 tab 強調清楚
- 不要過度擬真遊戲快捷欄

### Placeholder Panels

`Missions`、`Equipment`、`Shop` 在 API 尚未完成前，可以有 teaser / locked / placeholder。

但樣式要誠實表達：

- 已規劃
- 尚未開放
- 或功能預覽

不要假裝成已可操作的完整系統。

---

## 動效方向

### 可以有的動效

- gain 浮字
- cooldown 環
- 按鈕按下回饋
- 資源數字 bump
- tab 切換的輕微過渡

### 不應過度

避免：

- 滿版粒子
- 長時間發光
- 影響資訊辨識的背景動畫
- 每個元件都各自晃動

原則是：

- 動效幫助理解狀態
- 不是為了炫技

---

## Figma 實作準則

### 應先畫的東西

1. `Home / Mining`
2. `non-viewer view`
3. `auth degraded`
4. `session starting`
5. `click cooldown`
6. `bits pending / success / error`
7. `Missions / Equipment / Shop` placeholder

### 每個畫面都要對齊

- state inventory
- source of truth
- 真實 API 可接程度
- 是否為 placeholder

### 在 Figma 上要直接標記

- `real`
- `placeholder`
- `locked`
- `not connected yet`

避免設計稿看起來像所有功能都已完成。

---

## 驗收清單

1. 第一眼應該像 Twitch 內的 extension，不像獨立 app
2. viewer 能在數秒內看懂：
   - 目前是否可挖
   - 目前點數狀態
   - 是否在 cooldown
   - 是否有 bits 可買
3. `capybara miner` 可以成為辨識點，但不能壓過可讀性
4. `Missions / Equipment / Shop` 就算是 placeholder，也不會誤導成已可完整使用
5. 視覺風格不會滑向：
   - 遊戲 launcher
   - 錢包 popup
   - SaaS dashboard

---

## 後續建議

1. 補一份 `tachimint` design tokens 草案
2. 補一份 `Home / Mining` wireframe contract
3. 把 `Missions / Equipment / Shop` 的 placeholder contract 明文化
