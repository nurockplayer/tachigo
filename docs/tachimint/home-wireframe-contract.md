# Tachimint Home Wireframe Contract

## 目標

定義 `tachimint` 首頁 `Home / Mining` 的 wireframe contract，讓後續：

- Figma wireframe
- UI implementation
- component 切分

都能依照同一套版面與 state 規則進行。

這份文件不追求高保真視覺細節，而是先固定：

- 區塊順序
- 視覺層級
- 各 state 進來時要替換哪個區塊
- 哪些內容必須永遠在首屏內可見

參考文件：

- [design-draft.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-draft.md)
- [design-tokens.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-tokens.md)
- [target-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/target-state-inventory.md)

---

## Wireframe 原則

### 1. 首屏必須看懂核心互動

使用者不應該需要捲動才知道：

- 目前是否可挖
- 現在有多少 points
- cooldown 是否存在
- session / auth 是否異常

### 2. 區塊替換優於整頁切換

除了 `context loading` 與 `non-viewer view` 之外，大部分 state 不應該整頁重畫，而是局部替換：

- header 狀態
- mining stage CTA
- banner
- bits strip

### 3. 主要互動永遠在中段

`Mine` 按鈕、主數字、capybara miner 這三個元素要構成明確中心，不應被 secondary card 或裝飾打散。

---

## 畫面骨架

### 固定骨架

```text
+--------------------------------------------------+
| Top Status Bar                                   |
+--------------------------------------------------+
| Optional Inline Banner                           |
+--------------------------------------------------+
| Main Mining Stage                                |
| - Capybara / main focus                          |
| - Primary points                                 |
| - CTA / cooldown                                 |
+--------------------------------------------------+
| Resource Row                                     |
| - Spendable | Cumulative | Heartbeat             |
+--------------------------------------------------+
| Bits Strip                                       |
+--------------------------------------------------+
| Bottom Navigation                                |
+--------------------------------------------------+
```

### 區塊優先順序

1. `Top Status Bar`
2. `Optional Inline Banner`
3. `Main Mining Stage`
4. `Resource Row`
5. `Bits Strip`
6. `Bottom Navigation`

---

## 區塊規格

## 1. Top Status Bar

### 目的

承載最小但必要的全局資訊：

- 當前面板標題
- channel / role context
- auth / session / bits 小狀態

### 內容

左側：

- `Tachimint`
- 次要小字：
  - channel name
  - 或 `Viewer mode`

右側：

- auth status dot
- bits status dot
- heartbeat / session small chip

### 高度建議

- `44px - 52px`

### 不應承載

- 大型標語
- 大 hero 圖
- 大量數字資訊

---

## 2. Optional Inline Banner

### 目的

承載重要但不該遮蔽首頁操作的狀態。

### 可出現的 state

- `auth degraded`
- `heartbeat error`
- `session starting`
- `session error`

### 規則

- 一次只顯示一條主 banner
- 優先顯示最影響操作的狀態

優先順序建議：

1. `session error`
2. `auth account unlinked`
3. `auth backend unavailable`
4. `heartbeat error`
5. `session starting`

### 畫面形式

- 橫向卡片
- icon + 1 行標題 + 1 行短說明
- 可有 secondary action，例如 `Retry`

---

## 3. Main Mining Stage

### 目的

作為首頁主焦點，集中呈現：

- 挖礦角色
- 主數字
- 主要 CTA
- cooldown / gain 回饋

### 區塊拆分

#### A. Focus Visual

內容：

- capybara miner
- 輕量背景紋理或氛圍層

規則：

- 角色圖不能壓過數字與 CTA
- 只作為辨識點，不做大型插畫展示

#### B. Primary Points

內容：

- 主數字：建議優先顯示 `spendable`
- 次要標示：`Points` 或 `Mining balance`
- 視需要補一行 `+gain`

規則：

- 數字是這一區最重要元素
- `balance loading` 時用 skeleton 或 placeholder，不能空白

#### C. Main CTA

按鈕文案建議：

- `Mine`

狀態替換：

- `click ready`：主 CTA 可點
- `click cooldown`：disabled + countdown
- `session starting`：disabled + `Preparing…`
- `click error`：恢復 CTA，並透過 banner 或 small text 提示

#### D. Cooldown / Gain Layer

表現方式：

- cooldown ring
- 數字倒數
- gain 浮字

規則：

- 這層是輔助層，不可遮住主數字
- gain 動畫停留短，不影響下一次讀取

### 高度建議

- `220px - 280px`

---

## 4. Resource Row

### 目的

把主舞台之外的核心數值拆出來，提升可掃描性。

