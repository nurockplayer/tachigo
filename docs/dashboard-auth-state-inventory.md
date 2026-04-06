# Dashboard Auth State Inventory

## 目標

定義 `dashboard` 在登入、token restore、route guard 與權限判斷時的完整狀態。

這份文件優先處理：

- login page
- app 啟動後的 auth restore
- protected routes
- `401` / `403` 的 UI 行為

不處理：

- 單一業務頁面的資料 state
- channel config / stats / transactions 自身的資料狀態

---

## 使用者

- streamer
- agency
- admin

---

## 主要任務

1. 已有帳號的使用者可以登入 `dashboard`
2. 已登入使用者重新整理後，session 能被正確 restore
3. 未登入使用者不能進 protected routes
4. 已登入但權限不足的使用者，看到明確的 `forbidden` UI，而不是被誤導成 logout

---

## 依賴資料

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/users/me`
- access token
- refresh token
- user role

---

## Source of Truth

### Auth state

單一來源，建議集中在 auth provider / auth store。

建議欄位：

- `authState: 'booting' | 'anonymous' | 'authenticating' | 'authenticated' | 'refreshing' | 'forbidden' | 'error'`
- `user: User | null`
- `accessTokenReady: boolean`
- `refreshTokenPresent: boolean`

### Route state

由 router / protected route 從 auth state 推導，不自行重複保存。

建議 derived state：

- `canEnterProtectedRoute`
- `shouldRedirectToLogin`
- `shouldShowForbidden`

### UI-only state

- login form values
- form validation errors
- submit pending

---

## 狀態表

| State | 觸發條件 | UI 表現 | 是否可操作 | API / 資料來源 |
|-------|----------|---------|------------|----------------|
| `booting` | App 啟動，尚未檢查 token / session | 全頁 loading 或 app shell skeleton | 否 | local token + restore flow |
| `anonymous` | 沒有有效登入資訊 | 顯示 login page | 是 | local auth state |
| `authenticating` | 使用者送出 login form | login button pending、表單 disable | 否 | `POST /auth/login` |
| `authenticated` | login / refresh / restore 成功，且 `GET /users/me` 成功 | 可進入 protected app | 是 | access token + user profile |
| `refreshing` | access token 過期，正在 refresh | 保留原頁面或全域 loading overlay | 建議局部禁止敏感操作 | `POST /auth/refresh` |
| `forbidden` | 已登入，但 user role 不符頁面要求 | 顯示 forbidden state，不直接 logout | 否 | `403` 或 role gating |
| `error` | auth restore / login / refresh 發生不可恢復錯誤 | 顯示錯誤訊息與 retry / 返回登入 | 視情況 | network / `5xx` / invalid state |

---

## 狀態轉移

### App 啟動

1. 進入 `booting`
2. 檢查是否有 refresh token / session restore 資訊
3. 若沒有：
   - 轉成 `anonymous`
4. 若有：
   - 嘗試 refresh / restore
   - 成功後進 `authenticated`
   - 失敗後依錯誤類型進 `anonymous` 或 `error`

### 使用者登入

1. `anonymous`
2. 使用者送出表單
3. 進入 `authenticating`
4. 成功：
   - 儲存 token
   - 取得 user profile
   - 進入 `authenticated`
5. 失敗：
   - 回到 `anonymous`
   - 顯示表單錯誤或全域錯誤

### Access token 過期

1. `authenticated`
2. 某 API 回 `401`
3. 若 refresh token 存在且可嘗試 refresh：
   - 進 `refreshing`
   - 成功回 `authenticated`
   - 失敗依類型：
     - refresh token 無效 / session 已失效 → `anonymous`
     - 暫時性故障 → `error` 或保留頁面並提示 retry

### 權限不足

1. `authenticated`
2. 使用者進入不符合 role 的頁面，或 API 回 `403`
3. 進入 `forbidden`
4. 不應直接 logout

---

## 各狀態設計要求

## `booting`

### 必須有

- 明確的載入中畫面
- 不應閃出 login page 再跳回 app

### 不應有

- 尚未完成 restore 就先 render protected content

---

## `anonymous`

### 必須有

- login form
- 清楚的登入錯誤訊息區域

### 不應有

- 看起來像系統壞掉的空白畫面

---

## `authenticating`

### 必須有

- submit pending
- 防止重複送出

### 不應有

- 一次點兩次造成雙重登入請求

---

## `authenticated`

### 必須有

- user 基本識別資訊
- app shell 正常渲染
- token refresh 不應破壞目前頁面

---

## `refreshing`

### 必須有

- 對使用者來說是可理解的短暫中間狀態
- 若 refresh 成功，應盡可能無感恢復

### 特別注意

- refresh 暫時失敗不應一律當成 logout
- queued requests 不應因 refresh race condition 全部壞掉

---

## `forbidden`

### 必須有

- 明確文字說明沒有權限
- 若有必要，可提供返回安全頁面的 CTA

### 不應有

- 把 `403` 誤導成 `401`
- 直接清 session / 直接踢回 login

---

## `error`

### 必須有

- 錯誤訊息
- retry 或返回登入的選項

### 建議區分

- network / backend unavailable
- restore failed
- unknown auth state

---

## Route Guard 規則

### ProtectedRoute

應只根據 auth source of truth 做判斷：

- `booting` → render loading
- `anonymous` → redirect login
- `authenticating` / `refreshing` → render pending state
- `authenticated` → render route
- `forbidden` → render forbidden page
- `error` → render error page 或 fallback

### 角色型路由

如果頁面要求特定 role：

- 先確認是否 `authenticated`
- 再做 role 檢查
- role 不符應顯示 `forbidden`
- 不應直接 logout

---

## API 錯誤映射建議

| HTTP / 狀況 | dashboard auth 狀態 | UI 建議 |
|-------------|---------------------|---------|
| `400` login validation error | `anonymous` | 顯示表單錯誤 |
| `401` login failed | `anonymous` | 顯示帳密錯誤 |
| `401` access token expired + refresh success | `authenticated` | 無感恢復 |
| `401` refresh token invalid | `anonymous` | 回 login |
| `403` role mismatch | `forbidden` | 顯示 forbidden page |
| network / `5xx` during restore | `error` | 顯示暫時不可用與 retry |

---

## 驗收清單

1. 重整頁面後不應先看到 login 再跳回 app
2. refresh 成功時，使用者不應無故被登出
3. refresh 暫時失敗時，不應一律清 session
4. `403` 頁面應顯示為 forbidden，而不是 logout
5. login form 至少有：
   - idle
   - pending
   - validation error
   - credential error
   - backend unavailable

---

## 後續可接文件

建議接著補：

1. `dashboard-channel-config-state-inventory.md`
2. `docs/tachimint/home-state-inventory.md`
