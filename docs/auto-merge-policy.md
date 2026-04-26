# Auto-merge 與 Approve 語義

**決定日期**：2026-04-26
**Discussion**：https://github.com/nurockplayer/tachigo/discussions/359

## 決策摘要

開啟 GitHub auto-merge，設定 required 1 approving review。Approve = 授權 merge。

## 問題背景

Scope Police 強制分拆 PR，分拆後的 PR merge 回 develop 會造成其他 open PR 需要 rebase（merge cascade）。根本原因是 PR 在 approved 狀態停留太久——沒有人負責去按 merge。

## 設定

| 項目 | 值 |
|---|---|
| `allow_auto_merge` | `true` |
| `required_approving_review_count` | `1` |

## Approve 的意義

按 approve = 「這個 PR 現在可以進 develop」。不存在「approve 但還不 merge」的中間狀態。

Approve 前應確認：CI 全過、scope 正確、無 blocker。

## Review 動作對應

| 嚴重程度 | 動作 |
|---|---|
| blocker | Request changes |
| major | Approve + Comment 說明風險 |
| minor / nit | Approve + Comment（可選） |

minor / nit 不用 Request changes——在 auto-merge 下這會卡住整個 pipeline，成本不相稱。改用 Comment，作者自行決定是否修或開 follow-up。

## 詳細討論

完整的方案比較、取捨分析、以及未來改回的條件，見 Discussion：
https://github.com/nurockplayer/tachigo/discussions/359
