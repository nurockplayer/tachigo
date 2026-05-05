# Tachimint UI Handoff Guide (Codex)

> 用途：給新加入的前端同事快速上手 `tachimint` 的 UI 修改工作。
> 定位：這是一份實作導向的交接文件，不是產品 spec，也不是 Chrome Extension migration 文件。

---

## 1. 先看這裡

`tachimint` 目前是 **Twitch-hosted extension panel**。

它的設計限制很簡單：

- 小尺寸 panel，不是完整網站
- 單欄 compact layout
- 使用者只會快速瞥一眼
- 核心只看 3 件事：
  - 現在有多少 Points
  - 現在能不能挖
  - 現在能不能買

如果你是第一次改畫面，先看這 4 個檔案：

- `tachimint/src/App.tsx`
  - 畫面結構與狀態切換
- `tachimint/src/index.css`
  - 樣式 token、spacing、動畫
- `tachimint/src/mock/twitch-ext.ts`
  - 本機 mock 資料來源
- `tachimint/src/hooks/useTwitch.ts`
  - viewer / broadcaster / products / backend ready 條件

本機啟動：

```bash
make dev
```

---

## 2. 新同事建議順序

1. 跑 `make dev`
2. 先看 `App.tsx`，搞清楚有哪些畫面分支
3. 再看 `index.css`，先從換皮開始
4. 需要更多假資料時改 `mock/twitch-ext.ts`
5. 非必要先不要動 hooks

最安全的起手式：

- 先只改 `tachimint/src/index.css`
- 第二步才改 `tachimint/src/App.tsx`
- 最後才碰 hooks 或 API 條件

---

## 3. 畫面地圖

### 固定 4 個主畫面

| 畫面 | 觸發條件 | 內容 |
| --- | --- | --- |
| Loading | `context` 尚未 ready | 置中 spinner |
| Broadcaster | `context.role === 'broadcaster'` | logo + 2 行說明文字 |
| Viewer Default | viewer 且未 success | 點數卡 + 礦鎬按鈕 + 商品區 |
| Viewer Success | Bits success | 成功 icon + `Token received!` + `Close` |

### Viewer 狀態變體

| 狀態 | UI 表現 |
| --- | --- |
| Auth degraded | header 右上紅點、挖礦 disabled |
| Balance unknown | 點數顯示 `—` |
| Heartbeat gain | 點數卡旁浮出 `+N` |
| Click gain | 礦鎬按鈕旁浮出 `+N` |
| Cooldown | 按鈕變淡、下方有倒數 |
| Bits pending | 按鈕價格變 `…` |
| Bits error | 商品區下方紅字錯誤 |
| No products | 顯示 `No products available.` |
| Bits unavailable | 顯示 `Bits not available.` |

---

## 4. 哪些檔案改什麼

### 只想改外觀

改：

- `tachimint/src/index.css`

適合：

- 換配色
- 換陰影 / 邊框 / radius
- 調整卡片與按鈕質感
- 強化 mining / HUD / pixel 風格

### 想改版面或文案

改：

- `tachimint/src/App.tsx`

可能一起改：

- `tachimint/src/i18n/locales/zh-TW/common.json`
- `tachimint/src/i18n/locales/en/common.json`
- `tachimint/src/i18n/locales/zh-CN/common.json`

注意：

- 現在大部分文案仍是英文硬編碼
- i18n 目前只覆蓋少數字串

### 想改互動規則

看：

- `tachimint/src/hooks/useTwitch.ts`
- `tachimint/src/hooks/useHeartbeat.ts`
- `tachimint/src/hooks/useClickBoost.ts`
- `tachimint/src/hooks/useBits.ts`

注意：

- watch-to-points 依賴 `backendReady`
- Bits 流程和 watch 流程是分開的
- backend 掛掉時，Bits 區塊未必會一起消失

### 想讓設計稿更好做

改：

- `tachimint/src/mock/twitch-ext.ts`

適合：

- 增加 mock 商品
- 改成 broadcaster mock
- 模擬不同狀態資料

---

## 5. 不要踩這些坑

