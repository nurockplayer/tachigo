---
title: 圖譜探索器
sidebar_position: 6
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

# 圖譜探索器

Graphify 的知識圖譜適合當成「影響範圍雷達」：快速看哪些檔案、domain、docs 可能互相關聯，再回到人工整理過的 source map 驗證。

## 現階段用途

| 用途 | 怎麼判讀 |
|---|---|
| Onboarding | 先看 cluster 名稱和 hub nodes，建立 repo 的粗略地圖。 |
| Impact analysis | 改某個 service 前，找附近 handlers、models、tests、docs。 |
| Docs cleanup | 找到孤立文件、命名不一致或過時 cluster。 |
| Review prep | 大 PR review 前，用圖譜找可能漏看的入口。 |

## 限制

- Graphify edge 可能來自 AST、檔名、文字與推論，不代表真實 runtime dependency。
- `tachiya` 是獨立 repo；除非把兩個 repo 都餵給 graphify，否則 cross-repo edge 只會出現在人工文件或外部連結。
- 圖譜不是 source of truth。架構事實以 source、tests、migration、API contract 和 Dev Portal domain map 為準。

## 本機使用方式

目前 graphify 產物沒有 commit 進 repo。要重建時，在本機對 `tachigo` 與同層 `tachiya` repo 產生輸出，然後用瀏覽器打開 graph HTML。

```bash
graphify /path/to/tachigo /path/to/tachiya
```

若後續要把互動圖嵌入 Docusaurus，建議另開 PR：

1. 產生 `graph.html` / `graph.json`。
2. 將可公開瀏覽的 HTML 複製到 `apps/docs/static/dev-portal/graph.html`。
3. 在本頁加入連結，並標註 graphify 產生時間與輸入 commit。

## 快速交叉檢查

| 想確認 | 先看 |
|---|---|
| Points / ledger hub | [Points / ledger / watch time](/tachigo/dev-portal/domain-maps#points--ledger--watch-time) |
| Auth / identity hub | [Auth / identity](/tachigo/dev-portal/domain-maps#auth--identity) |
| Extension / sidepanel hub | [Extension / sidepanel](/tachigo/dev-portal/domain-maps#extension--sidepanel) |
| tachigo ↔ tachiya edge | [Coupon redemption flow](/tachigo/dev-portal/flows#coupon-redemption-flow) |
