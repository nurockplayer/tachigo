# Tachigo Extension UI 提示詞

> 用途：整理 `tachimint` 前端畫面設計時可直接丟給 Claude、ChatGPT、Figma AI、v0 等工具的提示詞。
> 範圍：僅針對 Chrome Extension / 小尺寸 extension panel UI，不涵蓋 dashboard。

---

## 1. 產品背景

> **名詞統一：** 本專案前端產品定位是 **Chrome Extension**。若程式碼、舊文件或 API 命名中仍出現 `Twitch Extension`、`extension_jwt` 等字樣，應視為歷史命名或現行串接細節，不代表產品形式仍是 Twitch 的 Extension。

Tachigo 是一個 Twitch 忠誠點數與 Web3 獎勵平台。觀眾在觀看直播時，會透過 heartbeat 累積 `T-Point`，並可透過互動按鈕、Bits 購買與獎勵機制增加參與感。

目前 extension 前端 `tachimint` 的核心內容包含：

- 顯示目前點數餘額
- 觀看時累積點數並即時回饋
- 點擊互動按鈕取得額外點數
- 顯示可用 Bits 購買的獎勵項目
- 區分 `viewer` 與 `broadcaster` 視圖

參考來源：

- [docs/architecture.md](/Users/tachikoma/Documents/Web3/tachigo/docs/architecture.md)
- [docs/watch-to-points-design.md](/Users/tachikoma/Documents/Web3/tachigo/docs/watch-to-points-design.md)
- [tachimint/src/App.tsx](/Users/tachikoma/Documents/Web3/tachigo/tachimint/src/App.tsx)

---

## 2. 共用設計重點

這些重點建議在所有 UI 提示詞中保留：

- 這是直播旁的小尺寸 extension panel，不是完整網站，也不是後台
- 使用者會快速瞥視畫面，所以資訊層級必須非常清楚
- 核心資訊是 `T-Point` 餘額與當前互動狀態
- 需要表現「觀看累積」與「點擊加成」的即時回饋
- 要支援商品 / 獎勵列表、成功狀態、錯誤狀態、空狀態
- 需考慮 `viewer` 與 `broadcaster` 兩種畫面
- 建議面板寬度以 `320px` 到 `420px` 為目標

---

## 3. 負面提示詞

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

## 4. 完整設計提示詞

