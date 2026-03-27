# 功能討論文件

> **專案名稱：** tachigo、tachiya
> **日期：** 2026-03-27
> **參與成員：** nurockplayer、新理解、5lime、Erick、YUAN

---

## 1. 專案概述

tachigo 是一個整合 Twitch Extension 的 Web3 忠誠點數平台，讓觀眾透過觀看直播（定時 Heartbeat 回報）累積鏈上忠誠代幣，並可用於tachiya商城折扣、虛擬頭像客製化、投票/賭博等消費場景。

---

## 2. 功能清單

| # | 功能名稱 | 簡短描述 | 優先級（高/中/低） | 負責人 |
|---|---------|---------|-------------------|--------|
| 1 | 點數系統 | 觀眾觀看直播（Heartbeat）累積點數，雙帳本記帳 | 高 | 5lime |
| 2 | Agency / Streamer 管理 | 建立 Agency 與 Streamer 關聯，管理各自的代幣設定 | 高 | 5lime |
| 3 | 商城折扣 | 觀眾用持有代幣換取商城折扣，消耗 spendable_balance | 高 | Erick / YUAN |
| 4 | 虛擬頭像客製化 | 消耗平台代幣解鎖頭像外觀，提供屬性加成 | 中 | 新理解 / YUAN |
| 5 | 投票 / 賭博機制 | 觀眾用鏈下餘額參與投票或賭博活動 | 中 | Erick |
| 6 | 後台管理介面 | Agency / Streamer 管理後台（Dashboard） | 中 | 新理解 |
| 7 | 鏈上 Token Claim | 鏈下累積點數 → Soulbound ERC-20 on-chain mint | 低 | nurockplayer |
| 8 | 私人直播票券 | 用代幣購買一對一或小組私人直播資格 | 低 | TBD |

---

## 3. 功能拆解

### 功能 1：點數系統

**這個功能在幹嘛？**
> 觀眾在 Twitch Extension 上觀看直播時，定時回報 Heartbeat → 後端驗證在線狀態 → 更新雙帳本（cumulative_total 增加、spendable_balance 增加）。

**這筆資料需要上鏈嗎？**
> MVP 階段不需要，存在後端 PostgreSQL 即可。Phase 2 才將鏈下累積結果 mint 成 Soulbound ERC-20。

#### 前端

- [ ] Extension 畫面顯示目前點數餘額（spendable_balance）
- [ ] 顯示累積總點數（cumulative_total）
- [ ] Heartbeat 回報後即時更新顯示

#### 後端

- [ ] `points_ledger` 資料表（user_id, agency_id, cumulative_total, spendable_balance）
- [ ] API：`POST /watch/heartbeat` — 驗證在線狀態，更新雙帳本
- [ ] API：`GET /points/balance` — 回傳目前餘額

#### 鏈上（智能合約）

- [ ] Phase 2：Soulbound ERC-20，`mint()` 函式（只有後端可呼叫）
- [ ] Phase 2：Factory 模式，每個 Agency 部署獨立合約

#### 資料怎麼流動？

```
觀眾在 Extension 觀看直播
  → 前端定時送 Heartbeat 給後端 POST /watch/heartbeat
  → 後端驗證在線狀態（JWT + 防重放）
  → 更新 points_ledger：cumulative_total ↑、spendable_balance ↑
  → 回傳最新餘額給前端顯示
```

#### 備註

- 依賴：需要 Agency / Streamer 管理先建立，才能知道點數屬於哪個 Agency
- 風險：Heartbeat 頻率與點數發放速率需定義（例：每 N 秒發 M 點）

---

### 功能 2：Agency / Streamer 管理

**這個功能在幹嘛？**
> 建立 Agency（經紀公司）與旗下 Streamer 的關聯，Agency 可設定代幣名稱、折扣比例等參數。

**這筆資料需要上鏈嗎？**
> 不需要，Agency 設定存在後端資料庫即可。Phase 2 才在鏈上部署對應合約。

#### 前端

- [ ] Dashboard：Agency 管理頁面（新增 / 編輯 Agency）
- [ ] Dashboard：Streamer 列表與所屬 Agency 管理
- [ ] 代幣設定介面（折扣比例、代幣名稱）

#### 後端

- [ ] `agencies` 資料表
- [ ] `streamers` 資料表（外鍵連結 agency）
- [ ] API：`POST /agencies` — 建立 Agency
- [ ] API：`GET /agencies/:id/streamers` — 列出旗下 Streamer
- [ ] API：`PUT /agencies/:id/settings` — 更新代幣設定

#### 鏈上（智能合約）

- [ ] Phase 2：AgencyFactory 合約，`deployToken(agencyId)` 為每個 Agency 部署 ERC-20

#### 資料怎麼流動？

