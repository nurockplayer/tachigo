# Sepolia Claim / Mint MVP 建議出票順序

> 用途：把 `plans/sepolia-claim-mint-mvp.md` 拆成可實際建立 issue / PR 的順序。
> 原則：先跑通 Sepolia MVP，不把主網、Router Contract、完整 dApp UX 混進來。

---

## 1. 建議順序總覽

1. 合約 review 收尾
2. Sepolia 部署腳本與部署文件
3. Account linking 與 claim target address 規格
4. claim idempotency 設計與 job 記錄 schema（**先於 mint service**）
5. 後端 Sepolia mint service
6. `claim-sepolia` API
7. 前端最小 claim 入口
8. E2E 驗證與文件更新

---

## 2. 建議 issue 拆法

## Issue 1: Soulbound ERC-20 review 收尾

### 目標

讓 `TachiToken` 合約語意完整，可作為 Sepolia MVP 的鏈上載體。

### 範圍

- 修正 review feedback
- 確認 soulbound 語意完整
- 補測試

### 完成條件

- `forge test` 全綠
- 合約 review 結束
- 可進入部署準備

---

## Issue 2: Sepolia 部署腳本與部署文件

### 目標

讓團隊能一致地把 `TachiToken` 部署到 Sepolia，並保留 deploy 資訊。

### 範圍

- Foundry deploy script
- deploy 參數說明
- owner / signer 策略記錄
- Sepolia contract address 記錄方式

### 完成條件

- 可用指令完成部署
- 有 contract address / chain id / tx hash
- 文件可重現部署流程
- **owner 策略已明確記錄（MVP = deployer wallet，私鑰透過環境變數管理）**

---

## Issue 3: Account linking 與 claim target address 規格

### 目標

定義並實作「Twitch user ↔ wallet address」的關聯機制，讓後端 claim 時能穩定取得目標地址。

### 背景

目前 repo 有兩套獨立的 user record：
- Twitch Extension JWT 路徑（tachimint viewer）
- Web3 SIWE 路徑（`/auth/web3/nonce` + `/auth/web3/verify`）

claim 的前提是「知道哪個 Twitch viewer 對應到哪個 wallet address」，這需要明確的 linking 流程。

### 範圍

- 決定 account linking 觸發的畫面與時機（建議：dashboard 或獨立頁面，不在 extension 內）
- 定義 linking API（Twitch user ↔ Web3 provider 關聯）
- 確認或補最小 schema（`auth_providers` 是否足夠，或需要新的關聯欄位）
- 定義使用者主 wallet address 規則（MVP 先只取一個主地址）
- 決定沒有綁定 wallet 時 API / 前端的處理方式

### 完成條件

- account linking 流程有明確規格（畫面 + API）
- 後端 service 可穩定取得 claim target address
- 未綁定 wallet 的錯誤路徑有明確定義

---

## Issue 4: claim idempotency 設計與 job 記錄 schema

### 目標

在實作 mint service 與 claim API 之前，先定義 idempotency 機制與 job 記錄結構，避免後續 API 上線後再補時留下 double mint 風險窗口。

### 範圍

- 定義 idempotency key 規格（request id 來源、格式）
- 設計 `mint_jobs`（或 claim status）資料表 schema
- 定義 tx hash 與 userID / amount 的對應關係
- 定義 reserve → pending → settled / failed 狀態機

### 完成條件

- schema migration 可執行
- 狀態機有明確定義（含失敗解凍路徑）
- 後續 mint service 與 claim API 可直接依此實作

---

## Issue 5: 後端 Sepolia mint service

### 目標

讓後端能安全呼叫 Sepolia 合約 `mint(to, amount)`，並整合 Issue 4 定義的 job 記錄。

### 範圍

- EVM client / signer 初始化
- env 設定（`SEPOLIA_RPC_URL`、`SEPOLIA_CHAIN_ID`、`TACHI_TOKEN_ADDRESS`、`SEPOLIA_SIGNER_KEY`）
- 合約呼叫封裝
- tx hash 回傳
- 寫入 mint_jobs 記錄

### 完成條件

- 可從後端成功送出 Sepolia mint
- 能取得 tx hash
- 失敗可回報明確錯誤
- mint_jobs 狀態正確更新

---

## Issue 6: `claim-sepolia` API

### 目標

新增一條明確的 API，讓使用者把 `T-Point` 兌換成 Sepolia `$TACHI`。

### 建議路徑

- `POST /users/me/tachi/claim-sepolia`

### 範圍

- 驗證 spendable balance
- 取得 wallet address（依賴 Issue 3 的 linking 規格）
- DB reserve（spendable → reserved）
- 呼叫 mint service（Issue 5）
- tx confirmed → DB settle；tx failed → DB 解凍
- 回傳 claim 結果與 tx hash

### 完成條件

- API 可用
- success / insufficient balance / wallet missing / chain error 都有明確回應
- 重送相同 idempotency key 不會 double mint

---

## Issue 7: 前端最小 claim 入口

### 目標

讓使用者有地方發起 claim，但不把 `tachimint` 變成完整 wallet dApp。

### 建議做法

- 先放 dashboard 或獨立頁面
- extension 只保留 secondary 入口或暫不處理

### 範圍

- 顯示可 claim 點數
- 顯示 wallet address
- Claim 按鈕
- pending / success / error
- explorer link

### 完成條件

- 使用者能從 UI 發起 claim
- 成功後能看到 tx hash / Sepolia 結果

---

## Issue 8: E2E 驗證與文件更新

### 目標

確認整條 Sepolia MVP 流程可跑通，並把文件補齊。

### 驗證流程

1. viewer 累積 `T-Point`
2. 使用者登入並有 wallet address
3. 發起 claim
4. DB 餘額扣除正確
5. `tachi_balances` 更新正確
6. Sepolia token balance 增加正確
7. explorer 可查到 tx

### 完成條件

- 有一份手動驗證結果
- 文件更新完成

---

## 3. 建議 PR 範圍

### PR A: contracts only

- Issue 1
- Issue 2

### PR B: backend linking + idempotency schema

- Issue 3
- Issue 4

### PR C: backend mint + claim API

- Issue 5
- Issue 6

### PR D: frontend claim UI

- Issue 7

### PR E: docs / verification

- Issue 8

---

## 4. 建議不要混在同一輪的內容

- Router Contract
- 主網部署
- claim fee burn
- treasury reclaim
- 多鏈支援
- extension 內完整 wallet UX
- 大規模 dashboard redesign

---

## 5. 最短里程碑定義

如果要用一句話定義這個里程碑：

```text
使用者已能把在 tachimint 累積的 T-Point，透過平台 claim 流程，成功兌換成 Sepolia 上的 $TACHI。
```
