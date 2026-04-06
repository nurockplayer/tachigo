# Tachimint Design Tokens

## 目標

定義 `tachimint` 第一版設計 tokens，作為：

- Figma styles
- CSS variables
- React component styling

的共同基準。

這份文件的目標不是一次做出完整 design system，而是先固定最重要的視覺骨架，避免後續：

- 每個畫面自己挑色
- 間距與圓角風格飄移
- 狀態色不一致
- extension panel 在不同頁面像不同產品

參考文件：

- [visual-principles.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/visual-principles.md)
- [design-draft.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/design-draft.md)

---

## Token 使用原則

### 1. 先服務可讀性

所有 token 都要先服務：

- 小面積可讀性
- 狀態辨識
- Twitch 深色環境相容性

### 2. 紫色是系統語言，不是裝飾洪水

`tachimint` 可以使用 Twitch-compatible purple，但應作為：

- active
- focus
- selection
- key CTA

而不是整片背景都變亮紫。

### 3. 金色只作為獎勵點綴

gold / amber 只能用在：

- gain
- reward
- highlight accent

不要把所有卡片都做成金邊 fantasy 卡。

---

## 色彩 Tokens

### Core Surface

```text
--tm-color-bg-app:        #0E0E13
--tm-color-bg-panel:      #15151D
--tm-color-bg-elevated:   #1D1D2A
--tm-color-bg-muted:      #242437
--tm-color-bg-overlay:    rgba(8, 8, 12, 0.72)
```

用途：

- `bg-app`: 最外層背景
- `bg-panel`: 主 panel / card
- `bg-elevated`: hover / active surface
- `bg-muted`: 次要區塊、placeholder

### Border / Divider

```text
--tm-color-border-soft:   rgba(255, 255, 255, 0.08)
--tm-color-border-strong: rgba(255, 255, 255, 0.16)
--tm-color-divider:       rgba(255, 255, 255, 0.06)
```

原則：

- 優先用透明白做邊界
- 不用太亮的實線框

### Text

```text
--tm-color-text-primary:   #F6F7FB
--tm-color-text-secondary: #B7B8C9
--tm-color-text-muted:     #8A8CA4
--tm-color-text-disabled:  #66687D
--tm-color-text-inverse:   #0E0E13
```

用途：

- `primary`: 主數字、標題、CTA 文字
- `secondary`: 說明、次要數值
- `muted`: placeholder / 備註

### Brand / Accent

```text
--tm-color-brand-500:     #9147FF
--tm-color-brand-400:     #A970FF
--tm-color-brand-300:     #BF94FF
--tm-color-brand-soft:    rgba(145, 71, 255, 0.18)
```

說明：

- `brand-500` 為主 CTA、active tab、focus accent
- `brand-soft` 用於 selected surface 或 glow 的輕量底色

### Reward / Mining Accent

```text
--tm-color-reward-500:    #F4C86A
--tm-color-reward-400:    #FFD98E
--tm-color-reward-soft:   rgba(244, 200, 106, 0.16)
```

用途：

- gain 浮字
- reward highlight
- mining 主舞台的小面積點綴

### System Status

```text
--tm-color-success-500:   #3DD6A0
--tm-color-success-soft:  rgba(61, 214, 160, 0.16)

--tm-color-info-500:      #53B7F7
--tm-color-info-soft:     rgba(83, 183, 247, 0.16)

--tm-color-warning-500:   #FFB84D
--tm-color-warning-soft:  rgba(255, 184, 77, 0.16)

--tm-color-error-500:     #FF6B81
--tm-color-error-soft:    rgba(255, 107, 129, 0.16)

--tm-color-cooldown-500:  #8FA2FF
--tm-color-cooldown-soft: rgba(143, 162, 255, 0.16)
```

建議對應：

- `success`: bits 完成、可用狀態
- `info`: session / heartbeat running
- `warning`: unavailable / planned / coming soon
- `error`: auth degraded / request fail
- `cooldown`: click cooldown ring 與數字

---

## 字體 Tokens