```
Agency 管理者登入 Dashboard
  → 建立 Agency，設定代幣名稱與折扣比例
  → 後端儲存至 agencies 資料表
  → 新增 Streamer 並關聯至 Agency
```

#### 備註

- 依賴：Auth 系統（已完成）
- 風險：Agency 與 Streamer 的權限邊界需要明確定義

---

### 功能 3：商城折扣

**這個功能在幹嘛？**
> 觀眾在商城結帳時，系統根據其持有的 Agency 代幣計算折扣，消耗 spendable_balance。

**這筆資料需要上鏈嗎？**
> 不需要，折扣計算與餘額扣除在後端處理，商城透過 Saleor API 串接。

#### 前端

- [ ] 結帳頁面顯示可用折扣金額
- [ ] 確認使用折扣並送出訂單

#### 後端

- [ ] 折扣計算邏輯：`viewer_tokens ÷ streamer_total_tokens` × 折扣上限
- [ ] API：`GET /discount/calculate` — 計算可用折扣
- [ ] API：`POST /discount/apply` — 套用折扣，扣除 spendable_balance
- [ ] 串接 Saleor API 建立訂單

#### 鏈上（智能合約）

- [ ] 不需要

#### 資料怎麼流動？

```
觀眾進入商城結帳
  → 前端呼叫 GET /discount/calculate
  → 後端計算折扣比例（持有量 ÷ 總發行量）
  → 觀眾確認使用折扣
  → 後端扣除 spendable_balance，呼叫 Saleor API 建立訂單
```

#### 備註

- 依賴：點數系統（功能 1）、Saleor 整合
- 風險：Saleor 獨立 repo，API 串接需確認介面

---

### 功能 4：虛擬頭像客製化

**這個功能在幹嘛？**
> 觀眾消耗平台代幣解鎖頭像外觀或屬性加成，spendable_balance 扣減。

**這筆資料需要上鏈嗎？**
> 不需要，頭像狀態存在後端資料庫。

#### 前端

- [ ] 頭像客製化介面（顯示可解鎖的外觀選項與所需點數）
- [ ] 套用外觀後在 Extension 中顯示

#### 後端

- [ ] `avatar_items` 資料表（item_id, cost, bonus_type）
- [ ] `user_avatars` 資料表（user_id, item_id）
- [ ] API：`GET /avatar/items` — 列出可購買項目
- [ ] API：`POST /avatar/unlock` — 解鎖項目，扣除 spendable_balance

#### 鏈上（智能合約）

- [ ] 不需要

#### 資料怎麼流動？

```
觀眾在 Extension 開啟頭像商店
  → 前端呼叫 GET /avatar/items 顯示選項
  → 觀眾選擇並確認購買
  → 後端扣除 spendable_balance，記錄解鎖項目
  → 前端更新頭像顯示
```

#### 備註

- 依賴：點數系統（功能 1）
- 疑問：屬性加成的具體效果待定義（加在什麼上？）

---

### 功能 5：投票 / 賭博機制

**這個功能在幹嘛？**
> 觀眾用鏈下 spendable_balance 參與直播中的投票或賭博活動，結果決定點數重新分配。

**這筆資料需要上鏈嗎？**
> 不需要，完全鏈下處理（Soulbound 代幣不可轉移，所以餘額移動只在 DB 發生）。

#### 前端

- [ ] 投票 / 賭博活動 UI（顯示選項、賠率、下注介面）
- [ ] 結果公告與點數變動通知

#### 後端

- [ ] API：`POST /events/create` — Streamer 建立活動
- [ ] API：`POST /events/:id/bet` — 觀眾下注，鎖定 spendable_balance
- [ ] API：`POST /events/:id/settle` — 結算，重新分配餘額
- [ ] 防作弊機制（下注截止時間、結果驗證）

#### 鏈上（智能合約）

- [ ] 不需要（Soulbound 本身不可轉移，餘額移動只在 DB）

#### 資料怎麼流動？

```
Streamer 建立賭博活動
  → 觀眾在截止前下注（spendable_balance 暫時鎖定）
  → 活動結束，Streamer 公布結果
  → 後端結算：贏家 spendable_balance ↑，輸家扣除
```

#### 備註

- 依賴：點數系統（功能 1）、Agency / Streamer 管理（功能 2）
- 風險：防作弊機制複雜度高，MVP 可先做簡單投票，賭博延後

---

## 4. 開發順序

```
Phase 1（MVP）：
  → 點數系統（雙帳本記帳）
  → Agency / Streamer 管理
  → 商城折扣（Saleor 串接）

Phase 2：
  → 虛擬頭像客製化
  → 投票 / 賭博機制
  → 後台管理介面（Dashboard）

Phase 3：
  → 鏈上 Token Claim（Soulbound ERC-20 mint）
  → 私人直播票券
```

---

## 5. 技術架構

