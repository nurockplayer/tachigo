---
title: 領域地圖
sidebar_position: 2
status: active
owner: engineering
last_reviewed: 2026-05-13
source_of_truth: true
code_areas:
  - services/api
  - apps/extension
  - apps/dashboard
related_repos:
  - tachigo
  - tachiya
---

# 領域地圖

Domain Maps 是「我要改某個功能」之前的索引。P0 domains 已整理主要責任、資料流、source、tests 與風險；P1 domains 先保留入口。

## P0 領域

### Points / ledger / watch time

<span className="tachigo-status">P0 complete</span>

| 面向 | 入口 |
|---|---|
| 做什麼 | 把 Twitch viewer activity 轉成可花用 points，維護 watch sessions、points transactions、balances 與 spend/claim 邊界。 |
| 主要資料流 | extension heartbeat → watch service → points service → PostgreSQL ledger / balance。 |
| API handlers | [`watch_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/watch_handler.go), [`points_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/points_handler.go), [`spend_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/spend_handler.go), [`internal_points_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/internal_points_handler.go) |
| Services | [`watch_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/watch_service.go), [`points_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/points_service.go), [`spend_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/spend_service.go), [`tachiya_client.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/tachiya_client.go) |
| Models | [`points.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/points.go), [`watch_session.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/watch_session.go), [`watch_stats.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/watch_stats.go), [`tachi_balance.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/tachi_balance.go), [`coupon_redemption.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/coupon_redemption.go) |
| Tests | [`watch_service_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/watch_service_test.go), [`points_service_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/points_service_test.go), [`spend_service_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/spend_service_test.go), [`tachiya_client_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/tachiya_client_test.go) |
| Related docs | [Watch-to-points design](/tachigo/watch-to-points-design), [Sequence diagram](/tachigo/sequence-diagram), [Tokenomics](/tachigo/tokenomics) |

踩雷點：

- Ledger 必須能解釋每次加扣點，避免只改 balance 忘記 transaction。
- Coupon redemption 會跨 tachiya / Saleor，扣點與 external redemption id 要避免 double spend。
- 改 migration 時要檢查 backfill、NOT NULL default、rollback 與 deployment order。

### Auth / identity

<span className="tachigo-status">P0 complete</span>

| 面向 | 入口 |
|---|---|
| 做什麼 | 管 Twitch / Google / email / SIWE 身份、refresh token、JWT、role / permission 與 protected API。 |
| 主要資料流 | provider credential → auth service → user / auth provider / refresh token → middleware → domain route。 |
| API handlers | [`auth_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/auth_handler.go), [`email_auth_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/email_auth_handler.go), [`user_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/user_handler.go), [`address_handler.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/address_handler.go) |
| Services | [`auth_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/auth_service.go), [`email_auth_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/email_auth_service.go), [`user_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/user_service.go), [`siwe.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/siwe.go), [`oauth_token_crypto.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/oauth_token_crypto.go) |
| Middleware | [`auth.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/middleware/auth.go), [`internal_auth.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/middleware/internal_auth.go), [`rate_limit.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/middleware/rate_limit.go) |
| Models | [`user.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/user.go), [`auth_provider.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/auth_provider.go), [`refresh_token.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/refresh_token.go), [`email_auth.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/models/email_auth.go) |
| Tests | [`auth_service_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/auth_service_test.go), [`auth_handler_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/auth_handler_test.go), [`auth_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/middleware/auth_test.go), [`rbac_handler_test.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/handlers/rbac_handler_test.go) |
| Related docs | [Auth architecture](/tachigo/auth-architecture), [Backend permissions](/tachigo/backend-permissions) |

踩雷點：

- Auth 變更通常會影響 extension、dashboard 與 internal handlers，要檢查 token propagation。
- OAuth token / refresh token 不可出現在 log、URL query 或 PR artifacts。
- SIWE / wallet signature 需要驗證 nonce、expiration、chain id 與 signer ownership。

### Extension / sidepanel

<span className="tachigo-status">P0 complete</span>

| 面向 | 入口 |
|---|---|
| 做什麼 | Chrome sidepanel / Twitch extension runtime，處理登入、心跳、點數顯示、coupon shop、claim / raffle panels。 |
| 主要資料流 | Twitch context / extension storage → React hooks → API service → tachigo API。 |
| App entry | [`main.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/main.tsx), [`app/App.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/app/App.tsx), [`TwitchApp.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/TwitchApp.tsx) |
| Hooks | [`useHeartbeat.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useHeartbeat.ts), [`useTPoint.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useTPoint.ts), [`useTwitch.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useTwitch.ts), [`useRaffleResult.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useRaffleResult.ts) |
| API client | [`services/api.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/services/api.ts), [`couponRedeem.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/app/couponRedeem.ts), [`extension/storage.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/extension/storage.ts) |
| UI panels | [`CouponShopPanel.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/app/components/CouponShopPanel.tsx), [`ClaimPanel.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/app/components/ClaimPanel.tsx), [`RaffleResultPanel.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/app/components/RaffleResultPanel.tsx) |
| Tests | [`useTPoint.test.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useTPoint.test.tsx), [`api.test.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/services/api.test.ts), [`runtime-config.test.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/extension/runtime-config.test.ts), [`storage.test.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/extension/storage.test.ts) |
| Related docs | [Chrome sidepanel migration](/tachigo/history/2026-04-16-tachimint-chrome-sidepanel-migration), [Extension UI prompts](/tachigo/extension-ui-prompts) |

踩雷點：

- Heartbeat 可能遇到 stale closure、重複送出或 viewer context 尚未 ready。
- Extension runtime config 與 API base URL 不要寫死到 production-only 假設。
- i18n 文字需同步 `en`、`zh-TW`、`zh-CN`，避免 UI key 缺漏。

## P1 領域

| 領域 | 狀態 | 入口 |
|---|---|---|
| Raffle / airdrop | _Coming soon_ | [`raffle_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/raffle_service.go), [`airdrop_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/airdrop_service.go) |
| Claim / spend / coupon redemption | _Coming soon_ | [`claim_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/claim_service.go), [`spend_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/spend_service.go) |
| Dashboard | _Coming soon_ | [`apps/dashboard/src/App.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/dashboard/src/App.tsx), [`dataProvider.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/dashboard/src/providers/dataProvider.ts) |
| Tachiya commerce integration | _Coming soon_ | [`tachiya_client.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/tachiya_client.go), [`tachiya`](https://github.com/nurockplayer/tachiya) |
| AI workflow / PR scope policy | _Coming soon_ | [Codex autonomous workflow](/tachigo/ai/codex-autonomous-workflow), [PR scope policy](/tachigo/pr-scope-policy) |
