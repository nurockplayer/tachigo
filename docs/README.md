# Tachigo Docs

本目錄收納工程文件。root 只保留仍需要被日常引用的現況文件、active plan/proposal，或暫時保留待後續分類的討論參考；已完成的一次性盤點、遷移或決策紀錄放在 `docs/history/`。

## 目錄分類

| 位置 | 用途 | 命名 |
|---|---|---|
| `docs/` | 目前仍有效的架構、API、政策、設計文件，或仍在進行中的 plan/proposal | `<topic>.md` |
| `docs/history/` | 已完成的一次性遷移、盤點、整理或決策紀錄 | `YYYY-MM-DD-<topic>.md` |
| `docs/ai/` | 既有 AI 協作指南與 agent-facing 文件 | `<topic>.md` |
| `docs/superpowers/` | 既有 Superpowers specs / plans 歷史文件 | 維持既有日期命名 |
| `plans/` | 實作前計畫與完成後的執行紀錄 | `<feature-slug>.md` |

## Root Source Of Truth

| 文件 | 定位 |
|---|---|
| [`architecture.md`](architecture.md) | 系統整體架構與主要資料流 |
| [`auth-architecture.md`](auth-architecture.md) | Auth 現況與 migration guardrails |
| [`backend-permissions.md`](backend-permissions.md) | Backend role / permission 現況與變更 guardrails |
| [`auto-merge-policy.md`](auto-merge-policy.md) | Auto-merge 與 approve 語義 |
| [`dependabot-update-policy.md`](dependabot-update-policy.md) | Dependabot 更新政策 |
| [`draft-pr-auto-ready.md`](draft-pr-auto-ready.md) | Draft PR auto-ready 現行流程 |
| [`pr-scope-policy.md`](pr-scope-policy.md) | PR scope 與 required checks policy |
| [`sequence-diagram.md`](sequence-diagram.md) | Watch / points / dashboard 主要時序圖 |
| [`tokenomics.md`](tokenomics.md) | `$TACHI` 平台幣經濟模型 |
| [`uuid-v7.md`](uuid-v7.md) | UUID 版本策略 |
| [`watch-to-points-design.md`](watch-to-points-design.md) | Watch-to-points 已完成設計 |

## Active Plans / Proposals

| 文件 | 定位 |
|---|---|
| [`atlas-migration-plan.md`](atlas-migration-plan.md) | #463 Atlas migration 拆分實作計畫；不得單獨視為已完成狀態 |
| [`atlas-schema-reconciliation.md`](atlas-schema-reconciliation.md) | #463 baseline 前 schema reconciliation evidence / procedure |
| [`non-web3-launch-readiness.md`](non-web3-launch-readiness.md) | 暫時捨棄 Web3 上鏈部分後的上線距離、Codex 自動化能力與真人介入層級 snapshot |

## Root Reference / Discussion Notes

| 文件 | 定位 |
|---|---|
| [`extension-ui-prompts.md`](extension-ui-prompts.md) | Tachimint UI prompt reference，不是產品 runtime source of truth |
| [`feature-discussion.md`](feature-discussion.md) | 早期產品討論紀錄；仍保留供背景參考，不得單獨視為 implementation source of truth |
| [`tachimint-loyalty-claim-boundary.md`](tachimint-loyalty-claim-boundary.md) | Tachimint / claim flow 邊界討論短文件，不是正式 architecture source of truth |

## Historical Records

| 文件 | 定位 |
|---|---|
| [`history/2026-04-16-chrome-extension-terminology-audit.md`](history/2026-04-16-chrome-extension-terminology-audit.md) | Chrome / Twitch / extension terminology 盤點 |
| [`history/2026-04-16-tachimint-chrome-sidepanel-migration.md`](history/2026-04-16-tachimint-chrome-sidepanel-migration.md) | Tachimint Chrome sidepanel migration decision record |
| [`history/2026-04-18-git-lfs-assets.md`](history/2026-04-18-git-lfs-assets.md) | Git LFS asset handling 決策與操作紀錄 |
| [`history/2026-04-30-monorepo-directory-refactor.md`](history/2026-04-30-monorepo-directory-refactor.md) | Monorepo directory refactor 歷史紀錄 |
| [`history/2026-05-01-dashboard-stack-evaluation.md`](history/2026-05-01-dashboard-stack-evaluation.md) | Dashboard Refine.dev 技術選型決策紀錄 |

## 命名原則

- 目前仍是 source of truth 的文件不加日期，例如 `architecture.md`、`tokenomics.md`。
- 歷史紀錄與決策紀錄使用日期開頭，例如 `2026-04-30-monorepo-directory-refactor.md`。
- 日期使用該事件完成、merge 或正式採納的日期；若只有月份，使用最接近的完成日期並在文件內說明。
- 已完成但仍保留的歷史文件，文件頂端應標註狀態與最後校正日期。
- 狀態不明文件不得在盤點 PR 中硬搬或硬刪；只能補狀態標記，或另開 issue 做 source-of-truth 判定。
