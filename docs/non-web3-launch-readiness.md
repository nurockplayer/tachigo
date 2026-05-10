# Non-Web3 Launch Readiness Snapshot

狀態：active planning snapshot
最後校正：2026-05-10

本文記錄「暫時捨棄 Web3 上鏈部分」後，tachigo 距離可上線狀態的粗估、可交給 Codex 5.5 xHigh 全自動處理的範圍，以及仍需要真人介入的決策層級。

這不是 go-live approval，也不是完整 security review。它是 2026-05-10 依據 `develop`、GitHub PR / issue queue、CI 狀態與現有 docs 做出的工程判斷 snapshot。

## Scope

本次評估排除：

- 主網 / 測試網合約部署
- mint / burn / on-chain claim 的正式上線責任
- 錢包與鏈上資產作為 MVP 必要條件

本次評估保留：

- Viewer identity / auth
- Chrome sidepanel / Tachimint viewer flow
- watch-to-points / off-chain ledger
- coupon shop / Tachiya off-chain redemption
- streamer / agency / admin dashboard
- raffle / airdrop，如果產品決定納入首版
- deployment、migration、monitoring、backup、runbook

## Current Snapshot

- `develop` 最新 CI 在 2026-05-10 檢查時為 green。
- 先前 `#531` 到 `#558` 已 merged，並以 `autonomous-sprint-merged` 追蹤。
- 自動 queue 已恢復處理新工作，檢查時 open PR 包含：
  - `#560`：Dependabot pnpm lockfile repair，ready，CI green，待 review。
  - `#564`：GitHub SSH 443 push workaround docs，ready，CI green，待 review。
  - `#565`：refresh token rotation race fix，ready，CI green，待 review。
  - `#562`：email auth token transactional consumption，仍是 draft 且 `DIRTY`，需要 rebase / conflict resolution 後再 review。
- Backend 功能面已相當完整，包含 auth、Twitch / Google OAuth、Twitch Extension auth、Watch Points、claim / spend、dashboard API、airdrop、raffle 等。
- CI / automation 已能自動產 PR、跑 checks、標籤、auto-ready；但 GitHub review / branch policy / secrets / deploy target 仍會限制完全無人上線。

## Readiness Estimate

| 範圍 | 粗估完成度 | 主要缺口 |
|---|---:|---|
| Backend core，不含上鏈 | 75-85% | auth race / swallowed error 收尾、transaction boundary、migration 正式化、coupon reconciliation |
| Dashboard | 55-70% | 真人營運流程 UAT、role / permission 驗收、settings / unsupported flows 的產品決策 |
| Extension / viewer sidepanel | 60-75% | Chrome / Twitch 真機 UAT、legacy `/extension` 路徑移除前置條件確認、上鏈相關 UI 去留 |
| CI / repo automation | 75-85% | review gate / auto-merge policy 已強，但仍依賴 required review、secret 與外部通知設定 |
| Production infra / launch ops | 35-55% | deploy workflow、environment、domain、DB backup、monitoring、alerting、runbook |
| Product / ops policy | 40-60% | coupon compensation、support process、abuse handling、data retention、final MVP scope |

整體判斷：不含 Web3 上鏈時，專案不是「還很遠」，而是已進入 production-readiness 收尾期。工程上可以靠自動 sprint 大幅推進；真正阻塞公開上線的是外部平台、營運政策、部署與真人 UAT。

## What Two Codex 5.5 xHigh Agents Can Do

兩個 Codex 5.5 xHigh 全自動可以有效處理「已被 issue 或文件界定清楚」的工程工作。

建議分工：

| Lane | 責任 | 代表工作 |
|---|---|---|
| Codex A：backend / infra | auth hardening、migration、CI、deploy scaffolding | `#562`、`#565`、`#420`、`#463`、`#462` 的可程式化部分、health checks、backup / deploy runbook |
| Codex B：frontend / QA / docs | dashboard / extension UAT polish、API contract、docs cleanup | `#401`、`#321`、`#354`、dashboard / extension smoke tests、closed beta checklist |