| 層級 | 技術選擇 | 備註 |
|------|---------|------|
| **前端框架** | React 19 + TypeScript + Vite | Twitch Extension（tachimint） |
| **樣式** | TBD | |
| **後端框架** | Go + Gin + GORM | |
| **資料庫** | PostgreSQL 16 | Docker Compose 本地開發 |
| **認證** | JWT + Twitch OAuth + Google OAuth + Web3/SIWE | 已完成 |
| **商城** | Saleor（獨立 repo，API 串接） | |
| **鏈** | Sepolia（測試網） | Phase 2+ |
| **合約語言** | Solidity | Phase 2+ |
| **合約框架** | Foundry + OpenZeppelin | Phase 2+ |
| **Token 類型** | Soulbound ERC-20（不可轉移、可燒毀） | Factory 模式，每個 Agency 一份合約 |
| **部署** | Docker Compose | 正式環境 TBD |

---

## 6. 待討論 / 未決定事項

- [ ] 賭博機制的防作弊設計（結果如何驗證公正性？）
- [ ] 投票與賭博是否為同一套機制，還是分開實作？
- [ ] Avatar 屬性加成的具體效果定義
- [ ] 私人直播的範圍：一對一 vs 小組（Phase 3 再討論）
- [ ] Dashboard 的權限設計：Agency 管理者 vs Streamer 各自能看到什麼？
- [ ] Saleor 的部署與 tachigo 後端的 API 串接介面

---

## 7. 參考資源

- [docs/architecture.md](architecture.md) — 系統整體架構圖
- [GitHub Issue #12](https://github.com/nurockplayer/tachigo/issues/12) — Token 系統架構
- [GitHub Issue #15](https://github.com/nurockplayer/tachigo/issues/15) — 商城與 Token 消費機制
- [GitHub Issue #17](https://github.com/nurockplayer/tachigo/issues/17) — Token 經濟設計與 Soulbound 衝突
- [GitHub Issue #18](https://github.com/nurockplayer/tachigo/issues/18) — 後台管理介面

---

## 附錄 A：初學者建議的思考順序

> 如果你不知道從哪裡開始，按這個順序思考：

1. **使用者能做什麼？** → 先列功能，不管技術
2. **畫面長怎樣？** → 簡單畫 wireframe 或列出頁面
3. **資料從哪來、存到哪？** → 想清楚資料流（前端 ↔ 後端 ↔ 鏈上）
4. **哪些東西要上鏈？** → 只有需要去中心化信任的才上鏈，其他用後端
5. **最小能跑的版本是什麼？** → 先做 MVP，再疊加功能
6. **誰做什麼？** → 按能力分工，不要一個人全包

---

## 附錄 B：完整填寫範例 — NFT Mint 功能

> 以下示範一個完整的功能拆解，讓你知道填出來應該長什麼樣。

### 功能：NFT Mint

**這個功能在幹嘛？**
> 使用者連接錢包後，按下 Mint 按鈕 → 付 ETH 鑄造一個 NFT → 畫面顯示鑄造成功與 NFT 圖片。

**這筆資料需要上鏈嗎？**
> 需要。NFT 的擁有權紀錄必須上鏈，這是核心價值。
> 但 NFT 的圖片本身太大，存在 IPFS，合約只存圖片的連結（tokenURI）。

#### 前端

- [ ] Mint 頁面 UI（顯示價格、剩餘數量、Mint 按鈕）
- [ ] 連接錢包按鈕（MetaMask）
- [ ] 按下 Mint 後呼叫合約，顯示交易等待中的 loading 狀態
- [ ] 交易成功後，顯示 NFT 圖片與 Etherscan 連結

#### 後端

- [ ] API：`GET /api/nft/remaining` — 回傳剩餘可 Mint 數量（讀取合約 totalSupply）
- [ ] API：`GET /api/nft/{tokenId}` — 回傳 NFT metadata（給 OpenSea 讀取用）

#### 鏈上（智能合約）

- [ ] ERC-721 合約，包含 `mint()` 函式（payable，檢查價格和數量上限）
- [ ] `totalSupply()` 讀取已鑄造數量
- [ ] `withdraw()` 讓合約擁有者提領收入
- [ ] 部署到測試鏈（Sepolia）做測試

#### 資料怎麼流動？

```
使用者點擊 Mint
  → 前端用 wagmi 呼叫合約的 mint()，附帶 0.01 ETH
  → 合約檢查金額和數量，鑄造 NFT，emit Transfer 事件
  → 前端監聽交易 receipt，確認成功
  → 前端呼叫後端 GET /api/nft/{tokenId} 取得圖片連結
  → 顯示成功畫面
```

#### 備註

- 依賴：需要先完成「錢包連接」功能
- 風險：IPFS 上傳圖片的流程需要研究（用 Pinata？NFT.Storage？）
- 疑問：要不要做白名單（whitelist）機制？先討論
