# RaffleDetailPage 海洋主題改版 — 設計規格

## 背景

現有 `apps/dashboard/src/pages/RaffleDetailPage.tsx` 使用純 CSS 扭蛋機動畫，視覺效果不足。改版目標：以提供的海洋主題扭蛋機插圖為主視覺，打造沉浸感更強的抽獎操作介面。

---

## 版面結構

三欄水平佈局（`flex` 排列）：

```text
[左欄 160px] [中欄 flex:1] [右欄 160px]
```

- **左欄**：玻璃面板（glassmorphism），包含「參加方式」說明卡 + 即時聊天室
- **中欄**：海洋主題扭蛋機插圖（主視覺）+ 「開始抽獎」按鈕 + 提示文字
- **右欄**：玻璃面板，中獎名單（等待中顯示 ? 圓圈，抽出後即時更新）

左右欄寬約為中欄的 40%，讓扭蛋機圖片成為絕對視覺重心。

---

## 視覺規格

### 背景
- 深海藍漸層：`#050e24` → `#0a1e4a`
- 無背景圖（背景保持單純深色）

### 扭蛋機圖片
- 路徑：`apps/dashboard/src/assets/raffle-bg.jpg.png`
- 顯示方式：`width: 100%; height: auto; object-fit: contain`
- 效果：`drop-shadow(0 0 50px rgba(30,100,255,.55))`

### 標題區（中欄頂部）
- 「觀眾抽獎」大標題，藍色發光 text-shadow
- 副標「扭蛋抽出幸運觀眾！」膠囊標籤樣式
- 星星裝飾（閃爍 animation）

### 玻璃面板
- `background: rgba(4,14,52,.55)`
- `border: 1px solid rgba(80,160,255,.22)`
- `backdrop-filter: blur(14px)`
- `border-radius: 14px`

### LIVE badge
- 固定左上角，紅點閃爍動畫

---

## 互動行為

### 抽獎流程
1. 點擊「開始抽獎」
2. 扭蛋機圖片震動（`shake` keyframe，0.5s）
3. 彩球從機台出口彈出（`popout` keyframe，1.2s，`cubic-bezier(.22,.68,0,1.15)`）
4. 中獎者彈窗出現（全螢幕遮罩 + 彈窗 bounce in）
5. 關閉彈窗 → 中獎名單右欄即時新增該得獎者（slide-in animation）

### 聊天室（左欄）
- 初始顯示 7 則靜態留言
- 每 2.8 秒自動新增一則模擬留言，超過 8 則時移除最舊的
- 新留言有 fade-in slide-down animation

### 中獎名單（右欄）
- 初始：? 圓圈 + 「等待抽獎中...」
- 首次抽出後：隱藏佔位符，顯示名單
- 最多顯示 5 筆，最新在最上方
- 每筆格式：`#排名 / 名字 / 時間`

---

## 現有功能的處置

本次改版**只替換視覺展示層**，不動現有管理功能。原 `RaffleDetailPage.tsx` 的以下功能保留：

| 功能 | 處置 |
|---|---|
| CSV 上傳 | 保留，移至側邊或折疊區 |
| 鎖定名單（Activate） | 保留按鈕 |
| 統計（匯入 / 已抽 / 剩餘） | 保留 StatCard |
| Discord Webhook 設定 | 保留 |
| 結束活動 | 保留 |
| 返回列表按鈕 | 保留 |

管理功能的版面安排在實作計劃階段確定（可考慮折疊 panel 或頁面頂部 bar）。

---

## 檔案異動範圍

| 檔案 | 動作 |
|---|---|
| `apps/dashboard/src/pages/RaffleDetailPage.tsx` | 主要改版 |
| `apps/dashboard/src/assets/raffle-bg.jpg.png` | 新增至版控（binary image asset） |
| `apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx` | 新增 2 個測試（WinnerModal 顯示 / 關閉） |

不涉及後端、API、路由。

---

## 完成條件

- [ ] 三欄佈局正確，中欄明顯大於左右欄
- [ ] 扭蛋機圖片完整顯示，無裁切
- [ ] 點擊抽獎 → 震動 → 彩球彈出 → 中獎彈窗
- [ ] 聊天室自動滾動新留言
- [ ] 右欄中獎名單即時更新
- [ ] 所有原有管理功能（CSV、鎖定、統計、Discord、結束）均可操作
- [ ] TypeScript 型別無錯誤
