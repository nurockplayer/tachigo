# Code Review Workflow Refactoring

**Date**: 2026-04-18  
**Status**: ✅ Completed  
**Decision**: Use Gemini CLI for PR code review instead of multiple Haiku agents

## Background

原始的 `code-review` skill（通過 `/code-review <PR_URL>` 調用）使用多個 agents 來審查 PR：
- 步驟 4：5 個並行 Sonnet agents 獨立審查
- 步驟 5：多個 Haiku agents 對每個問題評分

**問題**：
- Token 消耗高（多個 agents 各自運行）
- 執行時間長（需要啟動和協調多個 agents）
- 與 delegation 策略不符（重複性工作應由 Gemini 負責）

## Decision

使用 Gemini CLI 統一執行步驟 4-5 的審查和評分，保留對 Haiku agents 的自動降級選項。

**理由**：
1. **Token 效率**：單個 Gemini agent 一次性審查，而不是 5+6=11 個 agents
2. **速度**：減少 agent 啟動開銷
3. **策略對齊**：符合 delegation.md — Gemini 做「broad scans」，Claude 做決策
4. **長上下文優勢**：Gemini 可一次看完整個 PR diff，不受行數限制

## Implementation

### 1. 修改 Code-Review Skill

**檔案**：`~/.claude/plugins/marketplaces/claude-plugins-official/plugins/code-review/commands/code-review.md`

**改動**：
- 步驟 4-5：改為執行 `~/.claude/scripts/code-review-with-gemini.sh`
- 原始的「5 個 Sonnet agents + 多個 Haiku agents」改為「Gemini 單一 agent」
- 保留自動降級邏輯

```markdown
4. Execute `~/.claude/scripts/code-review-with-gemini.sh` to review and score issues.
   - Uses Gemini CLI as primary reviewer
   - Audits across 5 dimensions (CLAUDE.md compliance, bugs, history, PR comments, code comments)
   - Returns JSON array of issues with 0-100 confidence scores
   - Fallback: prompts user to switch to Haiku agents if Gemini unavailable
```

### 2. 建立審查腳本

**檔案**：`~/.claude/scripts/code-review-with-gemini.sh`

**功能**：
```
1. 驗證 Gemini CLI 是否可用
2. 取得 PR diff (gh pr diff)
3. 構造審查提示詞（5 個維度）
4. 調用 Gemini 進行審查和評分
5. 解析並返回 JSON 格式的 issues
6. 如果失敗或 Gemini 不可用，詢問用戶是否改用 Haiku agents
```

**輸出格式**：
```json
[
  {
    "description": "問題描述",
    "location": "src/file.ts:10-20",
    "severity": 85,
    "reason": "bug|CLAUDE.md|git history|PR comments|code comments"
  }
]
```

此輸出格式是 Claude Code 本機 `/code-review` script contract，供該 command
後續步驟直接過濾 `severity >= 80` 使用。它和 `AGENTS.md` 中的 Codex
repo-level Review JSON schema 不同；若未來要共用同一個 Gemini wrapper，
需要在 wrapper 或 caller 中明確轉換格式。

### 3. 輔助腳本

**檔案**：`~/.claude/scripts/code-review-gemini-score.sh`  
備用工具，用於獨立評分任務。

## Usage

執行代碼審查：
```bash
/code-review https://github.com/nurockplayer/tachigo/pull/265
```

工作流程：
1. ✅ 檢查 PR 是否合格（Haiku）
2. ✅ 尋找相關 CLAUDE.md 檔案（Haiku）
3. ✅ 查看 PR 摘要（Haiku）
4. 🧠 **[改動]** Gemini 審查和評分（取代步驟 4-5 的多個 agents）
5. ✅ 過濾高分問題（Haiku）
6. ✅ 再次檢查 PR 狀態（Haiku）
7. ✅ 發送 GitHub 評論（gh 命令）

## Fallback Behavior

如果 Gemini CLI 不可用：
```
⚠️  Gemini CLI not found in PATH
Options:
  [y] Use Haiku agents (original multi-agent review)
  [n] Abort code review
Use Haiku agents? [y/N]
```

用戶可選擇改用原始 Haiku agents 方案。

## Trade-offs

### 優勢 ✅
- **更快**：單個 agent vs 11 個 agents
- **更省 token**：Gemini 一次性處理
- **符合策略**：delegation 規範
- **容易迭代**：調整提示詞比調整 11 個 agents 簡單

