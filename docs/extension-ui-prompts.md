# Tachigo Extension UI 提示詞

> 用途：整理 `tachimint` 小尺寸 extension panel 的 UI 設計提示詞。
> 狀態：設計探索參考，不是實作 spec，也不代表產品形式已從 Twitch-hosted runtime 遷移完成。
> 最後更新：2026-04-10

---

## 1. 使用前提

這份文件只處理畫面與互動語言，不定義 auth/runtime 架構。

在目前 repo 中：

- 可運作的實作仍是 Twitch-hosted extension panel
- `window.Twitch.ext`、`extension_jwt`、Bits 相關流程仍存在
- 若要探索 Chrome Extension 版本，應視為未來設計方向，而不是當前實作事實

因此，這裡的 prompt 請用於 UI 概念發想與視覺探索。

---

## 2. 共用設計重點

- 這是直播旁的小尺寸 extension panel，不是完整網站，也不是後台
- 使用者會快速瞥視畫面，所以資訊層級必須非常清楚
- 核心資訊是 `T-Point` 餘額、觀看累積狀態、互動按鈕、獎勵項目
- 需考慮 `viewer` 與 `broadcaster` 兩種畫面
- 不要做成 dashboard，也不要像完整網站

目前可從現有產品行為抽出的核心區塊：

- 點數餘額總覽
- 觀看累積 / heartbeat 回饋
- 點擊加成互動
- Bits 獎勵或購買項目
- 成功 / 錯誤 / 空狀態

---

## 3. 負面提示詞

```text
避免：
- 普通 dashboard 感
- 過度企業化
- 太像 NFT 交易平台
- 大量表格與資料卡
- 太厚重的蒸汽龐克
- 太幼稚的兒童遊戲風
- 過多發光導致資訊難讀
- 太平、太 generic 的白底紫按鈕版型
```

---

## 4. 完整設計提示詞

```text
請設計一個小尺寸的 extension panel UI，產品名稱是 Tachigo，風格主軸為「像素遊戲採礦風」。

產品背景：
Tachigo 是一個 Twitch 忠誠點數與 Web3 獎勵平台。觀眾在觀看直播時會累積 T-Point，並透過互動按鈕、Bits 獎勵與回饋機制增加參與感。這個 UI 是直播旁的小型 extension panel，不是完整網站，也不是後台。

重要限制：
- 這份設計提示詞只處理 UI，不定義 auth/runtime 架構
- 若要把畫面 framing 成 Chrome Extension，請視為未來概念稿，不要假設現有實作已完成遷移

核心風格方向：
- 像素遊戲
- 採礦 / 挖寶 / 地下城資源收集感
- 8-bit / 16-bit 啟發，但整體仍需現代、乾淨、可讀
- 像直播畫面旁的小型遊戲 HUD
- 要有「挖到資源」的即時回饋感

畫面需求：

1. Header
- 顯示 Tachigo 品牌名稱
- 小型在線狀態指示
- 像遊戲 HUD 標題列

2. 點數總覽區
- 大字顯示目前 T-Point 餘額
- 點數增加時有即時浮動提示，例如 +5、+10

3. 互動區
- 主按鈕像礦鎬、礦石核心或像素採礦物件
- 點擊後要有清楚回饋
- 可呈現 cooldown / loading / disabled 狀態

4. 獎勵區
- 2 到 4 個 reward items
- 每個 item 有名稱、價格、狀態與 CTA
- 看起來像遊戲商店或 inventory slot

5. 狀態區
- 成功時像獲得資源或解鎖獎勵
- 錯誤時像遊戲提示框，不要像系統報錯
- 空狀態可有輕微敘事感

6. Broadcaster 視圖
- 簡化版資訊面板
- 以觀眾互動狀態與活動資訊為主

視覺要求：
- 深色礦坑 / 洞穴背景
- 配色可用炭黑、石板灰、礦石藍綠、熔岩橘、金礦黃
- 可加入 pixel border、hard shadow、tiny grid
- 保持資訊清楚，不要過度復古到難用
```

---

## 5. Figma AI 短提示詞

```text
設計一個小尺寸 extension panel UI，產品名稱是 Tachigo，風格為像素遊戲採礦風。這是一個直播旁的迷你互動面板，讓觀眾查看 T-Point、看到點數增加回饋、點擊互動按鈕，並瀏覽 2 到 4 個獎勵項目。畫面需適合 320px 到 420px 寬度，像遊戲 HUD 與直播插件的結合，不要做成 generic SaaS dashboard。
```

---

## 6. React / Tailwind 生成提示詞

```text
請生成一個 React + TypeScript + Tailwind 的小尺寸 extension panel UI，產品名稱是 Tachigo，風格是像素遊戲採礦風。

需求：
- 適合 320px 到 420px 寬
- 單欄 compact layout
- 深色礦坑 / 洞穴背景
- 像素遊戲 HUD 視覺
- 顯示 T-Point 餘額、互動按鈕、reward list、成功/錯誤/空狀態
- 不要做成 generic SaaS card layout

請先輸出靜態 UI mock，不需要串 API。
```