適合全自動完成：

- 小到中型 bug fix
- race condition / transaction fix
- missing tests / regression tests
- CI failure diagnosis and repair
- PR review loop：review comment -> fix -> push -> re-check
- docs / runbook / checklist
- API contract generation
- legacy code cleanup，前提是 issue scope 清楚
- launch checklist 中可由 repo 驗證的項目

不適合全自動拍板：

- 是否正式砍掉或改名 Web3 / claim / token 相關產品表面
- coupon redemption 失敗時的補償政策
- 是否移除 `/extension/*` 路徑，除非已完成外部 dependency verification
- production deployment target
- OAuth / Twitch / Chrome Web Store / Tachiya / Saleor 帳號與憑證
- privacy / legal / data retention
- final go / no-go

## Human Intervention Required

真人不需要介入每一個 PR，但需要在下列層級做決策。

| 層級 | 需要人做什麼 | AI 可以協助什麼 |
|---|---|---|
| MVP scope | 決定首版是否包含 raffle、airdrop、coupon shop、dashboard 哪些頁面；決定 Web3 UI 是隱藏、改名或保留 | 整理選項、拆 issue、實作被選方案 |
| Ops policy | 決定 coupon compensation、客服流程、濫用處理、異常點數處理 | 寫 reconciliation job、admin tooling、runbook |
| Production environment | 提供 domain、DB、hosting、secret、OAuth app、Twitch / Google / Discord webhook 設定 | 寫 deploy workflow、env validation、health check、監控文件 |
| External platform UAT | 用真實 Twitch / Google / Chrome / Tachiya / Saleor 帳號跑完整流程 | 產 checklist、自動化可跑部分、修 UAT 發現的 bug |
| Risk acceptance | 對 auth、ledger、coupon、privacy、launch timing 做最後批准 | 提供風險摘要、測試結果、rollback plan |

## Practical Timeline

| 目標 | 估計 |
|---|---|
| Engineering demo / preview | 24-48 小時內有機會 |
| Closed beta，不含 Web3 且允許人工營運補救 | 2-5 天工程時間 |
| Public production launch | 至少 2-4 週 calendar time |

這個估計假設兩個 Codex 5.5 xHigh 可以持續處理 queue，且人能快速提供 scope、secret、deploy target 與 UAT 回饋。

## Recommended Next Queue

優先把自動 sprint 從「繼續找零散 issue」收斂成 closed beta launch checklist：

1. Merge / fix current queue
   - review and merge `#560`
   - review and merge `#564`
   - review and merge `#565`
   - rebase / repair `#562`，再跑完整 CI 與 review

2. Backend safety
   - 完成 `#420` 剩餘 swallowed error audit
   - 完成 `#462` 的 reconciliation / compensation 可程式化部分
   - 釐清 `#463` Atlas migration baseline 與 staging validation path

3. Product surface cleanup
   - 決定 Web3 / claim / token UI 在非上鏈 MVP 的命名與可見性
   - 僅在外部依賴確認後處理 `#390`
   - 清理 `#321` demo extension，如果仍無 runtime 依賴

4. Frontend and UAT
   - dashboard core workflows：login、streamers、raffles、transactions、settings
   - extension core workflows：auth、points balance、coupon / claim surface、raffle result
   - 建立 smoke / E2E checklist，分成自動可驗證與真人必驗證

5. Launch operations
   - `#212` deploy workflow planning 落地
   - `#545` Discord webhook secret 由人提供後接上
   - 補 production env validation、backup、monitoring、rollback runbook

## Decision Summary

- 兩個 Codex 5.5 xHigh 可以把剩餘工程工作推到約 80-90% launch readiness。
- 最後 10-20% 不是單純 code volume，而是產品、營運、憑證、外部平台與風險接受。
- 若目標是 closed beta，現在應該停止擴張 feature，集中處理 auth hardening、migration、reconciliation、deploy、UAT。
- 若目標是 public launch，必須先由真人定義 non-Web3 MVP scope 與 production owner，再讓 Codex 依 checklist 全自動推進。