```text
請設計一個小尺寸的 Twitch / Chrome extension 面板 UI，產品名稱是 Tachigo，風格主軸為「像素遊戲採礦風」。

產品背景：
Tachigo 是一個 Twitch 忠誠點數平台。觀眾在觀看直播時可累積 T-Point，並透過互動按鈕、Bits 購買與獎勵機制增加參與感。這個 UI 是直播旁的小型 extension 面板，不是完整網站，也不是後台系統。

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

請包含以下區塊：

1. Header
- 顯示 Tachigo 品牌名稱
- 像遊戲 HUD 標題列
- 有小型連線狀態 / 在線狀態指示燈
- 可以帶一點像素金屬框、霓虹描邊或像素角框細節

2. 點數總覽區
- 大字顯示目前 T-Point 餘額
- 數字要像資源欄或金幣欄，是畫面主角
- 要有點數增加時的浮動提示，例如 +5、+10
- 視覺感受像「剛挖到礦石 / 水晶 / 能量碎片」

3. 採礦互動區
- 中央主按鈕是像素風採礦角色、稿子、礦鎬、礦石核心或發光礦脈
- 讓人一看就知道「按這個可以挖」
- 點擊後有明確回饋：縮放、震動、碎石火花、像素粒子、數字跳出
- cooldown 狀態可用像素能量條、倒數圈、冷卻遮罩或進度槽呈現
- 不要用普通圓角按鈕敷衍，要像遊戲中的互動物件

4. 商品 / 獎勵區
- 顯示 2 到 4 個可購買項目
- 像遊戲商店或道具欄
- 每個項目有名稱、價格、購買按鈕
- 按鈕像「解鎖」「兌換」「購買」這種遊戲用語
- 商品卡可像道具卡、小型 inventory slot、像素寶箱欄位
- 明確區分可購買、冷卻中、售完、開發中狀態

5. 成功 / 錯誤 / 空狀態
- 成功時像取得寶物、獲得 token、解鎖獎勵
- 錯誤時不要像系統報錯，要像遊戲提示框
- 空狀態可像「礦坑暫時沒有新掉落」「尚未連線礦脈」這種敘事感提示

6. Broadcaster 版本
- 另提供簡化的 broadcaster 視圖
- 不以操作為主，而是顯示「觀眾正在採礦互動」「可啟用獎勵活動」
- 視覺上像控制台或直播事件資訊框

視覺語言要求：
- 深色背景，但要有洞穴、礦坑、夜間霓虹感
- 配色可用：炭黑、石板灰、熔岩橘、電光青、金礦黃、像素紫作點綴
- 不要整片 Twitch 紫
- 可加入像素邊框、像素陰影、格狀 UI、8-bit 小圖示
- 但仍要保有現代 UI 的可讀性與秩序
- 不要太復古到難用，也不要太扁平像一般 SaaS

動效要求：
- 點數增加時有像素數字跳出
- 採礦按鈕點擊時有碎石 / 火花 / 微震動效果
- 成功購買有像素寶箱開啟、閃光或獲得道具感
- 載入中可像能量充填或像素掃描
- 動效要短、清楚、有回饋，不要過度花俏

版面與 UX：
- 單欄設計
- 最重要的是點數餘額與採礦按鈕
- 商品列表放在下方
- 任何文字都要短
- 讓使用者 1 秒內看懂現在有多少點、能不能挖、能不能買

請避免：
- 普通 dashboard 感
- 太企業化
- 太像 NFT 交易平台
- 太厚重的蒸汽龐克
- 太幼稚的兒童遊戲風
- 過多發光導致資訊難讀
- 過度復古造成操作不清楚

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

## 5. Figma AI 短提示詞

```text
設計一個小尺寸 Twitch / Chrome extension 面板 UI，產品名稱是 Tachigo，風格為像素遊戲採礦風。這是一個直播旁的迷你互動面板，讓觀眾查看 T-Point、點擊採礦按鈕、看到點數增加動畫，並購買 2 到 4 個獎勵項目。整體像遊戲 HUD、礦坑資源面板、直播插件的結合。請使用深色洞穴背景、像素邊框、精緻 8-bit / 16-bit 啟發圖形、霓虹點綴但不要太花。重點區塊包含 Header、T-Point 總覽卡、採礦互動按鈕、商品列表、成功狀態、錯誤 / 空狀態，以及 broadcaster 簡化版。畫面需適合 320px 到 420px 寬度，資訊清楚、可讀性高、有即時回饋感，不要做成一般 SaaS dashboard。
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
- 有現代可讀性，不要過度復古
- 不要做成 generic SaaS card layout

元件需求：
1. Header
- 顯示 Tachigo
- 小型在線 / 連線狀態燈
- 有像素風標題條感

2. Points summary card
- 顯示 T-Point 餘額大數字
- 次要文字顯示目前狀態
- 支援 +5 / +10 這種浮動 gain badge 視覺

3. Mining interaction section
- 中央主 CTA 像像素礦鎬 / 礦石核心 / 採礦角色按鈕
- 要有 cooldown 狀態
- 要有 hover / active / disabled 樣式
- 按鈕看起來像遊戲互動物件，不是普通 button

4. Reward list
- 2 到 4 個 reward items
- 每個 item 有名稱、價格、狀態 badge、購買按鈕
- 視覺像 inventory slot / game shop item row

5. Success state
- 像獲得寶物 / 領到 token 的完成區塊

6. Error / empty state
- 文字簡短、帶遊戲提示框語氣

7. Broadcaster view
- 簡化版資訊面板
- 顯示觀眾互動狀態，不強調購買操作

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

## 7. 使用建議

- 若要先探索視覺方向：優先使用「完整設計提示詞」
- 若要快速出稿：優先使用「Figma AI 短提示詞」
- 若要直接開始刻畫面：優先使用「React / Tailwind 生成提示詞」
- 若生成結果太像通用 SaaS，記得追加「負面提示詞」
