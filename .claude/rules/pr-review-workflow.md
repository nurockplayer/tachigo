# PR 審查工作流程

## 審查分工

1. **Gemini 低成本初審**：掃描高風險區域（binary、schema、scope 污染等）
2. **Claude / Codex 驗證**：確認 blocker、檢查 CLAUDE.md 合規、掃明顯 bug
3. **人類決策確認**：blocker → CR？無 blocker → merge？

## 問題分級

| 等級 | 定義 |
|---|---|
| `blocker` | 必須擋 merge：正確性、安全、資料一致性、breaking change、高風險路徑缺測 |
| `major` | 重要但不擋 merge；建議開 follow-up issue |
| `minor` | 有用改善，不阻擋 |
| `nit` | 純風格 / 可讀性細節 |

## 審查流程

### 第一步：高風險區域優先掃描

找到 high blocker 立即停止，不繼續深入：

- Binary 檔案（字體、圖片、screenshot、bundle、archive）
- 重大變更：schema、migration、API contract、auth / payment 邏輯
- Scope 污染：混入無關改動
- CI 失敗

### 第二步：無 high blocker 時進行必要審查

- 檢查 CLAUDE.md 合規性（scope、細粒度、AI 協作守則）
- 掃描明顯 bug、邏輯錯誤、edge case
- 檢查 git history 一致性（commit 訊息、原子化）
- 驗證現有 CodeRabbit / Codex comment 中的建議

## Gemini CLI 限制

- 避免並發任務（觸發 429）
- 大型 PR 用一個完整 prompt，不分批
- 需多個分析時改序列執行
- 遇到 429 / quota exceeded / daily limit：立即停止，改用 Claude / Codex 以最小必要上下文完成

## 審查執行模式

| PR 類型 | 規模 | 風險 | 模式 |
|---|---|---|---|
| UI 文案 / 樣式 / 簡單 refactor / 測試補充 | < 200 行 | 低 | 極省 token |
| Feature / API / 業務邏輯 | 100-600 行 | 中 | 標準 |
| Auth / payment / migration / schema / security / 權限 | 任何 | 高 | 高風險 |
| CI / 基礎設施 | 任何 | 中-高 | 標準或高風險 |

**極省 token**：Claude 掃 diff → Codex 主審 → 有疑點才叫 Gemini

**標準**：Claude 掃 metadata → Gemini first-pass（串行）→ Codex 主審 → Claude 驗證決策

**高風險**：Claude 標高危 → Gemini 窄範圍掃 → Codex 細審 → Claude 深度驗證

## 結構化輸出格式

```
## Summary
一句話總結 PR 與風險等級（低 / 中 / 高）。

## Blockers
- [檔案:行號] 問題 / 影響 / 建議

## Majors
- [檔案:行號] 問題 / 影響 / 建議

## Minors / Nits
- 建議

## Questions
- 需要作者確認的地方

## Recommended action
Change Request / Merge
```

## 互動話術

| 情況 | 結尾動作 |
|---|---|
| 有 blocker | 列出 blocker → 問「同意提交 Change Request 嗎？」 |
| 有 major/minor、無 blocker | 列出非阻塞建議 → 問「是否同意 Merge？」 |
| 只有 nit | 問「沒有 blocker，是否直接 Merge？」 |
| 不確定 | 列出疑點 → 問「繼續調查或先暫停？」 |

## Codex 執行守則

- **以 Diff 為中心**：優先讀 diff，不任意擴大上下文
- **問題分級要精準**：只有真的會破壞功能 / 安全性 / 資料的才標 blocker；沒把握的標 major
- **需要更多上下文時先說明理由**，讓 Claude 決定是否展開，避免盲目擴張
- **立即停止的情況**：發現明確 blocker、上下文快速膨脹、遇到架構決策岔路
