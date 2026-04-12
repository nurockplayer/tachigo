# Sepolia Claim / Mint MVP 計畫

> 狀態：規劃中
> 最後更新：2026-04-10
> 目標：讓使用者可以把 `T-Point` 兌換成鏈上 `$TACHI`，先跑通 **Sepolia 測試網 MVP**。
> 前置：watch-to-points 已完成、DB-only claim 已存在、Web3 nonce/signature auth 已存在、Soulbound ERC-20 合約另由合約 PR 推進中。

---

## 1. 這份計畫要解決什麼

目前專案已經有兩段基礎：

1. `tachimint` 內累積 `T-Point`
2. 後端 DB-only claim，將 `T-Point` 轉入 `tachi_balances`

但還沒有真正跑通：

- wallet 綁定 / claim 對應哪個地址
- Sepolia 合約部署
- 後端或使用者觸發鏈上 mint
- 前端 claim UI / 成功失敗回饋

這份計畫的目標不是直接做完整 Phase 2，而是先讓整條「`T-Point` → claim → Sepolia `$TACHI`」能在測試網跑通。

---

## 2. 範圍定義

### 本計畫包含

- Soulbound ERC-20 合約在 Sepolia 部署
- 決定 claim 的最短執行模型
- 後端保存 / 取得使用者 wallet address
- claim API 與 on-chain mint 串接
- 最小可用的前端 claim 入口
- 端到端驗證流程

### 本計畫不包含

- 主網部署
- Router Contract 完整設計
- 1% claim fee burn
- 7 日未 claim 回收
- 多種鏈支援
- 完整錢包資產管理頁
- 大範圍 UI redesign

---

## 3. MVP 架構決策

### 建議採用的最短路徑

先做 **Server-mediated mint on Sepolia**：

```text
viewer 累積 T-Point
  → 前端發起 claim
  → 後端檢查 spendable_balance
  → 後端扣除 DB 餘額
  → 後端呼叫已部署的 TachiToken.mint(to, amount)
  → 成功後回寫 mint result / tx hash
  → 前端顯示 Sepolia claim 成功
```

### 為什麼先不用「使用者自己呼叫合約 claim」

- 目前 repo 雖有 wallet auth 基礎，但還沒有完整 claim signature / on-chain claim contract 設計
- 若直接做 user-side contract call，會多出：
  - claim message spec
  - replay protection
  - 合約 claim 驗證邏輯
  - 前端 wallet transaction UX
- 對 Sepolia MVP 來說，這會讓 scope 膨脹太多

### 這個決策的 tradeoff

優點：

- 最快跑通
- 可以驗證 tokenomics 與鏈上資料流
- 前後端都較容易控制錯誤處理

缺點：

- 後端需要持有 deployer / owner 私鑰或 relayer
- 不是最終主網模型
- 之後升級成 Router Contract 時要再重構一次

---

## 4. 必要前置條件

### A. 合約層

- `TachiToken.sol` 合約通過 review
- Foundry 測試通過
- 決定 owner 是：
  - 暫時用 deployer wallet
  - 或 Defender Relayer / 專用 hot wallet

### B. 後端層

- claim API 行為與 DB transaction 再確認
- 能保存使用者 wallet address
- 能安全讀取 Sepolia RPC 與 signer 設定

> **決策：MVP owner 策略**
> Sepolia MVP 階段使用 deployer wallet + 環境變數管理私鑰（`SEPOLIA_SIGNER_KEY`），不引入 Defender Relayer 等外部服務。主網再評估升級策略。

### C. 前端層

- 有 wallet connect / wallet link 最小入口
- 有 claim 成功 / 失敗 UI

---

## 5. 工作拆解

## 5.1 合約與部署

### 目標

讓 `$TACHI` Soulbound ERC-20 可部署到 Sepolia，並取得正式合約地址供後端與前端使用。

### 任務

- [ ] 完成 `TachiToken` review feedback
- [ ] 補 deploy script
- [ ] 定義部署參數與 owner 策略
- [ ] 部署到 Sepolia
- [ ] 記錄 contract address、chain id、deploy tx hash
- [ ] 補一份部署操作說明文件

### 交付物

- `contracts/script/...`
- Sepolia contract address
- 部署說明文件

---

## 5.2 Wallet 綁定資料模型

### 目標

讓每個使用者有明確的 claim 目標地址。

### 重要前提：Twitch user 與 Web3 user 的 account linking

目前 repo 有兩套獨立的 user record：
- Twitch Extension JWT 路徑（`tachimint` 用戶，Twitch viewer）
- Web3 SIWE 路徑（`/auth/web3/nonce` + `/auth/web3/verify`，錢包登入）

claim 的前提是「知道哪個 Twitch viewer 對應到哪個 wallet address」，因此必須先決定 **account linking 在哪個畫面、哪個時機完成**，這不是單純查現有資料就能解決的問題。

> **決策：MVP account linking 策略**
> Sepolia MVP 建議採用最簡單的 linking 流程：使用者在 dashboard 或獨立頁面先完成 SIWE 登入，後端將 Web3 provider 與 Twitch user record 關聯。Twitch Extension 本身不做 wallet 連線流程，避免 extension 變成 dApp。

### 建議做法

優先沿用既有 Web3 auth/provider 資料，不另外發明新表；若現有 `auth_providers` 已可穩定取到 wallet address，就直接複用。

### 任務

