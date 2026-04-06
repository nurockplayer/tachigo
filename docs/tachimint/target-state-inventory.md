# Tachimint Target State Inventory

## 目標

定義 `tachimint` 在目標架構下的理想 state ownership 與畫面狀態。

這份文件承接：

- [frontend-roadmap.md](/Users/tachikoma/Documents/Web3/tachigo/docs/frontend-roadmap.md)
- [frontend-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/frontend-state-inventory.md)
- [home-state-inventory.md](/Users/tachikoma/Documents/Web3/tachigo/docs/tachimint/home-state-inventory.md)

重點不是描述現在怎麼運作，而是定義後續重構後：

- 每份 state 應該由哪支 hook 擁有
- 哪些狀態應該被畫出來
- 哪些狀態轉移應該被固定
- 哪些視覺原則必須讓它看起來像 Twitch 內的 extension，而不是獨立 app

---

## 目標架構

首頁 `Home / Mining` 目標拆成六支 hook：

- `useTwitch`
- `useWatchSession`
- `useHeartbeat`
- `useBalance`
- `useClickBoost`
- `useBits`

其中：

- `useTwitch` 處理 Twitch context、extension auth、Bits 商品初始化、viewer / non-viewer 分流
- `useWatchSession` 處理 start / end 與 session readiness
- `useHeartbeat` 只負責 heartbeat timer 與成功後通知 refetch
- `useBalance` 持有 `spendable` / `cumulative` / gain animation
- `useClickBoost` 只處理 click、cooldown、optimistic update
- `useBits` 只處理交易狀態與 receipt 驗證

---

## 使用者

- viewer
- broadcaster
- moderator
- external

---

## 主要任務

1. 非 viewer 不進 viewer 遊戲流程
2. viewer 完成 auth 後啟動 watch session
3. session ready 後取得 balance
4. heartbeat 只做定時同步，不自行持有 balance source of truth
5. click 走 cooldown + optimistic balance update
6. Bits 交易與首頁其他 state 不互相污染

---

## 依賴資料

- `window.Twitch.ext.onContext`
- `window.Twitch.ext.onAuthorized`
- `window.Twitch.ext.bits.getProducts`
- `POST /api/v1/extension/auth/login`
- `POST /api/v1/extension/watch/start`
- `POST /api/v1/extension/watch/heartbeat`
- `POST /api/v1/extension/watch/click`
- `POST /api/v1/extension/watch/end`
- `GET /api/v1/extension/watch/balance`
- `POST /api/v1/extension/bits/complete`

---

## Source of Truth

### `useTwitch`

擁有：

- `context`
- `extensionJwt`
- `products`
- `bitsEnabled`
- `authState`
- `tachigoAuthReady`

建議 `authState`：

- `loading`
- `ready`
- `account_unlinked`
- `backend_unavailable`

不應再持有：

- balance
- heartbeat error
- click cooldown

---

### `useWatchSession`

擁有：

- `sessionReady`
- `sessionStarting`
- `sessionError`

責任：

- auth ready + viewer + channelId 存在時，呼叫 `start`
- `beforeunload` best-effort `end`

不應再持有：

- balance
- click state

---

### `useHeartbeat`

擁有：

- `heartbeatState`
- `heartbeatError`

建議 `heartbeatState`：

- `idle`
- `running`
- `error`

責任：

- 定時呼叫 heartbeat
- 成功後觸發 `onSuccess()` / `refetch()`

不應再持有：

- `balance`
- `gain`
- `isAnimating`
- `syncBalance`

---

### `useBalance`

擁有：

- `spendable`
- `cumulative`
- `balanceState`
- `gain`
- `isAnimating`

建議 `balanceState`：

- `idle`
- `loading`
- `ready`
- `error`

責任：

- session ready 後拉 `GET /watch/balance`
- heartbeat 成功後 refetch
- click 成功後做 optimistic update

這裡應該成為 points 顯示的單一 source of truth。

---

### `useClickBoost`

擁有：

- `cooldownState`
- `cooldownMs`
- `clickState`
- `gain`
- `isAnimating`

