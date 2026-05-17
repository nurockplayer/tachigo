---
title: Start Here
sidebar_position: 1
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

# Start Here

這條路徑讓你不用先讀完整 `docs/` 樹，也能快速知道 tachigo 怎麼運作、哪些地方最常改、哪些流程跨到 tachiya。

## First hour

| 順序 | 讀什麼 | 你要帶走的 mental model |
|---|---|---|
| 1 | [Dev Portal Home](/tachigo/) | tachigo API 是核心 boundary；extension、dashboard、tachiya 都圍繞它協作。 |
| 2 | [系統整體架構](/tachigo/architecture) | client layer、Go backend、PostgreSQL、tachiya / Saleor、chain 的角色分工。 |
| 3 | [Domain Maps](/tachigo/dev-portal/domain-maps) | 先掌握 Points、Auth、Extension 三個 P0 domain。 |
| 4 | [Cross-Repo Flows](/tachigo/dev-portal/flows) | 看 watch flow 與 coupon redemption flow 如何跨 repo。 |

## First day

| 任務 | 路徑 | 檢查點 |
|---|---|---|
| 跑起本機服務 | `make dev` | Docker compose 會啟動 API 與前端服務。 |
| 看 API 入口 | [`services/api/internal/router/router.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/router/router.go) | 確認 public、auth、internal route 怎麼分層。 |
| 看 extension heartbeat | [`apps/extension/src/hooks/useHeartbeat.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useHeartbeat.ts) | viewer activity 如何回報到 API。 |
| 看 dashboard 入口 | [`apps/dashboard/src/App.tsx`](https://github.com/nurockplayer/tachigo/blob/develop/apps/dashboard/src/App.tsx) | Refine / React Router resource 如何對上 API。 |
| 看 tachiya 邊界 | [Coupon redemption flow](/tachigo/dev-portal/flows#coupon-redemption-flow) | tachigo 扣點，tachiya 保護 Saleor commerce logic。 |

## First PR

1. 先找 issue，確認 scope 只做 source of truth 明確要求的事情。
2. 從 [Daily Dev Guide](/tachigo/dev-portal/daily-dev-guide) 找對應 domain 的 source、tests、policy。
3. 若改 API handler，確認 Swagger / shared types / frontend assumptions 是否需要一起更新。
4. 若改 package dependency，讀 [Supply-chain Security Guardrails](/tachigo/ai/supply-chain-security)，PR body 必須說明套件、版本、用途與 guardrail。
5. 開 PR 前跑最小可驗證命令，並在 PR template 的 Acceptance Criteria / 測試方式寫清楚。

## 常見方向

| 我要做什麼 | 從這裡開始 |
|---|---|
| 改觀看累點、ledger、餘額 | [Points / ledger / watch time](/tachigo/dev-portal/domain-maps#points--ledger--watch-time) |
| 改 login、JWT、provider、role | [Auth / identity](/tachigo/dev-portal/domain-maps#auth--identity) |
| 改 sidepanel UI 或 heartbeat | [Extension / sidepanel](/tachigo/dev-portal/domain-maps#extension--sidepanel) |
| 查跨 tachiya 的折扣碼流程 | [Coupon redemption flow](/tachigo/dev-portal/flows#coupon-redemption-flow) |
| 找文件或 plan 的位置 | [Source Index](/tachigo/dev-portal/source-index) |
