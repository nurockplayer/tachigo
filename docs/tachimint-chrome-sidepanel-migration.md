# Tachimint Chrome Sidepanel Migration Decision

> 用途：記錄 `tachimint` 前端 migration 的已定案方向，作為後續 PR 的共同 source of truth。
> 狀態：已定案的 migration decision，不是完整實作 spec。
> 最後更新：2026-04-16

---

## 1. 本次已定案事項

`tachimint` 前端接下來採用以下方向：

1. 保留 `tachimint/` 目錄名稱，不另開長期並行的第二前端 product surface
2. 前端 runtime 由舊的 Twitch-hosted panel，遷移為 Chrome sidepanel extension
3. 身份來源在本階段仍沿用 Twitch 相關流程
4. backend contract 在本階段沿用既有 API，不於本輪重設
5. `extensions/tachigo-demo-sidepanel/` 視為 migration source，不是長期保留的正式產品入口

---

## 2. 本輪明確不做

這次 migration decision 只定義前端方向，不包含以下內容：

- 不重做 backend API contract
- 不重新設計 viewer identity / channel context 來源
- 不同步擴張到 `backend/` 或 `dashboard/`
- 不在同一輪順便做 terminology 全面重命名
- 不把 demo state 直接當成正式 domain model 定案

---

## 3. 為什麼要這樣切

目前 `tachimint` 內仍有既有產品邏輯與後端契約，例如 Twitch auth、heartbeat、claim 與相關 API wiring；另一方面，`extensions/tachigo-demo-sidepanel/` 已經提供較接近目標方向的 Chrome sidepanel shell 與新 UI。

因此這次 migration 採用的策略是：

- 保留 `tachimint/` 作為唯一正式前端 surface
- 以 `extensions/` 的 runtime 與 UI 作為 migration source
- 逐步把既有 `tachimint` 的產品邏輯接回新的 sidepanel shell

---

## 4. 後續 PR 期望拆法

後續實作以小顆粒 frontend PR 進行，原則如下：

1. 先建立 `tachimint` 的 sidepanel runtime 骨架
2. 再導入新 app shell 與視覺
3. 再把 Twitch auth、heartbeat、claim 等既有邏輯接回
4. 最後再移除 `extensions/tachigo-demo-sidepanel/` 與過時舊殼

---

## 5. 與現有命名的關係

本 decision 只處理產品 runtime 與 migration 邊界，不直接表示下列命名在本輪就要同步變更：

- `extension_jwt`
- `/extension/*` routes
- `ExtensionService`
- 其他既有 contract / schema / Swagger 用語

這些內容若要調整，應另開獨立的 contract / refactor scope 處理。