### Font Family

建議先以高可讀無襯線為主：

```text
--tm-font-sans: "Inter", "Segoe UI", sans-serif
--tm-font-display: "Inter", "Segoe UI", sans-serif
--tm-font-mono: "JetBrains Mono", "SFMono-Regular", monospace
```

說明：

- 不建議上來就用重 fantasy display font
- `display` 仍用同一家族，靠字重與 spacing 做辨識

### Font Size

```text
--tm-font-size-10: 10px
--tm-font-size-12: 12px
--tm-font-size-14: 14px
--tm-font-size-16: 16px
--tm-font-size-18: 18px
--tm-font-size-20: 20px
--tm-font-size-24: 24px
--tm-font-size-32: 32px
```

### Font Weight

```text
--tm-font-weight-regular: 400
--tm-font-weight-medium:  500
--tm-font-weight-semibold: 600
--tm-font-weight-bold:    700
```

### Line Height

```text
--tm-line-height-tight:   1.15
--tm-line-height-snug:    1.3
--tm-line-height-normal:  1.5
```

### Typography Roles

```text
--tm-type-title-lg:   700 24px/1.15 var(--tm-font-display)
--tm-type-title-md:   700 20px/1.15 var(--tm-font-display)
--tm-type-title-sm:   600 16px/1.3  var(--tm-font-display)

--tm-type-body-md:    400 14px/1.5  var(--tm-font-sans)
--tm-type-body-sm:    400 12px/1.5  var(--tm-font-sans)

--tm-type-label-md:   600 12px/1.3  var(--tm-font-sans)
--tm-type-label-sm:   600 10px/1.3  var(--tm-font-sans)

--tm-type-number-xl:  700 32px/1.0  var(--tm-font-display)
--tm-type-number-lg:  700 24px/1.0  var(--tm-font-display)
--tm-type-number-md:  700 18px/1.0  var(--tm-font-display)
```

---

## Spacing Tokens

### Space Scale

```text
--tm-space-2:   2px
--tm-space-4:   4px
--tm-space-6:   6px
--tm-space-8:   8px
--tm-space-10:  10px
--tm-space-12:  12px
--tm-space-16:  16px
--tm-space-20:  20px
--tm-space-24:  24px
--tm-space-32:  32px
```

### 用法建議

- 微小圖示 / dot 間距：`4-6`
- card 內 padding：`12-16`
- section 間距：`16-20`
- panel 主要段落間距：`20-24`

---

## Radius Tokens

```text
--tm-radius-sm:  8px
--tm-radius-md:  12px
--tm-radius-lg:  16px
--tm-radius-xl:  20px
--tm-radius-pill: 999px
```

原則：

- 小型 chips / status dot container：`sm`
- 一般 card / button：`md`
- mining stage / major panel：`lg`

不要：

- 每個元件都圓到像手機 app 卡片

---

## Shadow Tokens

```text
--tm-shadow-sm:  0 2px 8px rgba(0, 0, 0, 0.18)
--tm-shadow-md:  0 8px 24px rgba(0, 0, 0, 0.24)
--tm-shadow-lg:  0 16px 40px rgba(0, 0, 0, 0.32)

--tm-glow-brand: 0 0 0 1px rgba(145, 71, 255, 0.18), 0 8px 24px rgba(145, 71, 255, 0.16)
--tm-glow-reward: 0 0 0 1px rgba(244, 200, 106, 0.18), 0 8px 24px rgba(244, 200, 106, 0.14)
```

原則：

- 陰影要偏柔，不要像浮空 modal
- glow 只給 CTA、active 狀態、小面積 reward accent

---

## Motion Tokens

```text
--tm-motion-fast:   120ms
--tm-motion-base:   180ms
--tm-motion-slow:   260ms

--tm-ease-standard: cubic-bezier(0.2, 0.0, 0, 1)
--tm-ease-emphasis: cubic-bezier(0.2, 0.8, 0.2, 1)
```

用途：

- hover / press：`fast`
- tab / card transition：`base`
- gain bump / stage emphasis：`slow`

