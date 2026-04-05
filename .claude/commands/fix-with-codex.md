用 `codex:rescue` 處理這個問題，目標是盡量直接修好並把 Claude token 用量壓低。

問題：
$ARGUMENTS

請你：
1. 自行搜尋相關檔案與呼叫路徑。
2. 找出最可能的 root cause，避免列太多分支猜測。
3. 直接完成修復。
4. 補最小必要測試。
5. 跑相關驗證。

回覆格式請嚴格限制為：
1. `Root cause`
2. `Changes made`
3. `Verification`
4. `Remaining risk`

限制：
- 不要貼完整 diff
- 不要貼冗長 command output
- 不要貼大段原始碼
- 只列出實際修改的檔案
- 測試只摘要關鍵結果
