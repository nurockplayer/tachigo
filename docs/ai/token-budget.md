# Token 節流預設

Codex / Claude 在 PR review 與大型工作中，預設先收斂上下文，再決定是否深入。目標是保留審查品質，同時避免第一回合就讀完整 repo、完整 diff 或大量 log。

## PR Review

除非使用者明確要求一次完成完整審查，PR review 預設拆成回合執行。

第一回合只整理摘要，不進入完整 diff 深審：

- PR title / linked issue / scope
- changed files 與 diff stat
- CI / required checks 狀態
- 已有 review comments / requested changes 摘要
- 初步高風險區域
- 建議下一步要深入看的檔案

第一回合結尾要詢問使用者是否繼續深入 review，或只針對指定檔案 / 風險點檢查。

使用者確認後，才讀取必要 diff、Gemini 摘要、CI log 或 review thread；仍優先只讀必要片段。

## 大型工作

大型工作也預設拆成多回合：

1. 先做最小必要盤點與摘要。
2. 回報建議切分方式。
3. 等使用者確認後，再進入實作、深度審查或大量讀檔。

除非必要，不要在第一回合讀完整 repo、完整 diff、完整 CI log 或大量歷史上下文。
