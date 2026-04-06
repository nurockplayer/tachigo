# Tachimint Home / Mining State Inventory

## 目標

定義 `tachimint` 目前首頁 `Home / Mining` 畫面的真實 state，作為後續重構：

- `useTwitch`
- `useHeartbeat`
- `useClickBoost`
- `useBits`

的基準文件。

這份文件描述的是**現況**，不是目標中的六支 hook 架構，也不是未來 GameFi 完整設計稿。

---

## 使用者

- viewer
- broadcaster
- moderator（目前未明確處理）
- external（目前未明確處理）

---

## 主要任務

1. 等待 Twitch Extension context 與 authorization 就緒
2. viewer 進入首頁後看到 points、挖礦按鈕與 Bits 商品
3. heartbeat 定時同步點數
4. click 觸發點數增益與 cooldown
5. Bits 交易可進入 pending / success / error 流程

---

## 依賴資料

- `window.Twitch.ext.onContext`
- `window.Twitch.ext.onAuthorized`
- `window.Twitch.ext.bits.getProducts`
- `window.Twitch.ext.bits.useBits`
- `POST /api/v1/extension/auth/login`
- heartbeat API（目前前端 contract 與後端預期不一致）
- click API
- bits complete API

---

## Source of Truth

### `useTwitch`

持有：

- `context`
- `jwt`
- `products`
- `bitsEnabled`
- `authError`

說明：

- `context` 決定 viewer / broadcaster 分支
- `jwt` 目前同時被 `useBits` 與 `useHeartbeat` 使用
- backend login 失敗時只留下 `authError`，UI 目前只顯示紅點

### `useHeartbeat`

持有：

- `balance`
- `gain`
- `isAnimating`
- `error`
- `syncBalance()`

說明：

- 目前同時擁有 balance state 與 gain animation ownership
- `syncBalance()` 被 `useClickBoost` 用來避免下一次 heartbeat 動畫重複計算

### `useClickBoost`

持有：

- `cooldownMs`
- `isAnimating`
- `gain`

說明：

- click 成功後會呼叫 `onBalanceUpdate(result.balance)`，目前實際是把 heartbeat hook 裡的 balance 同步掉

### `useBits`

持有：

- `status`
- `error`

`status` 目前只有四種：

- `idle`
- `pending`
- `success`
- `error`

---

## 狀態表

| State | 觸發條件 | UI 表現 | 是否可操作 | API / 資料來源 |
|-------|----------|---------|------------|----------------|
| `context loading` | `context === null` | 全畫面 loading spinner，文字 `Connecting…` | 否 | `useTwitch.context` |
| `viewer ready` | `context.role === 'viewer'` 且進入主面板 | 顯示 points、mine button、products | 是 | `App.tsx` viewer 分支 |
| `broadcaster view` | `context.role === 'broadcaster'` | 顯示 broadcaster 專用簡單提示畫面 | 否 | `App.tsx` broadcaster 分支 |
| `moderator/external undefined` | `context` 存在但 role 非 `viewer` / `broadcaster` | 目前會落回 viewer 主畫面結構，未明確分支 | 部分可操作，但語意不清楚 | `App.tsx` 現況 |
| `auth degraded` | backend login 失敗，`authError !== null` | header 右上角顯示紅點，title 為錯誤訊息 | 仍可部分操作 | `useTwitch.authError` |
| `balance unknown` | `balance === null` | points 顯示 `—` | click 可操作與否取決於 viewer/cooldown，不取決於 balance | `useHeartbeat.balance` |
| `heartbeat success` | `sendHeartbeat()` 成功 | 若有新點數則 balance 更新、顯示 `+N 點` 與 bump 動畫 | 是 | `useHeartbeat` |
| `heartbeat error` | `sendHeartbeat()` 丟錯 | 畫面沒有獨立錯誤區，只是 hook 內有 `error` state | 是 | `useHeartbeat.error` |
| `click ready` | `channelId` 存在、`enabled === true`、`cooldownMs === 0` | mine button 可點擊 | 是 | `useClickBoost` |
| `click cooldown` | `cooldownMs > 0` | mine button disabled，顯示秒數倒數 | 否 | `useClickBoost.cooldownMs` |
| `click gain` | click 成功後 `gain !== null` | 顯示 `+delta` 浮字 1.5s | 否，通常同時在 cooldown 中 | `useClickBoost.gain` |
| `click unexpected error` | click 非 429 失敗 | UI 會解除 cooldown，但沒有獨立錯誤訊息 | 會重新可操作 | `useClickBoost` catch 分支 |
| `bits unavailable` | `bitsEnabled === false` | 顯示 `Bits not available.` | 否 | `useTwitch.bitsEnabled` |
| `bits idle` | `status === 'idle'` | 顯示商品列表與購買按鈕 | 是 | `useBits.status` |
| `bits pending` | `status === 'pending'` | 購買按鈕 disabled，文案變 `…` | 否 | `useBits.status` |
| `bits success` | `status === 'success'` | 顯示 success 區塊與 reload button | 否 | `useBits.status` |
| `bits error` | `status === 'error'` | 顯示錯誤文字，商品列表仍保留 | 是 | `useBits.error` |

