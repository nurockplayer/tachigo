# Tachimint Secondary Panels Wireframe Contract

## 目標

定義 `tachimint` 次要三個主 tab 的 wireframe contract：

- `Missions`
- `Equipment`
- `Shop`

這份文件的目的，是讓這三個 panel 在 API 尚未完整接好前，也能先有：

- 一致的版型
- 誠實的 placeholder
- 可延續到正式功能的結構

參考文件：

- [design-draft.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-draft.md)
- [design-tokens.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-tokens.md)
- [target-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/target-state-inventory.md)
- [home-wireframe-contract.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/home-wireframe-contract.md)

---

## 共用原則

### 1. 次要 panel 不能比 Home 更重

`Home / Mining` 永遠是主舞台。

所以 `Missions / Equipment / Shop` 應該：

- 清楚
- 可讀
- 有結構

但不應：

- 比 Home 更像完整遊戲介面
- 用更重的裝飾搶主角

### 2. Placeholder 要誠實

在真 API 尚未接好前，這三個 panel 可以存在，但必須明確標示：

- `Preview`
- `Planned`
- `Locked`
- `Not connected`

不要假裝成：

- 可真的領取獎勵
- 可真的裝備
- 可真的購買完整商城內容

### 3. Panel 版型要共用

三個 panel 應沿用同一組結構語言：

1. panel header
2. featured block
3. list / slots / products
4. optional footer note

---

## 共用畫面骨架

```text
+--------------------------------------------------+
| Top Status Bar                                   |
+--------------------------------------------------+
| Panel Header                                     |
| - Title                                          |
| - Short status / helper text                     |
+--------------------------------------------------+
| Featured Block                                   |
+--------------------------------------------------+
| Main Content List / Slots / Products             |
+--------------------------------------------------+
| Optional Footer Note / Placeholder Hint          |
+--------------------------------------------------+
| Bottom Navigation                                |
+--------------------------------------------------+
```

### 與 Home 的差異

- 不需要 `Optional Inline Banner` 常駐位
- 不需要大型主 CTA 置中
- 內容可偏列表型

但仍要保留：

- top status continuity
- bottom nav continuity
- 一致的 panel padding / card 語言

---

## Missions Panel

## 目標

提供任務與進度的概念入口，但在後端尚未完成前，不把它畫成完整 quest system。

## 區塊規格

### 1. Panel Header

內容：

- `Missions`
- 短 helper text，例如：
  - `Watch and progress`
  - `Mission preview`

### 2. Featured Mission Block

內容：

- 主要 mission title
- 簡短說明
- progress area
- reward preview

規則：

- 只放 1 張 featured mission
- 若尚未接 API，progress 要明示為 preview

### 3. Mission List

建議 2~4 張卡：

- `Active`
- `Upcoming`
- `Locked`

每張卡可包含：

- title
- short description
- status badge
- optional progress line

### 4. Footer Note

可顯示：

- `More mission types coming soon`

## 狀態替換

### `missions preview only`

- Featured block 顯示 preview 標示
- list 中至少一張卡標成 `Locked`

### `missions unavailable`

- panel 不消失
- 轉成 placeholder card + 說明

### `missions real`

- progress 條、reward、status badge 才能改為真資料語氣

## 不應做成

- RPG 任務日誌
- 大量敘事文本
- 滿版世界觀說明

---

## Equipment Panel

## 目標

建立裝備概念與 slot-based 結構，但先不要把它畫成完整 inventory game。

## 區塊規格

### 1. Panel Header

內容：

- `Equipment`
- helper text，例如：
  - `Upgrade preview`
  - `Loadout preview`

### 2. Miner Loadout Block

內容：

- capybara / miner mini portrait
- 2~4 個主要裝備 slot

slot 建議：

- `Tool`
- `Charm`
- `Boost`
- `Skin`

### 3. Item Preview List

每張卡可包含：

- item name
- rarity / type badge
- short effect text
- `Preview` 或 `Locked`

### 4. Footer Note

可顯示：

- `Equipment system not connected yet`

## 狀態替換

### `equipment preview only`

- slot 可見
- item card 可見
- 所有 action 皆 disabled

### `equipment locked`

- slot 顯示 lock icon
- item card 顯示 unlock hint

### `equipment real`

- 才允許顯示 equip / unequip action

## 不應做成

- Diablo 背包
- MMORPG 紙娃娃
- 滿格 inventory grid

---

## Shop Panel

## 目標

提供商城與 boost 的統一入口，但保持 Twitch extension 的輕量交易感。

## 區塊規格

### 1. Panel Header

內容：

- `Shop`
- helper text，例如：
  - `Boost and upgrades`
  - `Support with Bits`

### 2. Featured Offer Block

內容：

- 主要 boost / featured item
- 1 行說明
- price / availability

規則：

- 這張卡比 list 稍大
- 但不能做成滿版商城首頁

### 3. Product List

建議 2~4 張產品卡：

- Bits-supported item
- planned boost
- locked future item

每張卡可包含：

- item name
- item type
- short effect
- state badge
- CTA / disabled CTA

### 4. Footer Note

可顯示：

- `Some items are preview only`

## 狀態替換

### `shop available`

- featured offer 與至少一張商品卡可互動

### `bits unavailable`

- 保留 panel
- item 改成 unavailable 樣式
- 說明當前頻道無法使用 Bits

### `shop preview only`

- CTA 全部 disabled
- badge 清楚寫 `Preview`

## 不應做成

- NFT marketplace
- 財務產品頁
- 複雜價格表與比較表

---

## 共用元件建議

這三個 panel 可共用：

- `PanelHeader`
- `FeatureCard`
- `StatusBadge`
- `PlaceholderCard`
- `LockStateCard`
- `MiniProgress`
- `ItemPreviewCard`
- `FooterHint`

---

## 首屏資訊要求

在較小尺寸下，這三個 panel 的首屏至少要看見：

1. panel title
2. 1 個 featured block
3. 至少 1 個 list item / slot 區
4. bottom nav

若空間不足，優先壓縮：

- helper text 長度
- item 說明文字
- footer note

不要優先壓縮：

- panel title
- featured block
- current tab indication

---

## Wireframe 註記規則

在 Figma / wireframe 上，建議每個區塊標記：

- `real`
- `preview`
- `locked`
- `not connected`

例如：

- featured mission：`preview`
- equipment slots：`preview`
- bits offer：`real` 或 `preview`

---

## 驗收清單

1. 三個 panel 看起來屬於同一個產品
2. 結構與 Home 一致，但不搶主舞台
3. placeholder 語意清楚，不誤導為已完成功能
4. `Missions` 偏進度與任務
5. `Equipment` 偏 slot 與 upgrade 概念
6. `Shop` 偏 boost / item 入口，而不是大型商城

---

## 後續建議

1. 若要繼續深化，可分別補：
   - `missions-state-inventory.md`
   - `equipment-state-inventory.md`
   - `shop-state-inventory.md`
2. 或直接依這份 contract 畫三個 panel 的低保真 wireframe