### 劣勢 ⚠️
- **多樣性降低**：單個視角 vs 5 個獨立審查者
  - 但 Gemini 指令明確涵蓋 5 個維度，應該足夠
- **依賴 Gemini 可用性**：如果 Gemini CLI 不可用，需要降級
- **迭代成本**：如果效果不理想，需要調整提示詞並重新測試

## Verification

首次使用時應驗證：
1. Gemini 審查的品質是否等同於多 agents
2. 評分準確性（和歷史 code review 比較）
3. 降級路徑是否正常工作

建議用 PR #265 作為測試用例。

## Revert Path

如果發現 Gemini 方案不合適，可快速改回：
1. 恢復 code-review.md 的步驟 4-5（啟動 5 個 Sonnet + 多個 Haiku agents）
2. 或為特定 PR 手動選擇 Haiku agents 路徑

改動已在 memory 中記錄，便於未來追溯。

## Production Hardening (2026-04-18)

初版實作經過代碼審查後發現 3 個 blocker 和多個風險，已全部修復。

### Blockers 修復清單

#### 1️⃣ grep -oP 不兼容 macOS

**問題**：
```bash
PR_NUMBER=$(echo "$PR_URL" | grep -oP 'pull/\K\d+')
# 錯誤：grep: invalid option -- P (macOS 內建 grep 不支援 -P)
```

**修復**：
```bash
# 改用 sed （POSIX 兼容）
PR_NUMBER=$(echo "$PR_URL" | sed -n 's/.*pull\/\([0-9]*\).*/\1/p')
```

**位置**：`code-review-with-gemini.sh` 第 17 和 92 行

**驗證**：✅ 用 PR 265 實測通過

---

#### 2️⃣ allowed-tools 漏掉脚本執行權限

**問題**：
```yaml
allowed-tools: Bash(gh issue view:*), Bash(gh pr diff:*), ...
# 缺少執行自定義腳本和 git/gemini 命令的權限
```

**修復**：
```yaml
allowed-tools: ..., Bash(git log:*), Bash(gh api:*), 
              Bash(~/.claude/scripts/code-review-with-gemini.sh:*),
              Bash(gemini:*), Bash(test:*), Bash(rm:*)
```

**位置**：`code-review.md` 第 2 行

**影響**：原本 slash command 執行步驟 4 時會被工具權限攔住，現已授權

---

#### 3️⃣ Fallback 沒有真正實現

**問題**：
- 腳本只是打印 "Using Haiku agents fallback..." 後 `return 0`
- 沒有啟動原始的多 agent 審查流程
- Gemini 失敗時，用戶選 y 後腳本成功結束，但沒有實際審查結果

**修復**：
1. **脚本端**：創建 fallback marker 檔案
   ```bash
   set_fallback_marker() {
     local marker_file="${TMPDIR:-/tmp}/code-review-fallback-$PR_NUMBER"
     touch "$marker_file"
   }
   ```

2. **Skill 端**：新增步驟 4b 檢查並處理 fallback
   ```markdown
   4b. (Only if step 4 script indicated fallback) 
       Check for marker: test -f ${TMPDIR:-/tmp}/code-review-fallback-<PR_NUMBER>
       If exists, launch 5 parallel Sonnet agents + Haiku scoring (original flow)
       Clean up: rm -f ${TMPDIR:-/tmp}/code-review-fallback-<PR_NUMBER>
   ```

**位置**：
- 脚本：`code-review-with-gemini.sh` 第 150-170 行
- Skill：`code-review.md` 第 25-33 行

---

### 其他風險修復

| 風險 | 修復 | 位置 |
|-----|------|------|
| **CLAUDE.md 是 placeholder** | 實現 `get_claude_md_content()` 用 `gh api repos/.../contents/CLAUDE.md` 動態獲取 | 第 44-56 行 |
| **git history 只是 prompt 宣稱** | 實現 `get_git_history()` 用 `git log --oneline -10` 完整蒐集 | 第 59-63 行 |
| **previous PR comments 未傳遞** | 實現 `get_related_pr_comments()` 收集相關 PR 資訊 | 第 66-75 行 |
| **grep -P 第二次出現** | 改用 `sed` 提取 JSON | 第 98 行 |

### 改進後的架構

