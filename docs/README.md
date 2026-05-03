# Tachigo Docs

本目錄收納工程文件。文件分成兩種定位：

- **現況 source of truth**：描述目前仍有效的架構、政策、規格。檔名不加日期，內容改變時直接更新。
- **歷史紀錄**：像 SQL migration log 一樣，記錄某次遷移、盤點或決策當下的背景與結果。檔名使用日期開頭，完成後原則上只做狀態校正，不作為 active TODO。

## 目錄分類

| 位置 | 用途 | 命名 |
|---|---|---|
| `docs/` | 目前仍有效的架構、API、政策與設計文件 | `<topic>.md` |
| `docs/history/` | 已完成的一次性遷移、盤點、整理紀錄 | `YYYY-MM-DD-<topic>.md` |
| `docs/ai/` | 既有 AI 協作指南與 agent-facing 文件 | `<topic>.md` |
| `docs/superpowers/` | 既有 Superpowers specs / plans 歷史文件 | 維持既有日期命名 |
| `plans/` | 實作前計畫與完成後的執行紀錄 | `<feature-slug>.md` |

## 命名原則

- 目前仍是 source of truth 的文件不加日期，例如 `architecture.md`、`tokenomics.md`。
- 歷史紀錄與決策紀錄使用日期開頭，例如 `2026-04-30-monorepo-directory-refactor.md`。
- 日期使用該事件完成、merge 或正式採納的日期；若只有月份，使用最接近的完成日期並在文件內說明。
- 已完成但仍保留的歷史文件，文件頂端應標註狀態與最後校正日期。
- 本規範先定義新增分類與命名原則，不要求批次搬移既有目錄；未列出的既有目錄應維持現況，待後續 PR 另行收斂。
