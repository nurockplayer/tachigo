# Claude Code + Codex 工作流

本專案建議使用「Claude Code 負責指揮，Codex 負責執行」的協作模式。

目標有兩個：

1. 降低 Claude Code token 消耗
2. 讓實作、測試、搜尋、修改更穩定且可重複

## 核心原則

### Claude Code 負責什麼

- 理解需求
- 做架構與取捨判斷
- 拆解任務
- 審查 Codex 執行結果
- 撰寫 review、總結、設計決策

### Codex 負責什麼

- 搜尋檔案
- 閱讀相關程式碼
- 修改檔案
- 跑測試
- 執行指令
- 收斂錯誤原因

一句話原則：

「需要即時看輸出來做決策」的工作交給 Claude Code；
「可以直接執行並回報結果」的工作交給 Codex。

## 為什麼這樣比較省 token

如果讓 Claude Code 自己做以下事情，通常很容易浪費 token：

- 逐檔閱讀大量程式碼
- 一輪一輪跑測試再看錯誤
- 反覆嘗試 shell 指令
- 把很長的 log 與 diff 都帶進上下文

相對地，把這些工作交給 Codex，然後只讓它回傳短摘要，Claude Code 就只需要處理高密度資訊。

## 建議工作流程

1. 先在 Claude Code 中定義目標
2. 用 `codex:rescue` 或快捷指令把執行面工作交給 Codex
3. 讓 Codex 完成搜尋、修改、測試與驗證
4. Claude Code 只閱讀最後摘要
5. 若需要取捨或 review，再由 Claude Code 接手

## 快捷指令

完整指令清單與說明見 [CLAUDE.md](../CLAUDE.md)（AI 分工 → 建議優先使用的快捷指令）。

各指令的輸出格式限制定義在 `.claude/commands/*.md`。

## 什麼情況不要讓 Claude Code 自己做

以下工作通常都應該優先交給 Codex：

- 大量檔案搜尋
- trace call chain
- 修 bug 後反覆跑測試
- 批次改檔
- 執行 bash / CLI 任務
- 看 PR patch 並找具體問題

## 什麼情況適合由 Claude Code 自己做

- 需求還不清楚，需要先釐清方向
- 多個方案之間要做架構取捨
- 要寫給人看的文字
- 要把 Codex 回報的結果再做審查或決策

## 節省 token 的 prompt 寫法

建議在委派給 Codex 時，刻意限制輸出格式，例如：

```text
請只回：
1. Root cause
2. Changes made
3. Verification
4. Remaining risk

不要貼完整 diff
不要貼冗長 command output
只列出有改的檔案
```

這樣可以避免 Claude Code 吃進大量低價值內容。

## 常見反模式

### 反模式 1：讓 Claude 自己逐檔讀 codebase

這通常最耗 token，也最容易把大量低價值上下文帶進對話。

更好的做法：

- 先用 `/explore-with-codex`
- 只拿摘要回來

### 反模式 2：讓 Claude 一輪一輪看測試失敗

如果每次失敗都把整段 log 餵回 Claude，很快就會把上下文撐大。

更好的做法：

- 先用 `/test-with-codex`
- 只看收斂後的 failure summary

### 反模式 3：讓 Claude 直接做大量修改

這會讓 Claude 同時承擔理解、改檔、驗證三種工作，token 消耗通常比較高。

更好的做法：

- 需求與判斷留給 Claude
- 修改與驗證交給 Codex

## 建議

如果任務包含下列任一項，幾乎都應該先考慮交給 Codex：

- 「找」
- 「改」
- 「跑」
- 「修」
- 「測」
- 「review」

如果任務核心是：

- 「判斷」
- 「取捨」
- 「定方向」
- 「寫結論」

那就先由 Claude Code 主導。
