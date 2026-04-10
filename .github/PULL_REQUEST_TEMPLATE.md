## 什麼改動
<!-- 簡單說明做了什麼 -->

## 為什麼
<!-- 背景、需求，或關聯的 issue（e.g. closes #123） -->

## Release Context
<!-- 若這是正式 release PR，請填：`develop -> main`；否則填 `n/a` -->
- Release type：<!-- `develop -> main` 或 `n/a` -->

## Scope 對齊
<!-- 一般 feature PR 若超過約 35 個檔案、diff 過大、同時改多個 product surface，或依賴尚未 merge 的 backend contract，會被 PR Scope Police 自動擋下。正式 `develop -> main` release PR 請使用 `[release]` title prefix，並在上方 Release Context 填 `develop -> main`。 -->
- Source of truth：<!-- issue / PR / 文件，例如 #115 -->
- Depends on PR：<!-- `none` 或 `#123` -->
- Backend contract already in develop:
  - [ ] yes
  - [ ] no
- If no, this PR is:
  - [ ] stacked on dependency branch
  - [ ] intentionally blocked until dependency merges
- 本 PR 是否完全在 source of truth 範圍內？
  - [ ] 是
  - [ ] 否，已另開 issue / PR 處理超出部分
- 本 PR 明確不做：
  - <!-- 例：不做 future panels / 不做 router / 不做 dashboard UI -->

## 超出範圍內容
<!-- 若有額外重構、future work、design exploration、順手修正，請寫在這裡；若無請填「無」 -->

## 測試方式
- [ ] 本地測試過
- [ ] 有寫 / 更新測試

## 備註
<!-- 其他需要 reviewer 注意的事情（可留空） -->
