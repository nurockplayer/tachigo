---
title: Daily Dev Guide
sidebar_position: 3
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

# Daily Dev Guide

這頁回答「我要改 X，從哪裡開始？」每一列都是可以直接進入 source 的日常路徑。

## Change map

| 我要改 | 先看 | 常跑驗證 | 注意 |
|---|---|---|---|
| Watch / points award | [`watch_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/watch_service.go), [`points_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/points_service.go) | `docker compose run --no-deps --rm app go test ./internal/services ./internal/handlers` | double credit、ledger / balance consistency、time window。 |
| Extension heartbeat | [`useHeartbeat.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/hooks/useHeartbeat.ts), [`api.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/extension/src/services/api.ts) | `pnpm --filter ./apps/extension test` | stale state、runtime config、API shape。 |
| Login / token | [`auth_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/auth_service.go), [`auth.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/middleware/auth.go) | `docker compose run --no-deps --rm app go test ./internal/services ./internal/middleware ./internal/handlers` | token leak、refresh rotation、role guard。 |
| Dashboard page | [`apps/dashboard/src/pages`](https://github.com/nurockplayer/tachigo/tree/develop/apps/dashboard/src/pages), [`dataProvider.ts`](https://github.com/nurockplayer/tachigo/blob/develop/apps/dashboard/src/providers/dataProvider.ts) | `pnpm --filter ./apps/dashboard test && pnpm --filter ./apps/dashboard build` | loading / error state、API contract、auth redirect。 |
| Coupon redemption | [`spend_service.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/spend_service.go), [`tachiya_client.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/services/tachiya_client.go) | `docker compose run --no-deps --rm app go test ./internal/services ./internal/handlers` | idempotency、timeout、partial failure、tachiya contract。 |
| Docs portal | [`docs/dev-portal`](https://github.com/nurockplayer/tachigo/tree/develop/docs/dev-portal), [`apps/docs`](https://github.com/nurockplayer/tachigo/tree/develop/apps/docs) | `pnpm build:docs` | broken links、stale source paths、scope creep。 |

## PR scope routine

1. 找到 source issue，確認 Acceptance Criteria。
2. 用 [Domain Maps](/tachigo/dev-portal/domain-maps) 找 P0 / P1 domain 邊界。
3. 若改動跨 backend + frontend、或 diff 可能超過 400 行，先拆 PR。
4. 若碰到 migration、auth、wallet signature、points ledger、金流或權限模型，PR body 要寫風險與驗證。
5. 若新增 dependency，必須說明套件名稱、版本、用途、lifecycle script、lockfile 變更與 guardrail 結果。

## Useful commands

| 目的 | 指令 |
|---|---|
| 啟動全部服務 | `make dev` |
| 停止服務 | `make down` |
| 後端測試 | `docker compose run --no-deps --rm app go test ./...` |
| Extension build | `pnpm --filter ./apps/extension build` |
| Dashboard build | `pnpm --filter ./apps/dashboard build` |
| Docs build | `pnpm build:docs` |
| Supply-chain guardrail | `make supply-chain-check` |

## Review heuristics

| Surface | 快速檢查 |
|---|---|
| API handler | request validation、auth middleware、Swagger / shared type 是否需同步。 |
| Service | transaction boundary、context cancellation、idempotency、nil error handling。 |
| Frontend | loading / error / empty state、double submit、auth token 是否外洩。 |
| Docs | 是否指回 source、是否把 proposal 寫成已完成事實。 |