- [ ] 決定 account linking 流程（哪個畫面 / API 觸發 Twitch user ↔ wallet address 關聯）
- [ ] 確認目前登入後如何取得使用者 wallet address
- [ ] 確認是否允許一人多地址；Sepolia MVP 建議先只取一個主地址
- [ ] 決定 claim 預設地址來源
- [ ] 若現有資料結構不足，再補最小 schema / service

### 交付物

- wallet address 來源規格
- 後端可讀取的 `claim target address`

---

## 5.3 後端 Claim → Mint 串接

### 目標

讓 claim 不只是 DB 記帳，而是會產生 Sepolia mint transaction。

### 核心決策

MVP 建議新增新的 claim 路徑，而不是直接改壞既有 DB-only claim：

- `POST /users/me/tachi/claim-sepolia`

或保留原 API，但加 mode flag。前者比較乾淨。

### 後端任務

- [ ] 新增 Sepolia mint service
- [ ] 包裝合約呼叫：`mint(address to, uint256 amount)`
- [ ] 定義 env：
  - `SEPOLIA_RPC_URL`
  - `SEPOLIA_CHAIN_ID`
  - `TACHI_TOKEN_ADDRESS`
  - signer private key 或 relayer 設定
- [ ] claim transaction 與 mint transaction 的一致性策略
- [ ] 儲存 tx hash / mint result
- [ ] 失敗回滾策略

### 一致性建議

先採三步驟保守策略：

```text
① DB: spendable_balance → reserved
   （減少 spendable，但尚未 settled；防止重複提交）
② on-chain: 送出 mint tx，等待 tx confirmed
③-A tx confirmed → DB: reserved 轉 settled，tachi_balances 更新，寫入 tx hash
③-B tx failed    → DB: reserved 解凍，回原 spendable，回報錯誤
```

> **注意**：不可先 mint 再扣 DB。若 DB 在 step ③-A 失敗，使用者鏈上已有幣但 DB 未扣帳，會造成資產與帳本不一致。reserve 機制確保「扣帳意圖」在送鏈前就已鎖定。

理由：

- 先 reserve 確保同一筆 T-Point 不會被重複 claim
- 鏈上成功後再 settle，確保帳本與鏈上狀態一致
- 失敗可安全解凍，使用者不會損失資產

### 風險

- 鏈上交易 pending 時，API timeout / retry 行為要小心
- 需要 idempotency，避免重送造成 double mint

### 建議最小保護

- [ ] claim request id / idempotency key
- [ ] `mint_jobs` 或 claim status 記錄表
- [ ] tx hash 與 userID / amount 對應

---

## 5.4 前端 Claim 入口

### 目標

讓使用者看得到「把忠誠點數兌換成平台幣」的明確入口。

### UI 原則

- `tachimint` 仍以 loyalty points panel 為主
- Claim 是 secondary action
- 若 wallet / claim 流程太複雜，可導到另一個頁面完成

### 最短 UI 任務

- [ ] 顯示目前可 claim 的 `T-Point`
- [ ] 顯示已綁定的 wallet address
- [ ] Claim 按鈕
- [ ] pending / success / error 狀態
- [ ] 成功後顯示 Sepolia tx hash 或 explorer link

### 技術選項

Option A：
- claim UI 放在 dashboard 或獨立 web 頁面

Option B：
- extension 內只放 claim 入口，點下去導到外部頁面

MVP 建議：

- **先不要把完整 wallet UX 塞進 `tachimint`**
- 先做 dashboard / 獨立頁面 claim，比較不會把 extension 膨脹成 dApp

---

## 5.5 驗證與觀測

### 必要驗證

- [ ] 用戶能累積 `T-Point`
- [ ] 用戶能綁定 / 辨識 wallet address
- [ ] claim 成功後：
  - DB 餘額扣除正確
  - `tachi_balances` 更新正確
  - Sepolia token balance 增加正確
- [ ] tx hash 可追到 explorer
- [ ] 重送 claim request 不會 double mint

### 失敗情境

- [ ] wallet address 不存在
- [ ] claim amount > spendable balance
- [ ] signer / relayer 無法送交易
- [ ] contract revert
- [ ] RPC timeout / nonce 問題

---

## 6. 建議切票順序

### Phase A: 合約可部署

1. Soulbound ERC-20 review 修正
2. Sepolia deploy script
3. 部署文件 + deploy 結果

### Phase B: 鏈下資料對齊

4. wallet address 來源確認
5. claim target address 規格
6. claim idempotency / job 設計

### Phase C: 後端串鏈

7. Sepolia mint service
8. claim-sepolia API
9. tx hash / claim status persistence

### Phase D: 前端入口

10. 最小 claim UI
11. success / error / explorer link

### Phase E: 驗證

12. E2E 手動驗證
13. 文件更新

---

## 7. 成功定義

Sepolia MVP 完成的標準：

- 使用者先在平台累積 `T-Point`
- 使用者有一個可識別的 wallet address
- 使用者觸發 claim 後
  - DB 正確扣除對應 `T-Point`
  - `tachi_balances` 正確更新
  - Sepolia 上的 `$TACHI` 成功 mint 到該地址
  - 前端可看到成功結果與 tx hash

---

## 8. 本計畫後的下一步

Sepolia MVP 跑通後，再決定是否進入真正的 Phase 2：

- Router Contract
- user-side claim signature model
- claim fee burn
- treasury / reclaim
- 主網部署

在 Sepolia MVP 前，不建議先把這些一起混進同一輪。
