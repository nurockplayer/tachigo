用 `codex:rescue` 處理這個問題，目標是盡量直接修好並把 Claude token 用量壓低。

安全邊界：此命令只授權本機修改與驗證；不得 commit、push、開 PR、開立或編輯 issue、comment、review、approve 或 merge，除非使用者在該回合另外明確要求並再次確認。

問題：
$ARGUMENTS

請你：
1. 自行搜尋相關檔案與呼叫路徑。
2. 找出最可能的 root cause，避免列太多分支猜測。
3. 在本機完成修復。
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
