# 系統整體架構

```
┌────────────────────────────────────────────────────────────────────┐
│                           CLIENT LAYER                             │
│                                                                    │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ Chrome Sidepanel │  │  Dashboard [MVP] │  │  Wallet [Phs.2]  │  │
│  │   (tachimint)    │  │ React+Vite+Refine│  │  Claim on-chain  │  │
│  │ React+TypeScript │  │ Agency/Strm Mgmt │  │                  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘  │
└───────────┼─────────────────────┼─────────────────────┼────────────┘
            │ Heartbeat + JWT     │ Admin API           │ Claim
            v                     v                     v
┌──────────────────────────────────────────────────────────────────────┐
│                        BACKEND  (Go + Gin)                           │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────┐   │
│  │ AuthService  │  │  Extension   │  │   Points     │  │ Agency  │   │
│  │ [done]       │  │  Service     │  │   Service    │  │ Service │   │
│  │ Twitch/Google│  │  [done]      │  │   [TBD]      │  │ [TBD]   │   │
│  │ Web3/SIWE    │  │  Watch verify│  │   dual ledger│  │ agency  │   │
│  │ Email        │  │  JWT verify  │  │              │  │ stream  │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └────┬────┘   │
│         │                 │                 │               │        │
│  ┌──────┴─────────────────┴─────────────────┴───────────────┴────┐   │
│  │                    Sink Services  [TBD]                       │.  │
│  │          Store/Saleor (#15)   Gambling (#17)   Avatar         │   │
│  └───────────────────────────────────────────────────────────────┘.  │
└────────────────────────────────┬─────────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              v                  v                   v
┌──────────────────────┐  ┌──────────────┐  ┌────────────────────────┐
│      PostgreSQL      │  │   Sepolia    │  │    Token Sink Logic    │
│                      │  │   [TBD]      │  │                        │
│  [done] users        │  │              │  │  Store discount        │
│  [done] auth_provid. │  │  Factory     │  │    token deduction     │
│  [done] refresh_tok. │  │   └─ Agency  │  │                        │
│  [TBD]  points_ledger│  │      Token×N │  │  Avatar customization  │
│           cumulative │  │  Soulbound   │  │    platform token burn │
│           spendable  │  │  ERC-20      │  │                        │
│  [TBD]  agencies     │  │  Foundry +   │  │  Voting/Gambling (#17) │
│  [TBD]  transactions │  │  OpenZeppelin│  │    off-chain balance   │
└──────────────────────┘  └──────────────┘  │                        │
                                            │  Private stream [Phs2] │
                                            └────────────────────────┘

[done] = 已完成    [MVP] = MVP 已進入實作    [TBD] = MVP 待實作    [Phs.2] = Phase 2+
```

## 主要資料流

> 補充：`tachimint` 的前端 runtime 方向已定為 Chrome sidepanel extension；本階段 viewer identity 與既有 API contract 仍沿用 Twitch / extension auth 流程。詳見 [docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md](history/2026-04-16-tachimint-chrome-sidepanel-migration.md)。
> Dashboard frontend 已在 `apps/dashboard/` 以 React + Vite + Refine 進入 MVP 實作；上圖後端 `Agency Service [TBD]` 仍表示 agency/management backend maturity，而不是 dashboard app 不存在。

```
觀眾觀看直播（定時 Heartbeat）
  → Extension 回報在線狀態
  → points_ledger: spendable_balance ↑  cumulative_total ↑

觀眾用 Token Sink
  → points_ledger: spendable_balance ↓  cumulative_total 不動

Phase 2: 觀眾 Claim
  → 鏈下餘額 → Soulbound ERC-20 mint
```

## 與 Tachiya 的串接

### 串接流程

```
Twitch 觀眾
  → tachigo extension（累積 token）
  → tachigo go backend（驗證 token）
  → tachiya FastAPI（銷毀 token，產生折扣碼）
  → Saleor（套用折扣碼結帳）
```

### 三服務拆法（決策）

Go（tachigo）、FastAPI（tachiya api/）、Saleor 三者維持獨立，不整合。

| 服務 | 職責 |
|------|------|
| **Go（tachigo）** | Twitch 身份、忠誠點數、token 發放——自建會員系統 |
| **FastAPI（tachiya api/）** | Saleor 的自訂邏輯出口（折扣計算、分潤、webhook 處理） |
| **Saleor** | 電商核心（購物車、訂單、結帳） |

**會員系統不衝突**：Saleor Account 只管「能結帳的帳號」，Go 會員系統管忠誠點數與 Twitch 身份，兩者用 Saleor customer ID 關聯。

**為什麼不把 FastAPI 邏輯併入 Go**：FastAPI 是保護層，沒有它未來要自訂 Saleor 邏輯只能 fork。等 FastAPI 真的只剩一兩支 API 時再評估是否併入。

---

## 相關 Issues

- [#12](https://github.com/nurockplayer/tachigo/issues/12) Token 系統架構（鏈下記帳 vs 直接 mint）
- [#13](https://github.com/nurockplayer/tachigo/issues/13) 完整 Bits transaction 流程
- [#15](https://github.com/nurockplayer/tachigo/issues/15) 商城與 Token 消費機制
- [#17](https://github.com/nurockplayer/tachigo/issues/17) Token 經濟設計與 Soulbound 衝突
- [#18](https://github.com/nurockplayer/tachigo/issues/18) 獨立後台系統