---

## Layout Tokens

```text
--tm-panel-max-width:   360px
--tm-panel-min-width:   318px
--tm-header-height:     48px
--tm-bottom-nav-height: 60px
--tm-mining-stage-min-height: 220px
```

---

## Component Tokens

### Top Status Bar

```text
--tm-header-bg:           rgba(21, 21, 29, 0.88)
--tm-header-border:       var(--tm-color-border-soft)
--tm-header-padding-x:    var(--tm-space-12)
--tm-header-padding-y:    var(--tm-space-10)
```

### Main CTA

```text
--tm-button-primary-bg:         linear-gradient(180deg, #A970FF 0%, #9147FF 100%)
--tm-button-primary-text:       #FFFFFF
--tm-button-primary-radius:     var(--tm-radius-lg)
--tm-button-primary-shadow:     var(--tm-glow-brand)
--tm-button-primary-padding-x:  18px
--tm-button-primary-padding-y:  12px
```

### Cooldown CTA

```text
--tm-button-cooldown-bg:        #2B2D42
--tm-button-cooldown-text:      var(--tm-color-cooldown-500)
--tm-button-cooldown-ring:      var(--tm-color-cooldown-500)
```

### Resource Card

```text
--tm-card-bg:              var(--tm-color-bg-panel)
--tm-card-border:          var(--tm-color-border-soft)
--tm-card-radius:          var(--tm-radius-md)
--tm-card-padding:         var(--tm-space-12)
--tm-card-shadow:          var(--tm-shadow-sm)
```

### Status Banner

```text
--tm-banner-radius:        var(--tm-radius-md)
--tm-banner-padding-x:     var(--tm-space-12)
--tm-banner-padding-y:     var(--tm-space-10)
```

Banner variants：

- success：`success-soft + success-500`
- error：`error-soft + error-500`
- warning：`warning-soft + warning-500`
- info：`info-soft + info-500`

### Bottom Nav

```text
--tm-tabbar-bg:            rgba(17, 17, 24, 0.94)
--tm-tabbar-border:        var(--tm-color-border-soft)
--tm-tabbar-active-bg:     var(--tm-color-brand-soft)
--tm-tabbar-active-text:   var(--tm-color-text-primary)
--tm-tabbar-inactive-text: var(--tm-color-text-muted)
```

---

## 狀態對應建議

### Auth Degraded

- 背景：`error-soft`
- 圖示 / dot：`error-500`
- 文字：`text-primary`

### Session Starting

- 背景：`info-soft`
- 圖示：`info-500`

### Heartbeat Running

- 小狀態文字：`info-500`
- 不需要整塊大面積上色

### Click Gain

- 浮字：`reward-400`
- 微弱 glow：`glow-reward`

### Bits Success

- 成功狀態：`success-soft + success-500`

### Placeholder / Planned

- 背景：`bg-muted`
- 文字：`text-secondary`
- 標籤：`warning-soft + warning-500`

---

## CSS Variable 導入建議

若之後實作，可先從：

```css
:root {
  --tm-color-bg-app: #0E0E13;
  --tm-color-bg-panel: #15151D;
  --tm-color-brand-500: #9147FF;
  --tm-color-text-primary: #F6F7FB;
  --tm-space-12: 12px;
  --tm-radius-md: 12px;
}
```

開始，再依 panel / component 擴充。

---

## 驗收清單

1. 同一個頁面內不會出現三套不同紫色
2. CTA、status、placeholder 都能靠 token 命名一致管理
3. `Home / Missions / Equipment / Shop` 看起來屬於同一個產品
4. 色彩與陰影不會讓畫面滑向 launcher / wallet / SaaS 風格
5. tokens 足以支撐第一版 Figma 與切版，不需要每次重新定義基礎樣式

---

## 後續建議

1. 把這份 token 映射成 Figma color styles / text styles
2. 依 token 補 `Home / Mining` wireframe contract
3. 之後再視需要抽出 shared frontend tokens 與 `dashboard` 共用層