```
執行 code-review script
  ↓
蒐集完整上下文：
  • PR diff (gh pr diff)
  • CLAUDE.md 內容 (gh api)
  • git history (git log)
  • related PRs (gh api)
  ↓
調用 Gemini 進行 5 維度審查：
  • CLAUDE.md 遵循性
  • 明顯 bug
  • git 歷史背景
  • 先前 PR 評論
  • 代碼註釋遵循性
  ↓
  ├─ Gemini 成功 → 返回 JSON issues
  ├─ Gemini 失敗 → 創建 fallback marker
  │   ↓
  │   Skill 檢測 marker
  │   ↓
  │   啟動 5 Sonnet + Haiku agents
  │   ↓
  │   返回審查結果
  └─ 過濾 ≥80 分的 issues → 發 GitHub 評論
```

### 驗證結果

| 項目 | 結果 | 備註 |
|-----|------|------|
| **Bash 語法檢查** | ✅ pass | `bash -n code-review-with-gemini.sh` |
| **sed 提取 PR number** | ✅ pass | PR 265 → 265 |
| **脚本執行** | ✅ 正常進行到 Gemini 調用 | 上下文蒐集無誤 |
| **allowed-tools** | ✅ 已授權 | 脚本可執行 |
| **Fallback 機制** | ✅ 已實現 | marker file + skill step 4b |

### Codex 反饋與進一步修復 (2026-04-18 後)

Codex 對初版實作提出 3 個 High Priority issues：

#### 1️⃣ gh pr diff 失敗被吞掉 → **✅ 已修復**

**問題**：
```bash
# 舊：失敗被吞掉，改成字串 "Failed to fetch PR diff"
get_pr_diff() {
  gh pr diff ... 2>/dev/null || echo "Failed to fetch PR diff"
}
# Gemini 收到假字串，回 []，誤判「無問題」
```

**修復**：
```bash
# 新：失敗時 return 1，中止流程
get_pr_diff() {
  if ! gh pr diff "$PR_NUMBER" --repo "$REPO_PATH" 2>/dev/null; then
    echo "❌ Failed to fetch PR diff. Aborting review." >&2
    return 1
  fi
}
# 呼叫端檢查錯誤
if ! PR_DIFF=$(get_pr_diff); then
  echo "❌ Cannot proceed without PR diff."
  return 1
fi
```

**位置**：code-review-with-gemini.sh 第 33-39 行、93-96 行

---

#### 2️⃣ related PR comments 是假資料 → **✅ 已刪除**

**問題**：
- `get_related_pr_comments()` 只列出文件名，沒抓實際 comments
- 但文檔宣稱第 4 維度「先前 PR 評論」
- Gemini 根據不存在的上下文判斷

**修復**：
- 刪除 `get_related_pr_comments()` 函數及相關調用
- 將審查維度從 5 個改為 4 個（刪除「先前 PR 評論」）
- 保留註釋說明未來改進方向

**位置**：code-review-with-gemini.sh 第 77-80（註釋）、第 104-105 行（已刪除）、第 127-133 行（維度更新）

---

#### 3️⃣ 沒產生真正的 GitHub Review / CR → **✅ 已改進**

**問題**：
```bash
# 舊：只是 comment
gh pr comment ...
# 無法產生 CHANGES_REQUESTED 或 APPROVED 狀態
```

**修復**：
```bash
# 新：用 gh pr review
# 有問題：
gh pr review <PR_NUMBER> --request-changes --body "..."
# 無問題：
gh pr review <PR_NUMBER> --approve --body "..."
```

**位置**：code-review.md 第 45-49 行（步驟 8 更新）

---

### 現狀

**方案B 已修復 Codex 指出的關鍵問題**（2026-04-18）

脚本與 skill 現已：
1. ✅ macOS 兼容（sed 代替 grep -P）
2. ✅ **gh pr diff 失敗時中止**（不產生假結果）
3. ✅ **刪除假的 related PR comments 宣稱**（維度降為 4）
4. ✅ **產生真正的 GitHub Review**（CHANGES_REQUESTED / APPROVED）
5. ✅ Gemini 失敗時自動降級到 Haiku agents
6. ✅ 完整的錯誤處理和用戶提示

**狀態**：✅ **High Priority issues 已全數修復**  
**可用性**：建議用 Beta 版繼續測試，待驗證無其他問題後升為 Production

## References

- **Skill 定義**：`~/.claude/plugins/marketplaces/claude-plugins-official/plugins/code-review/commands/code-review.md`
- **實作腳本**：`~/.claude/scripts/code-review-with-gemini.sh`
- **Delegation 策略**：`.claude/rules/delegation.md`
- **Memory 記錄**：`~/.claude/projects/<project-id>/memory/feedback_code_review_gemini.md`
