# 系統整體架構

```
┌────────────────────────────────────────────────────────────────────┐
│                           CLIENT LAYER                             │
│                                                                    │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ Twitch Extension │  │  Dashboard [TBD] │  │  Wallet [Phs.2]  │  │
│  │   (tachimint)    │  │  Agency/Streamer │  │  Claim on-chain  │  │
│  │ React+TypeScript │  │  Mgmt Interface  │  │                  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘  │
└───────────┼─────────────────────┼─────────────────────┼────────────┘
            │ Bits + JWT          │ Admin API           │ Claim
            v                     v                     v
┌──────────────────────────────────────────────────────────────────────┐
│                        BACKEND  (Go + Gin)                           │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────┐   │
│  │ AuthService  │  │  Extension   │  │   Points     │  │ Agency  │   │
│  │ [done]       │  │  Service     │  │   Service    │  │ Service │   │
│  │ Twitch/Google│  │  [done]      │  │   [TBD]      │  │ [TBD]   │   │
│  │ Web3/SIWE    │  │  Bits verify │  │   dual ledger│  │ agency  │   │
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

[done] = 已完成    [TBD] = MVP 待實作    [Phs.2] = Phase 2+
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
