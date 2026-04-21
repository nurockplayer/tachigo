# Tachigo Extension — Figma 設計提示詞

> 用途：設計師重新設計 `tachimint` 畫面時，可直接貼給 Figma AI、ChatGPT、Claude、v0 等工具的提示詞集。
> 配合閱讀：[extension-ui-prompts.md](extension-ui-prompts.md)（現有 UI 結構、token、hooks 說明）

---

## 1. 負面提示詞（通用）

```text
避免：
- 普通 dashboard 感
- 過度企業化
- 太像 NFT 交易平台
- 白底紫按鈕的 generic SaaS 模板感
- 太多表格或資料卡
- 太厚重的蒸汽龐克
- 太幼稚的兒童遊戲風
- 過多發光導致資訊難讀
- 過度復古造成操作不清楚
- 擁擠、資訊層級混亂
```

---

## 2. 完整設計提示詞

```text
請設計一個小尺寸的 extension panel UI，產品名稱是 Tachigo，風格主軸為「像素遊戲採礦風」。

產品背景：
Tachigo 是一個 Twitch 忠誠點數平台。觀眾在觀看直播時可累積 Points，並透過互動按鈕、Bits 購買與獎勵機制增加參與感。這個 UI 是直播旁的小型 extension 面板，不是完整網站，也不是後台系統。

核心風格方向：
- 像素遊戲
- 採礦 / 挖寶 / 地下城資源收集感
- 8-bit / 16-bit 啟發，但整體仍需現代、乾淨、可讀
- 有遊戲 HUD 的感覺，像是直播畫面旁的迷你任務面板
- 小而精緻，不要做成老派粗糙 pixel art
- 要有「挖到資源」的爽感與即時回饋

畫面尺寸與情境：
- 適合 320px 到 420px 寬的小面板
- 使用者正在看 Twitch 直播，只會快速瞥視與點幾下
- 資訊密度可稍高，但層級要非常清楚
- 要像一個遊戲插件，而不是一般 Web App

請設計以下 4 個主畫面 Frame：

Frame A：Loading
- 深色背景 + 置中 loading 動畫
- 無文字、無 CTA
- 載入動畫可像能量充填或像素掃描

Frame B：Broadcaster View
- Header：Tachigo logo
- Body：僅顯示 2 行靜態說明文字（"Broadcaster view" / "Viewers can spend Bits to earn rewards."）
- 無任何互動元素或動態資料
- 視覺上像控制台說明框，而不是操作介面

Frame C：Viewer Default（主畫面）

1. Header
- 顯示 Tachigo 品牌名稱
- 像遊戲 HUD 標題列
- 有一個紅色小圓點（offline indicator）：正常狀態下隱藏，僅後端斷線時出現
- 可以帶一點像素金屬框、霓虹描邊或像素角框細節

2. 點數總覽區
- Label 文字「Points」
- 大字顯示目前點數餘額（無資料時顯示「—」）
- 數字要像資源欄或金幣欄，是畫面主角
- 浮動 gain chip（+N points），在點數增加時 fade up 出現於餘額旁

3. 採礦互動區
- 中央主按鈕是 ⛏ 礦鎬符號（64px 圓角方塊）
- 讓人一看就知道「按這個可以挖」
- 點擊後按鈕上方出現浮動 +N 文字（click gain）
- cooldown 狀態：按鈕下方顯示倒數秒數，按鈕呈 disabled 視覺
- 不要用普通圓角按鈕敷衍，要像遊戲中的互動物件

4. 商品 / 獎勵區
- 顯示 2 到 4 個可購買項目
- 每個項目有名稱 + Bits 購買按鈕（Twitch 官方樣式：藍紫漸層 + ♦ icon）
- 商品可附帶 [dev] badge（小 chip），表示開發環境測試商品
- Bits Pending 狀態：按鈕價格文字顯示為「…」
- 空狀態：整個商品區換成 1 行提示文字（"No products available." 或 "Bits not available."）
- 錯誤狀態：商品區下方出現 inline error 訊息（不是 toast / modal）

Frame D：Viewer Success
- 保留 Header 與點數總覽區
- 商品區完全換成：
  - 綠色圓形 ✓ icon（大型、置中）
  - "Token received!" 文字
  - "Close" 幽靈按鈕（outline style）

Viewer Default 額外需要 7 個狀態變體（Component Properties 或 Variants）：
- Offline：header 右側顯示紅色小圓點
- Balance Gain：餘額旁 +N points chip 出現
- Mine Gain：按鈕上方 +N 浮字出現
- Cooldown：按鈕 disabled + 下方倒數秒數
- Bits Pending：商品按鈕顯示 …
- Empty：商品區換成提示文字
- Error：商品區下方出現 inline 錯誤訊息

重要限制（所有 frame 皆適用）：
- 所有狀態 inline 顯示，不使用 modal / toast / overlay / retry button
- Bits 按鈕需保留 Twitch 官方樣式空間（藍紫漸層 + ♦ icon），不要自由替換
- 沒有 logged-out 專屬畫面

視覺語言要求：
- 深色背景，但要有洞穴、礦坑、夜間霓虹感
- 配色可用：炭黑、石板灰、熔岩橘、電光青、金礦黃、像素紫作點綴
- 不要整片 Twitch 紫
- 可加入像素邊框、像素陰影、格狀 UI、8-bit 小圖示
- 但仍要保有現代 UI 的可讀性與秩序
- 不要太復古到難用，也不要太扁平像一般 SaaS

動效要求：
- 點數增加時有 gain chip 從餘額旁 fade up 跳出（+N points）
- 採礦按鈕點擊時有碎石 / 火花 / 微震動效果，按鈕上方有 +N 浮字
- Viewer Success 畫面帶像素閃光或獲得道具感
- 載入中可像能量充填或像素掃描
- 動效要短、清楚、有回饋，不要過度花俏

版面與 UX：
- 單欄設計
- 最重要的是點數餘額與採礦按鈕
- 商品列表放在下方
- 任何文字都要短
- 讓使用者 1 秒內看懂現在有多少點、能不能挖、能不能買

請輸出：
- 完整視覺方向描述
- 元件拆解
- 色票建議
- 字體建議
- 小尺寸面板 wireframe 描述
- 若可以，請直接產出高保真 mockup 說明
```

