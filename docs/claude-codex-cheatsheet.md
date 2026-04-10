# Claude Code + Codex Cheat Sheet

## 分工原則

- Claude Code：理解需求、做決策、審查結果、寫結論
- Codex：搜尋、改檔、跑測試、執行指令、收斂錯誤

一句話判斷：

- 需要「判斷」→ Claude Code
- 需要「執行」→ Codex

## 什麼最該交給 Codex

- 找相關檔案
- trace call chain
- 修 bug
- 改多個檔案
- 跑測試 / lint / build
- review PR

## 什麼適合留給 Claude Code

- 架構取捨
- 需求拆解
- 設計討論
- 最終 review
- 寫給人的文字

## 快捷指令

完整清單見 [CLAUDE.md](../CLAUDE.md)（AI 分工 → 建議優先使用的快捷指令）。

| 指令 | 一句話 |
| --- | --- |
| `/fix-with-codex` | debug 並直接修好 |
| `/implement-with-codex` | 實作功能 |
| `/review-with-codex` | bug / regression / 測試缺口 |
| `/explore-with-codex` | 摸清現況 |
| `/plan-with-codex` | 探索 → 短版計畫 |
| `/test-with-codex` | 跑測試並收斂失敗 |

## 最省 token 的 prompt 寫法

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

## 常見反模式

- 讓 Claude 自己逐檔讀很多程式碼
- 把整段 test log 一直貼給 Claude
- 讓 Claude 自己反覆跑指令試錯
- 讓 Claude 同時負責理解、改檔、測試

## 建議工作流

1. 用 Claude 定義任務
2. 用快捷指令把執行面工作丟給 Codex
3. 只讀 Codex 的短摘要
4. 需要取捨時再回到 Claude

## 文件

- 完整版教學：`docs/claude-codex-workflow.md`
- 協作入口：`CLAUDE.md`
