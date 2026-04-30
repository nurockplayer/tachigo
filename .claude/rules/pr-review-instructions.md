# PR Review 指令格式

## 下指令時的標準格式

貼 CodeRabbit / Codex comment URL 時，附帶一句簡述 blocker 是什麼，避免 Claude 來回查詢浪費 token：

### 不夠清楚
```
修
https://github.com/nurockplayer/tachigo/pull/277#discussion_r3104971218
```

### 清楚的格式
```
修：https://github.com/nurockplayer/tachigo/pull/277#discussion_r3104971218 — CodeRabbit 說應該用 useCallback wrap 這個 handler
```

或如果 comment 本身很明確，也可以簡單一點：
```
修 PR #277：CodeRabbit 的 comment（scope 污染：混入無關 UI 改動）
https://github.com/nurockplayer/tachigo/pull/277#discussion_r3104971218
```

**為什麼：** Claude 不用額外花 token 查詢 comment 的內容，可以直接開始修改。特別是在 token 成本敏感的情境（如使用 Haiku）下，一句話背景能節省 10-30% 的查詢 overhead。

## PR 審查流程

見 [CLAUDE.md](../../CLAUDE.md) 的「Review 流程」段落。