可再補一句強化畫風：

```text
請參考「像素礦坑 + 直播 HUD + 小型 RPG 資源面板」的混合風格，讓 UI 看起來像一個可互動的直播遊戲插件，而不是普通的網頁小工具。
```

---

## 3. Figma AI 短提示詞

```text
設計一個小尺寸 Twitch extension panel UI，產品名稱是 Tachigo，風格為像素遊戲採礦風，寬度 320–420px，深色洞穴背景。

需要設計 4 個主畫面 Frame：
1. Loading — 置中 spinner，無文字
2. Broadcaster View — logo + 2 行靜態說明文字，無互動元素
3. Viewer Default — logo（含離線紅點 variant）、Points 餘額大數字、64px ⛏ 採礦按鈕、商品列表
4. Viewer Success — 保留 logo 與餘額，商品區換成 ✓ icon + "Token received!" + Close 幽靈按鈕

Viewer Default 需附帶 7 個狀態變體：Offline（紅點）、Balance Gain（+N chip）、Mine Gain（+N 浮字）、Cooldown（倒數 + disabled 按鈕）、Bits Pending（按鈕顯示 …）、Empty（無商品提示文字）、Error（inline 紅色錯誤訊息）。

重要限制：所有狀態 inline 顯示，不使用 modal / toast / overlay。Bits 購買按鈕使用 Twitch 官方樣式（藍紫漸層 + ♦ icon）。

整體像遊戲 HUD、礦坑資源面板、直播插件的結合。使用像素邊框、精緻 8-bit / 16-bit 啟發圖形、霓虹點綴但不要太花，維持可讀性。不要做成一般 SaaS dashboard。
```

---

## 4. React / Tailwind 生成提示詞

```text
請生成一個 React + TypeScript + Tailwind 的小尺寸 extension panel UI，產品名稱是 Tachigo，風格是像素遊戲採礦風。

需求：
- 適合 320px 到 420px 寬
- 單欄 compact layout
- 深色礦坑 / 洞穴背景
- 像素遊戲 HUD 視覺
- 有現代可讀性，不要過度復古
- 不要做成 generic SaaS card layout

元件需求：
1. Header
- 顯示「tachigo」品牌名稱
- 右側有一個紅色小圓點（offline dot）：預設 hidden，透過 prop `offline` 顯示
- 有像素風標題條感

2. Balance Section
- Label 文字「Points」
- 顯示餘額大數字；無資料時顯示「—」
- 支援 gain chip（+N points）：絕對定位於餘額旁，透過 prop `gainText` 控制，帶 fade-up animation class

3. Mine Button
- 中央主 CTA，64px 圓角方塊，顯示 ⛏ 礦鎬符號
- 支援 disabled 狀態（cooldown 時）
- 按鈕上方可顯示 click gain 浮字（透過 prop `clickGain`）
- 按鈕下方可顯示 cooldown 倒數秒數（透過 prop `cooldown`）
- 按鈕看起來像遊戲互動物件，不是普通 button

4. Products List（預設狀態）
- 2 到 4 個 reward items
- 每個 item：商品名稱 + 可選 [dev] badge + Bits 購買按鈕
- Bits 按鈕樣式預留 Twitch 官方元件（藍紫漸層 + ♦ icon）；靜態 mock 可用近似樣式
- Pending 狀態：按鈕價格文字替換為 `…`
- 視覺像 inventory slot / game shop item row

5. Empty / Hint 狀態
- 商品區換成單行提示文字（"No products available." 或 "Bits not available."）

6. Error 狀態
- 商品區下方顯示 inline error 訊息（紅色文字）
- 不使用 toast / modal / overlay

7. Success 狀態（取代商品區）
- 綠色圓形 ✓ icon（大型）
- "Token received!" 文字
- "Close" 幽靈按鈕（outline style）
- Header 與 Balance Section 正常顯示，只替換下方區塊

8. Broadcaster View（獨立 variant）
- Logo + 2 行靜態文字："Broadcaster view" / "Viewers can spend Bits to earn rewards."
- 無任何互動元素或動態資料

設計要求：
- 使用 CSS variables 或 Tailwind theme tokens 定義顏色
- 配色建議：炭黑、石板灰、礦石藍綠、熔岩橘、金礦黃、少量像素紫
- 可加入 pixel border、hard shadow、tiny grid、energy glow
- 但要維持內容清楚
- 動效只需用 class 呈現設計意圖，不必實作複雜 animation logic

請輸出：
- 可直接貼上的 React component
- 使用 Tailwind class
- 拆成清楚區塊
- 先做靜態 UI mock，不需要串 API
```

---

## 5. 使用建議

- 若要先探索視覺方向：優先使用「完整設計提示詞」（§2）
- 若要快速出稿：優先使用「Figma AI 短提示詞」（§3）
- 若要直接開始刻畫面：優先使用「React / Tailwind 生成提示詞」（§4）
- 若生成結果太像通用 SaaS，記得追加「負面提示詞」（§1）
- 設計前請先閱讀 [extension-ui-prompts.md](extension-ui-prompts.md) 了解現有 token 與 UI 結構
