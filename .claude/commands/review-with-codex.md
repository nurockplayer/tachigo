用 `codex:rescue` 審查這份變更，重點是找出高價值問題，並節省 Claude token。

前提：若審查範圍是 GitHub PR URL，Codex sandbox 需要有網路存取（`~/.codex/config.toml` 設定 `sandbox_permissions = ["network-full-access"]`）。

審查範圍：
$ARGUMENTS

請你：
1. 以 bug、regression、整合風險、缺少測試為主。
2. 降低 style nitpick 與低價值建議。
3. 若沒有明確問題，就直接說沒有 findings。

回覆格式請嚴格限制為：
1. `Findings`
2. `Open questions`
3. `Test gaps`

要求：
- 每個 finding 要附檔案與行號
- 依嚴重度排序
- 不要貼長篇 patch 或大量原文