建議：

- `clickState: 'idle' | 'pending' | 'success' | 'error'`
- `cooldownState: 'ready' | 'cooldown'`

責任：

- 發 click request
- 處理 optimistic cooldown
- 收到 `retry_after_ms` 時校準 cooldown
- 成功後呼叫 `optimisticClickUpdate()`

不應直接持有：

- 真實 balance

---

### `useBits`

擁有：

- `status`
- `error`

建議 `status`：

- `idle`
- `pending`
- `success`
- `error`
- `unavailable`

責任：

- 啟動 Twitch Bits flow
- receipt complete 後呼叫 backend 驗證

---

## 狀態表

| State | 觸發條件 | UI 表現 | 是否可操作 | Source of Truth |
|-------|----------|---------|------------|-----------------|
| `context loading` | `context === null` | 全畫面 loading | 否 | `useTwitch` |
| `non-viewer view` | `context.role !== 'viewer'` | 顯示 broadcaster / moderator / external 提示頁 | 否 | `useTwitch` + App role gate |
| `auth loading` | viewer 且 `authState === 'loading'` | 驗證中畫面 | 否 | `useTwitch` |
| `auth ready` | viewer 且 `authState === 'ready'` | 可進入 app shell | 是 | `useTwitch` |
| `auth account unlinked` | viewer login 401 且對應帳號未建立 | 顯示前往 tachigo.io 指引 | 否 | `useTwitch` |
| `auth backend unavailable` | viewer login 5xx / network / 其他不可恢復錯誤 | 顯示暫時不可用畫面 | 否 | `useTwitch` |
| `session starting` | `useWatchSession` 正在 start | 首頁主內容可顯示，但主要 CTA disabled | 否 | `useWatchSession` |
| `session ready` | start 成功 | heartbeat / click / balance 流程可啟用 | 是 | `useWatchSession` |
| `session error` | start 失敗 | 顯示 session error / retry | 否 | `useWatchSession` |
| `balance loading` | session ready 後首次抓 balance | points 區塊顯示 skeleton 或 placeholder | 否 | `useBalance` |
| `balance ready` | spendable / cumulative 取得成功 | 顯示兩個數字 | 是 | `useBalance` |
| `balance animating` | refetch 後 spendable 增加 | bump + gain 動畫 | 是 | `useBalance` |
| `balance error` | balance refetch 失敗 | 顯示局部錯誤 / retry | 視情況 | `useBalance` |
| `heartbeat running` | viewer + session ready | 顯示掛台同步狀態 | 是 | `useHeartbeat` |
| `heartbeat error` | heartbeat request 失敗 | 顯示 degraded banner | 是，但需提示 | `useHeartbeat` |
| `click ready` | `sessionReady && balance ready && cooldownMs === 0` | mine button 可點 | 是 | `useClickBoost` + derived gating |
| `click cooldown` | `cooldownMs > 0` | button disabled + countdown | 否 | `useClickBoost` |
| `click gain` | click 成功 | 顯示浮字與局部動效 | 否，通常在 cooldown 中 | `useClickBoost` |
| `click error` | click 非預期失敗 | 顯示局部錯誤並解除 lock | 是，可重試 | `useClickBoost` |
| `bits unavailable` | `bitsEnabled === false` 或商品不可用 | 顯示 unavailable / empty 區塊 | 否 | `useTwitch` + `useBits` |
| `bits idle` | `status === 'idle'` | 顯示可購買商品卡 | 是 | `useBits` |
| `bits pending` | `status === 'pending'` | 商品按鈕 disabled | 否 | `useBits` |
| `bits success` | `status === 'success'` | 顯示成功狀態 | 視產品決策 | `useBits` |
| `bits error` | `status === 'error'` | 顯示錯誤訊息 | 是，可重試 | `useBits` |

---

## 狀態轉移

### 1. Loading → non-viewer / auth

1. `context loading`
2. `onContext()` 回來
3. 若 `role !== 'viewer'`
   - 進 `non-viewer view`