---

## 狀態轉移

### 1. Loading → viewer / broadcaster

1. 初始進入 `context loading`
2. `window.Twitch.ext.onContext()` 回來後設定 `context`
3. 若 `context.role === 'broadcaster'`
   - 進入 `broadcaster view`
4. 其他情況
   - 進入主面板
   - 若 `role === 'viewer'`，heartbeat 會被啟用
   - 若 `role !== 'viewer'` 且不是 broadcaster，目前沒有明確分支，會落進 viewer 主面板結構

### 2. Viewer → auth degraded

1. `onAuthorized()` 取得 extension JWT
2. 前端嘗試呼叫 backend login
3. 若成功：
   - `setAuthToken(tokens.access_token)`
   - UI 沒有明確 success state，只是正常運作
4. 若失敗：
   - `authError = 'Backend unavailable'`
   - Header 顯示紅點
   - Bits 仍可能照常初始化並可用

### 3. Ready → heartbeat success / error

1. `useHeartbeat(enabled: isViewer)` 啟動
2. 若 heartbeat 成功：
   - 更新 `balance`
   - 若新 balance 大於前一筆，顯示 gain 與 bump 動畫
3. 若 heartbeat 失敗：
   - hook 內設定 `error = 'Heartbeat failed'`
   - App 目前沒有把這個 error render 出來

### 4. Ready → click cooldown / gain / error

1. 點擊 mine button
2. 立即進入 optimistic `click cooldown`
3. 若 click 成功：
   - 更新 balance baseline
   - 顯示 `click gain`
4. 若 server 回 `retry_after_ms`
   - cooldown 改用 server 指定秒數
5. 若非預期錯誤：
   - cooldown 解除
   - 沒有獨立錯誤 UI

### 5. Bits idle → pending / success / error

1. 使用者點商品按鈕
2. 進入 `bits pending`
3. 若 `onTransactionComplete` + backend 驗證成功：
   - 進 `bits success`
4. 若 backend 驗證失敗：
   - 進 `bits error`
5. 若 `onTransactionCancelled`
   - 回 `bits idle`

---

## 已知問題 / 與目標架構落差

### 1. `useHeartbeat` 同時擁有 balance 與 gain animation

現況：

- `useHeartbeat` 既是 heartbeat scheduler，又持有 `balance`、`gain`、`isAnimating`

落差：

- 和後續拆成 `useHeartbeat` + `useBalance` 的方向不一致

### 2. `useTwitch` 目前任何 role 都會嘗試 backend login

現況：

- `onAuthorized()` 後直接 login backend，沒有先看 role

落差：

- 後續目標架構預期應先分 viewer / non-viewer，再決定是否需要 tachigo auth

### 3. auth 失敗只顯示紅點

現況：

- backend login 失敗時只有 `authError` 紅點

落差：

- 後續應拆成可理解的 auth state 與對應畫面，而不是只靠 header indicator

### 4. `moderator / external` 目前沒有明確 UI 分支

現況：

- `App.tsx` 只特判 `broadcaster`
- 其他非 viewer 角色目前會落回主面板

落差：

- 後續應明確定義 non-viewer policy，至少不要誤用 viewer UI

### 5. heartbeat 目前以 `extensionJwt` 為輸入

現況：

- `useHeartbeat(extensionJwt, { enabled: isViewer })`

落差：

- 後端 watch API 的目標契約應該以 tachigo JWT + `channel_id` 為主，不應由 extension JWT 直接驅動 heartbeat request

### 6. click enabled 條件過寬

現況：

- 只要 `isViewer` 且沒有 cooldown，就可能可點

落差：

- 後續應至少依賴 session ready 與 balance ready，而不是只看 viewer role

### 7. heartbeat error 目前沒有實際 render

現況：

- `useHeartbeat.error` 存在，但 `App.tsx` 沒有顯示

落差：

- 使用者目前看不到 heartbeat 是否失敗，也沒有 retry / degraded UI

---

## 驗收清單

1. 文件中的每個 state 都能對應到目前程式碼裡的真實條件
2. 至少覆蓋以下情境：
   - `window.Twitch.ext` 尚未提供 context
   - `viewer` 正常進首頁
   - `broadcaster` 顯示非 viewer 畫面
   - backend login 失敗但 bits 仍可用
   - heartbeat 失敗
   - click cooldown
   - bits pending / success / error
3. 看完文件後，另一位工程師應能快速知道：
   - 現在 Home / Mining 的真實 state
   - 每份 state 現在由哪支 hook 持有
   - 哪些地方是後續重構的優先缺口

---

## 後續建議

這份文件之後最自然可接兩個方向：

1. 補一份 `target-state-inventory.md`
   以目標六支 hook 架構重新定義理想 state

2. 依照這份現況文件，開始拆：
   - `useBalance`
   - `useWatchSession`
   - `authState`
   - non-viewer 分流
