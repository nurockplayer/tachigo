---
title: 來源索引
sidebar_position: 5
status: active
owner: engineering
last_reviewed: 2026-05-13
source_of_truth: true
code_areas:
  - docs
  - services/api
  - apps/extension
  - apps/dashboard
related_repos:
  - tachigo
  - tachiya
---

# 來源索引

這頁承接原本 `docs/README.md` 的 taxonomy：Markdown 仍是 source of truth，Dev Portal 只把入口整理得更容易掃描。

## 目錄分類

| 位置 | 用途 | 命名 |
|---|---|---|
| `docs/` | 目前仍有效的架構、API、政策、設計文件，或仍在進行中的 plan/proposal | `<topic>.md` |
| `docs/dev-portal/` | Dev Portal 導覽頁：onboarding、domain map、flow、source index、graph explorer | `<topic>.md` |
| `docs/history/` | 已完成的一次性遷移、盤點、整理或決策紀錄 | `YYYY-MM-DD-<topic>.md` |
| `docs/ai/` | 既有 AI 協作指南與 agent-facing 文件 | `<topic>.md` |
| `docs/superpowers/specs/` | 已確認或待確認的設計規格；proposal 不得自動視為 implementation source of truth | `YYYY-MM-DD-<topic>-design.md` |
| `docs/superpowers/plans/` | 實作前計畫與完成後的執行紀錄 | `YYYY-MM-DD-<feature>.md` |
| `plans/` | repo 既有工作計畫或執行紀錄 | `<feature-slug>.md` |

## 核心 Source of Truth

| 文件 | 定位 |
|---|---|
| [architecture.md](/tachigo/architecture) | 系統整體架構與主要資料流 |
| [auth-architecture.md](/tachigo/auth-architecture) | Auth 現況與 migration guardrails |
| [backend-permissions.md](/tachigo/backend-permissions) | Backend role / permission 現況與變更 guardrails |
| [auto-merge-policy.md](/tachigo/auto-merge-policy) | Auto-merge 與 approve 語義 |
| [dependabot-update-policy.md](/tachigo/dependabot-update-policy) | Dependabot 更新政策 |
| [draft-pr-auto-ready.md](/tachigo/draft-pr-auto-ready) | Draft PR auto-ready 現行流程 |
| [pr-scope-policy.md](/tachigo/pr-scope-policy) | PR scope 與 required checks policy |
| [sequence-diagram.md](/tachigo/sequence-diagram) | Watch / points / dashboard 主要時序圖 |
| [tokenomics.md](/tachigo/tokenomics) | `$TACHI` 平台幣經濟模型 |
| [uuid-v7.md](/tachigo/uuid-v7) | UUID 版本策略 |
| [watch-to-points-design.md](/tachigo/watch-to-points-design) | Watch-to-points 已完成設計 |

## 進行中計畫與提案

| 文件 | 定位 |
|---|---|
| [atlas-migration-plan.md](/tachigo/atlas-migration-plan) | #463 Atlas migration 拆分實作計畫；不得單獨視為已完成狀態 |
| [atlas-schema-reconciliation.md](/tachigo/atlas-schema-reconciliation) | #463 baseline 前 schema reconciliation evidence / procedure |
| [non-web3-launch-readiness.md](/tachigo/non-web3-launch-readiness) | 暫時捨棄 Web3 上鏈部分後的上線距離、Codex 自動化能力與真人介入層級 snapshot |
| [openapi-codegen-flow.md](/tachigo/openapi-codegen-flow) | #401 OpenAPI → TypeScript contracts / codegen rollout 計畫 |
| [tachigo Dev Portal spec](/tachigo/superpowers/specs/2026-05-14-project-atlas-design) | #674 Dev Portal 設計規格；命名已避開 atlasgo tooling 混淆 |

## 參考與討論文件

| 文件 | 定位 |
|---|---|
| [extension-ui-prompts.md](/tachigo/extension-ui-prompts) | Tachimint UI prompt reference，不是產品 runtime source of truth |
| [feature-discussion.md](/tachigo/feature-discussion) | 早期產品討論紀錄；仍保留供背景參考，不得單獨視為 implementation source of truth |
| [tachimint-loyalty-claim-boundary.md](/tachigo/tachimint-loyalty-claim-boundary) | Tachimint / claim flow 邊界討論短文件，不是正式 architecture source of truth |

## 開發者入口運維

| 文件 | 定位 |
|---|---|
| [deployment.md](/tachigo/dev-portal/deployment) | Cloudflare Pages 手動連 repo、branch deploy / PR preview、公開 URL 驗證與 rollback/readback checklist |

## 程式碼來源對照

| 區域 | 原始碼 |
|---|---|
| API bootstrap / router | [`cmd/server`](https://github.com/nurockplayer/tachigo/tree/develop/services/api/cmd/server), [`internal/router/router.go`](https://github.com/nurockplayer/tachigo/blob/develop/services/api/internal/router/router.go) |
| API handlers | [`services/api/internal/handlers`](https://github.com/nurockplayer/tachigo/tree/develop/services/api/internal/handlers) |
| API services | [`services/api/internal/services`](https://github.com/nurockplayer/tachigo/tree/develop/services/api/internal/services) |
| API models / migrations | [`services/api/internal/models`](https://github.com/nurockplayer/tachigo/tree/develop/services/api/internal/models), [`services/api/migrations`](https://github.com/nurockplayer/tachigo/tree/develop/services/api/migrations) |
| Extension app | [`apps/extension/src`](https://github.com/nurockplayer/tachigo/tree/develop/apps/extension/src) |
| Dashboard app | [`apps/dashboard/src`](https://github.com/nurockplayer/tachigo/tree/develop/apps/dashboard/src) |
| Shared generated types | [`packages/shared-types`](https://github.com/nurockplayer/tachigo/tree/develop/packages/shared-types) |
| API client package | [`packages/api-client`](https://github.com/nurockplayer/tachigo/tree/develop/packages/api-client) |
| Docs portal | [`apps/docs`](https://github.com/nurockplayer/tachigo/tree/develop/apps/docs), [`docs/dev-portal`](https://github.com/nurockplayer/tachigo/tree/develop/docs/dev-portal) |

## 命名規則

- 目前仍是 source of truth 的文件不加日期，例如 `architecture.md`、`tokenomics.md`。
- 歷史紀錄與決策紀錄使用日期開頭，例如 `2026-04-30-monorepo-directory-refactor.md`。
- Proposal 不能在未採納前寫成完成狀態；Dev Portal 只連到它，並標註狀態。
- 可見名稱、sidebar label、slug 與目錄使用 `tachigo Dev Portal` / `Dev Portal`，不要把導覽網站命名成 Atlas，以免和 atlasgo migration tooling 混淆。