4. 若 `role === 'viewer'`
   - 進 `auth loading`
   - 等待 backend login 完成

### 2. Auth → session

1. `auth ready`
2. `useWatchSession` 啟動
3. 進 `session starting`
4. 成功 → `session ready`
5. 失敗 → `session error`

### 3. Session → balance / heartbeat

1. `session ready`
2. `useBalance` 首次抓資料 → `balance loading`
3. 成功 → `balance ready`
4. 同時 `useHeartbeat` 進 `heartbeat running`
5. heartbeat 成功後只觸發 `useBalance.refetch()`

### 4. Ready → click

1. `session ready` + `balance ready`
2. `click ready`
3. 點擊後：
   - 立刻進 `click cooldown`
   - 成功時顯示 `click gain`
   - 同步做 optimistic balance update
4. 非預期錯誤：
   - `click error`
   - 解除 cooldown

### 5. Ready → bits

1. `bits idle`
2. 使用者購買 → `bits pending`
3. 成功 → `bits success`
4. 驗證失敗 → `bits error`
5. 若平台不支援或無商品 → `bits unavailable`

---

## 目標設計要求

### 視覺方向

- Twitch-native first，GameFi flavor second
- 可以保留 capybara miner、gain feedback、missions / equipment / shop 等遊戲化元素
- 但整體 form factor、資訊密度、層級與語氣應優先像 Twitch extension

### 不應偏移成

- 獨立遊戲 launcher
- 錢包 popup
- 外部 SaaS dashboard

## 非 viewer 分流

### 必須有

- `broadcaster`
- `moderator`
- `external`

三者至少要共用一個明確 non-viewer 畫面，不可再誤落 viewer 主面板。

---

## Auth 狀態

### 必須有

- `loading`
- `ready`
- `account_unlinked`
- `backend_unavailable`

### 不應有

- 只用 header 紅點表達 auth 問題

---

## Balance 狀態

### 必須有

- 單一 source of truth
- loading / ready / error
- gain animation 與數字來源同一支 hook

### 不應有

- heartbeat 與 click 各自持有不同 balance state

---

## Click 狀態

### 必須有

- button enabled 條件綁定 `sessionReady && balance ready`
- cooldown 與 server `retry_after_ms` 可對齊
- unexpected error 可恢復

---

## Heartbeat 狀態

### 必須有

- running / error 可見
- heartbeat 失敗不應讓使用者完全無感

### 不應有

- 用 heartbeat response 直接當成 balance source of truth

---

## 已知與現況差異

### 1. 目標中 `useHeartbeat` 不再持有 balance

這和目前現況文件最大的差異之一，也是重構優先順序最高的地方。

### 2. 目標中非 viewer 一律先被 role gate 擋下

這用來修正現在 `moderator / external` 會落進 viewer 主畫面的問題。

### 3. 目標中 `authError` 會變成可理解的 authState

這用來修正現在只有紅點、沒有真正 fallback 畫面的問題。

### 4. 目標中 click / balance / heartbeat 三者責任被重新切乾淨

這用來避免後續再出現：

- 雙重 source of truth
- gain animation 重複計算
- click 與 heartbeat 互相污染

---

## 驗收清單

1. 每個重要畫面 state 都有對應的 source of truth
2. `useHeartbeat` 不再持有 balance
3. `useBalance` 成為 points 顯示唯一真實來源
4. non-viewer roles 不再落入 viewer 主畫面
5. auth failure 至少能區分：
   - account unlinked
   - backend unavailable
6. click enabled 條件至少依賴：
   - viewer
   - auth ready
   - session ready
   - balance ready
7. 看完文件後，另一位工程師應能直接開始拆 hook 與重構 UI state

---

## 後續建議

最自然的下一步有兩種：

1. 依這份文件開一組重構 issue
   - auth state
   - watch session lifecycle
   - useBalance extraction
   - non-viewer split

2. 補一份 `tachimint-hook-ownership.md`
   直接把六支 hook 的輸入、輸出、責任與互動方式文件化
