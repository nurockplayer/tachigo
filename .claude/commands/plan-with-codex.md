用 `codex:rescue` 先做必要探索，再產出一份短而可執行的計畫，並盡量節省 Claude token。

任務：
$ARGUMENTS

請你：
1. 先自行搜尋相關檔案、流程與相依元件。
2. 不要急著修改，先確認現況與限制。
3. 產出最小但可執行的方案。
4. 如果發現明顯 blocker，也一起列出。

回覆格式請嚴格限制為：
1. `Current state`
2. `Constraints`
3. `Proposed plan`
4. `Risks / blockers`

限制：
- 不要貼完整檔案內容
- 不要貼冗長搜尋輸出
- 不要直接開始實作
- 計畫以 3-6 個步驟為主，保持精簡
