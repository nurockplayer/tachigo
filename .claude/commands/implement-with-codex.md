用 `codex:rescue` 實作這個需求，並盡量節省 Claude token。

安全邊界：此命令只授權本機修改與驗證；不得 commit、push、開 PR、開立或編輯 issue、comment、review、approve 或 merge，除非使用者在該回合另外明確要求並再次確認。

需求：
$ARGUMENTS

請你：
1. 先快速找出相關檔案與現有實作。
2. 在 repo 中以本機修改完成需要的實作。
3. 補必要但最小化的測試。
4. 跑驗證並確認沒有明顯 regression。

回覆格式請嚴格限制為：
1. `Implementation summary`
2. `Changed files`
3. `Verification`
4. `Follow-up items`

限制：
- 不要貼完整 diff
- 不要貼冗長 log
- 不要重貼需求
- 只摘要最終結果
