# 前端 State Inventory

## 目的

這份文件用來統一 `tachigo` 前端在設計與實作前的狀態盤點方式。

目標是避免：

- UI 只有 happy path，沒有 error / empty / forbidden
- 同一份資料被多個 hook 或頁面各自持有
- 畫面設計與 API contract 脫鉤
- PR review 每次重新討論一次「這個狀態到底要怎麼顯示」

---

## 使用方式

每做一個新頁面、新 panel 或新流程，至少先列出以下狀態：

1. `loading`
2. `success`
3. `empty`
4. `partial`
5. `forbidden`
6. `error`
7. `retry / cooldown / unavailable`

如果某個狀態「理論上不會發生」，也要明寫原因，不要省略。

---

## State 定義

### `loading`

資料尚未取得或流程尚未初始化完成。

要回答：

- 這時候顯示 skeleton、spinner 還是整頁 loading？
- 使用者能不能操作？
- 哪個條件代表 loading 結束？

### `success`

資料完整、流程正常、主要操作可進行。

要回答：

- 畫面最主要資訊是什麼？
- 成功後有沒有次狀態，例如 animating / synced / stale？

### `empty`

不是錯誤，但目前沒有資料。

要回答：

- 為什麼是空的？
- 是否需要 CTA？
- 是第一次使用的空，還是篩選後的空？

### `partial`

主要流程可用，但有部分資料缺失、延遲或 placeholder。

要回答：

- 哪些資料是真的？
- 哪些是 mock / placeholder？
- 使用者是否需要知道「這塊尚未開放」？

### `forbidden`

使用者已登入，但沒有權限看或做某件事。

要回答：

- 這是 `401` 還是 `403`？
- 顯示方式是整頁擋下，還是局部禁用？
- 是否需要提示使用者該找誰或該去哪裡？

### `error`

系統或請求失敗，使用者目前無法完成目標。

要回答：

- 是整頁錯誤還是局部錯誤？
- 可不可以重試？
- 是否需要保留原本畫面資料？

### `retry / cooldown / unavailable`

這類不是傳統 error，但也不是正常 success。

常見於：

- click cooldown
- bits pending
- backend unavailable
- refresh / reconnect

要回答：

- 這是短暫中間狀態，還是明確失敗？
- 使用者現在能做什麼？
- 是否要顯示倒數、重試鈕或 disabled 說明？

---

## State Inventory 模板

可直接複製以下區塊到新文件或 issue：

```md
# <頁面 / 流程名稱>

## 目標

- 使用者：
- 主要任務：
- 依賴資料：

## Source of Truth

- Auth state:
- Session state:
- Primary data state:
- UI-only state:

## States

| State | 觸發條件 | UI 表現 | 是否可操作 | API / 資料來源 |
|-------|----------|---------|------------|----------------|
| loading |  |  |  |  |
| success |  |  |  |  |
| empty |  |  |  |  |
| partial |  |  |  |  |
| forbidden |  |  |  |  |
| error |  |  |  |  |
| retry/cooldown/unavailable |  |  |  |  |

## Transitions

- 從 loading 到 success：
- 從 loading 到 error：
- 從 success 到 cooldown / pending：
- 從 error 到 retry：

## Notes

- 哪些資料是 placeholder：
- 哪些狀態需要動畫：
- 哪些狀態需要 telemetry / logging：
```

---

## Dashboard 範例

### `Channel Config` 頁

| State | 觸發條件 | UI 表現 | 是否可操作 | API / 資料來源 |
|-------|----------|---------|------------|----------------|
| loading | 首次進頁，設定尚未載入 | 表單 skeleton | 否 | `GET /dashboard/channels/:channel_id/config` |
| success | config 成功取得 | 顯示表單與儲存按鈕 | 是 | config payload |
| empty | channel 尚未有設定，回預設值 | 顯示預設值與提示文案 | 是 | 預設 config |
| forbidden | 使用者不是該 channel owner | 整頁 forbidden state | 否 | `403` |
| error | API 失敗 | 錯誤訊息 + retry | 視情況 | `500` / network error |

---

## Tachimint 範例

### `Home / Mining` panel

| State | 觸發條件 | UI 表現 | 是否可操作 | API / 資料來源 |
|-------|----------|---------|------------|----------------|
| loading | Twitch context / auth 尚未完成 | 全畫面 loading | 否 | `useTwitch` |
| success | auth ready + session ready + balance ready | 主舞台、resource bar、mine button | 是 | `useWatchSession` + `useBalance` |
| partial | missions / buffs / equipment 尚未接真 API | 顯示 placeholder 區塊 | 部分可操作 | local placeholder |
| forbidden | non-viewer role | 顯示 broadcaster / non-viewer 提示 | 否 | `context.role` |
| error | backend unavailable / heartbeat error | 顯示錯誤 banner 或 fallback state | 視情況 | API error |
| cooldown | click 後等待下一次可點擊 | button disabled + countdown ring | 否 | `useClickBoost` |

---

## Source of Truth 原則

狀態盤點時，務必額外標註每個畫面的 source of truth。

### 建議分類

- `server state`
  來自 API / query cache，例如 config、balance、transactions

- `session state`
  與 auth、JWT、watch session 生命週期相關

- `ui state`
  純畫面狀態，例如 active tab、dialog open、hover、animation

- `derived state`
  從其他 state 推導出來，例如 `canClick`、`showEmptyState`

### 禁忌

- 同一份 server data 由多個 hook 各自維護
- 用 UI state 偷偷覆蓋 server truth，卻沒有校準策略
- 在 component tree 多處各自判斷 role / auth / forbidden

---

## Review Checklist

做前端 PR review 時，可以直接檢查：

1. 這個頁面有沒有列出完整 state inventory？
2. `401`、`403`、`404`、`500` 的 UI 是否被區分？
3. empty state 是否和 error state 分開？
4. source of truth 是否只有一份？
5. placeholder / mock data 是否有被明確標示？
6. 非 happy path 是否有對應測試或至少有設計說明？

---

## 建議下一步

可以優先為以下兩塊補正式 state inventory：

1. `dashboard` 的 `auth + protected routes + channel config`
2. `tachimint` 的 `viewer home / mining flow`