- 不要把它做成 dashboard
- 不要加新頁籤或多步驟流程
- 不要順手加 modal / drawer / toast 系統
- 不要把文件直接寫成 Chrome Extension 現況
- 不要在單一 UI PR 內混 runtime 改造

目前已知邊界：

- 所有狀態都 inline 顯示
- 沒有 logged-out 專用畫面
- `Close` 目前是 `window.location.reload()`
- 礦鎬按鈕是 `64px` 圓角方塊
- `dev` badge 文案目前就是 `dev`

---

## 6. 樣式基線

如果只是想先升級質感，先沿用這些 token：

| Token | 值 |
| --- | --- |
| `--bg` | `#0e0e10` |
| `--surface` | `#18181b` |
| `--border` | `#2d2d35` |
| `--text` | `#adadb8` |
| `--text-h` | `#efeff1` |
| `--accent` | `#a970ff` |
| `--accent-hover` | `#bf93ff` |
| `--success` | `#00c853` |
| `--error` | `#ff5252` |
| `--bits` | `#f5a623` |

目前 UI 比較像：

- Twitch 深色卡片
- 小幅 hover / bump / float 動畫
- 還沒有完整品牌化美術語言

所以最適合的方向是：

- 保持現有資訊結構
- 直接做視覺升級
- 不要一次加太多新流程

---

## 7. 交付前檢查清單

改完 UI 後，至少確認這些畫面還成立：

- Loading
- Broadcaster
- Viewer Default
- Viewer Success
- Balance gain
- Click gain
- Cooldown
- Bits pending
- No products
- Bits unavailable
- Bits error

如果這輪只是換皮，請特別確認：

- 沒有把層級做得比現在更難讀
- 主視覺仍然是 Points 與採礦按鈕
- 商品列沒有擠壞
- 小尺寸寬度下仍可讀

---

## 8. 可直接貼給 AI 的 prompt

### A. 視覺探索

```text
請幫我重畫一個 Twitch extension panel UI，產品名稱是 Tachigo。這不是完整網站，也不是 dashboard，而是直播旁 320px 到 420px 的小尺寸互動面板。

請保留現有結構，不要擴張功能流程。畫面必須包含：
- Header 品牌列
- Points 餘額卡
- 主互動採礦按鈕
- 商品列表
- loading / broadcaster / success / error / empty 狀態

風格方向：
- dark
- mining / treasure / cave
- game HUD
- pixel-inspired but still readable
- 不要 generic SaaS
- 不要 NFT marketplace

請輸出：
- 視覺方向摘要
- 色票與字體建議
- 四個主畫面的描述
- viewer 狀態變體建議
```

### B. React mock

```text
請產出一個 React + TypeScript + Tailwind 的靜態 UI mock，用來重畫 Twitch extension panel「Tachigo」。

限制：
- 寬度約 360px
- 單欄 compact layout
- 不新增頁籤、modal、side nav
- 不串 API
- 保留 loading、broadcaster、viewer、success 狀態

viewer 畫面要有：
- Header
- Points 餘額卡
- 採礦主按鈕
- gain feedback
- cooldown
- 商品列表
- error / empty 狀態
```

### C. 只改設計不改流程

```text
請針對既有 React extension panel 提供「只改設計、不改流程」的重畫建議。

不可改：
- broadcaster / viewer 分流
- watch-to-points 與 Bits 成功 / 失敗狀態
- header / balance / mine / products 主結構

請提供：
- 哪些 CSS token 最值得先調整
- 哪些視覺問題最影響質感
- 如何在不增加太多元件的前提下，讓畫面更像品牌化遊戲 UI
```

---

## 9. 相關檔案

- `tachimint/src/App.tsx`
- `tachimint/src/index.css`
- `tachimint/src/mock/twitch-ext.ts`
- `tachimint/src/hooks/useTwitch.ts`
- `tachimint/src/hooks/useHeartbeat.ts`
- `tachimint/src/hooks/useClickBoost.ts`
- `tachimint/src/hooks/useBits.ts`
- `docs/watch-to-points-design.md`
- `docs/history/2026-04-16-chrome-extension-terminology-audit.md`
