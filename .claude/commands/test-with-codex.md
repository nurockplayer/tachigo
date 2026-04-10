用 `codex:rescue` 幫我執行測試並收斂結果，盡量節省 Claude token。

測試範圍：
$ARGUMENTS

請你：
1. 執行最相關的測試，不要無限制擴大範圍。
2. 如果失敗，先收斂到最可能的 root cause。
3. 如果是環境問題、測試資料問題、或真實 bug，請區分清楚。
4. 若能順手修掉小問題，可一併處理；若不適合直接修，先回報。

回覆格式請嚴格限制為：
1. `Test scope`
2. `Result`
3. `Failure summary`
4. `Recommended next step`

限制：
- 不要貼完整 log
- 只貼關鍵錯誤片段
- 不要把所有通過的測試逐條列出
- 以摘要為主