### 建議欄位

卡片 1：

- `Spendable`

卡片 2：

- `Cumulative`

卡片 3：

- `Heartbeat`

### 規則

- 固定 3 欄或 2+1 欄
- 小卡內容只能放 1 個主值 + 1 個 label
- heartbeat 卡偏狀態型，不要塞太多細節

### 狀態對應

- `balance loading`：前兩張卡 skeleton
- `balance error`：前兩張卡顯示 error fallback
- `heartbeat running`：第三張卡顯示 `Synced`
- `heartbeat error`：第三張卡顯示 degraded 狀態

---

## 5. Bits Strip

### 目的

以低干擾方式呈現首頁內可見的 monetization / boost 模組。

### 結構

左側：

- 商品名稱
- 小型說明

右側：

- price / CTA
- pending / success / unavailable 狀態

### 規則

- 只顯示一個 featured product 或最簡化列表
- 不展開成完整商城
- `bits unavailable` 時保留區塊，但轉成 disabled / empty 狀態

### 狀態替換

- `bits idle`：可點商品
- `bits pending`：CTA disabled
- `bits success`：顯示成功提示，但不整頁 takeover
- `bits error`：顯示 inline error
- `bits unavailable`：顯示 `Not available in this channel`

---

## 6. Bottom Navigation

### 目的

在固定版型下提供主面板切換。

### 結構

四個 tab：

- `Home`
- `Missions`
- `Equipment`
- `Shop`

### 規則

- 固定在底部
- icon + 短 label
- active state 要比 hover state 更明確

### 不應做成

- 遊戲技能快捷欄
- 重型 fantasy 金屬按鈕列

---

## 主要 State 替換規則

## A. `context loading`

整頁替換為 loading state。

保留：

- 無

隱藏：

- 所有主 UI 區塊

## B. `non-viewer view`

整頁替換為 non-viewer 提示。

可保留：

- 簡化 header

隱藏：

- mining stage
- resource row
- bits strip
- bottom nav

## C. `auth loading`

保留骨架：

- header
- main mining stage
- resource row
- bits strip
- bottom nav

替換內容：

- banner 顯示 `Authorizing…`
- `Mine` disabled
- points 先不顯示真值

## D. `auth ready`

完整顯示正常 Home。

## E. `auth account unlinked`

保留：

- header
- banner
- main stage

替換：

- `Mine` disabled
- main stage 顯示引導文案
- bits strip 可視產品決定是否保留

## F. `auth backend unavailable`

保留：

- header
- banner
- main stage

替換：

- 主操作停用
- 明確顯示暫時不可用

## G. `session starting`

保留完整頁面。

替換：

- CTA disabled
- banner 或 CTA 附近顯示 `Preparing session…`

## H. `session ready`

完整正常狀態。

## I. `heartbeat error`

不整頁替換。

只改：

- banner 顯示 degraded
- resource row 的 heartbeat card 變成 error 變體

## J. `click cooldown`

只改 main stage：

- CTA disabled
- 顯示倒數
- 可持續顯示主數字

## K. `bits pending / success / error`

只改 bits strip。

不要影響：

- mining stage
- resource row
- bottom nav

---

## 首屏最低資訊要求

在 `360 x 720` 與 `318 x 500` 內，首屏至少要同時看見：

1. `Tachimint` / context header
2. 主 points 數字
3. `Mine` 按鈕
4. 至少一個狀態提示位置
5. bottom nav

若空間不足，優先壓縮：

- bits strip 高度
- decorative space
- resource row 文案長度

不要壓縮掉：

- 主數字
- CTA
- status 提示位

---

## Wireframe 註記規則

在 Figma / wireframe 上，建議每個區塊都標註：

- `real`
- `placeholder`
- `state-driven`
- `optional`

例如：

- `Top Status Bar`：`real`
- `Bits Strip`：`real`
- `Missions Tab Content`：`placeholder`
- `Auth Banner`：`state-driven`

---

## 驗收清單

1. `Mine`、主 points、capybara 三者形成明確中心
2. 重要 state 優先以區塊替換，而不是整頁重畫
3. `context loading` 與 `non-viewer view` 是唯二整頁替換情境
4. `bits` 狀態只影響 bits strip，不搶首頁控制權
5. 小尺寸下仍能一眼看懂是否可挖
6. wireframe 可直接交給設計或前端做下一步，不需要再重講版型規則

---

## 後續建議

1. 依這份 contract 畫 `Home / Mining` 低保真 wireframe
2. 再補 `Missions / Equipment / Shop` 的 wireframe contract
3. 若開始實作，將各區塊直接映射成 component tree
