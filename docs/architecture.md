# 系統整體架構

```
┌─────────────────────────────────────────────────────────────────────┐
│                            用戶端                                    │
│                                                                      │
│  ┌─────────────────┐   ┌──────────────────┐   ┌─────────────────┐  │
│  │ Twitch Extension│   │   後台管理 (TBD)  │   │  觀眾錢包       │  │
│  │   (tachimint)   │   │ 經紀公司/實況主   │   │  (Phase 2)      │  │
│  │ React+TypeScript│   │                  │   │  Claim 上鏈     │  │
│  └────────┬────────┘   └────────┬─────────┘   └───────┬─────────┘  │
└───────────┼─────────────────────┼─────────────────────┼────────────┘
            │ Bits + JWT          │ 管理設定             │ Claim
            ▼                     ▼                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        後端 (Go + Gin)                               │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────┐  │
│  │ Auth Service │  │  Extension   │  │  Points    │  │ Agency   │  │
│  │ ✅ 已完成    │  │  Service     │  │  Service   │  │ Service  │  │
│  │ Twitch/Google│  │  ✅ 已完成   │  │  🟡 TBD    │  │ 🟡 TBD  │  │
│  │ Web3/SIWE    │  │  Bits驗證    │  │  雙帳本    │  │ 經紀公司 │  │
│  │ Email        │  │  JWT驗證     │  │            │  │ 實況主   │  │
│  └──────┬───────┘  └──────┬───────┘  └─────┬──────┘  └────┬─────┘  │
│         │                 │                 │               │        │
│  ┌──────┴────────────────────────────────────────────────────────┐  │
│  │              Sink Services (🟡 全部 TBD)                      │  │
│  │   Store/Saleor (#15)    Voting/Gambling (#17)    Avatar       │  │
│  └───────────────────────────────────────────────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
            ┌──────────────────┼──────────────────┐
            ▼                  ▼                   ▼
┌───────────────────┐  ┌───────────────┐  ┌──────────────────────────┐
│   PostgreSQL      │  │ Sepolia (TBD) │  │   Token Sink 邏輯        │
│                   │  │               │  │                          │
│ ✅ users          │  │ Factory       │  │ 商城折扣                 │
│ ✅ auth_providers │  │  └─ Agency    │  │  實況主代幣折抵          │
│ ✅ refresh_tokens │  │     Token ×N  │  │                          │
│ 🟡 points_ledger  │  │ Soulbound     │  │ 虛擬換裝                 │
│    cumulative     │  │ ERC-20        │  │  平台幣消耗              │
│    spendable      │  │ (Foundry +    │  │                          │
│ 🟡 agencies       │  │  OpenZeppelin)│  │ 投票 / 賭博 (#17)        │
│ 🟡 transactions   │  │               │  │  鏈下扣餘額              │
└───────────────────┘  └───────────────┘  │                          │
                                           │ 私人直播票 (Phase 2+)    │
                                           └──────────────────────────┘

✅ = 已完成    🟡 = MVP 待實作    灰色 = Phase 2+
```

## 主要資料流

```
觀眾花 Bits
  → Extension 驗證 receipt
  → points_ledger: spendable_balance ↑  cumulative_total ↑

觀眾用 Token Sink
  → points_ledger: spendable_balance ↓  cumulative_total 不動

Phase 2: 觀眾 Claim
  → 鏈下餘額 → Soulbound ERC-20 mint
```

## 相關 Issues

- [#12](https://github.com/nurockplayer/tachigo/issues/12) Token 系統架構（鏈下記帳 vs 直接 mint）
- [#13](https://github.com/nurockplayer/tachigo/issues/13) 完整 Bits transaction 流程
- [#15](https://github.com/nurockplayer/tachigo/issues/15) 商城與 Token 消費機制
- [#17](https://github.com/nurockplayer/tachigo/issues/17) Token 經濟設計與 Soulbound 衝突
- [#18](https://github.com/nurockplayer/tachigo/issues/18) 獨立後台系統
