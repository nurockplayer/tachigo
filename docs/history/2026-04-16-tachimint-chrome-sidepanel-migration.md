# Tachimint Chrome Sidepanel Migration Decision

> 用途：記錄 `tachimint` 前端 migration 的已定案方向，作為後續 PR 的共同 source of truth。
> 狀態：歷史 migration decision record，不是完整實作 spec。
> 最後更新：2026-04-16
> 最後校正：2026-05-05（#490 docs root audit）

---

## 1. 本次已定案事項

`tachimint` 前端接下來採用以下方向：

1. 保留 `apps/extension/` 作為唯一正式 extension 前端 surface，不另開長期並行的第二 extension 前端 product surface
2. 前端 runtime 由舊的 Twitch-hosted panel，遷移為 Chrome sidepanel extension
3. 身份來源在本階段仍沿用 Twitch 相關流程
4. backend contract 在本階段沿用既有 API，不於本輪重設
5. `extensions/tachigo-demo-sidepanel/` 視為 migration source，不是長期保留的正式產品入口

---

## 2. 本輪明確不做

這次 migration decision 只定義前端方向，不包含以下內容：

- 不重做 backend API contract
- 不重新設計 viewer identity / channel context 來源
- 不同步擴張到 `backend/` 或 `apps/dashboard/`
- 不在同一輪順便做 terminology 全面重命名
- 不把 demo state 直接當成正式 domain model 定案

---

## 3. 為什麼要這樣切

目前 `tachimint` 內仍有既有產品邏輯與後端契約，例如 Twitch auth、heartbeat、claim 與相關 API wiring；另一方面，`extensions/tachigo-demo-sidepanel/` 已經提供較接近目標方向的 Chrome sidepanel shell 與新 UI。

因此這次 migration 採用的策略是：

- 保留 `apps/extension/` 作為唯一正式 extension 前端 surface
- 以 `extensions/` 的 runtime 與 UI 作為 migration source
- 逐步把既有 `tachimint` 的產品邏輯接回新的 sidepanel shell

---

## 4. Implementation Slice 計畫

後續實作以小顆粒 frontend PR 進行，拆成以下 4 個 slice：

| Slice | PR | 內容 |
|---|---|---|
| 1 / 4 | #265 | assets、fonts、base styles、i18n keys、demo coupon catalog；demo state foundation（storage / sanitization） |
| 2 / 4 | #266 | LoginScreen、LoadingScreen、LanguageSwitcher、useSound、theme |
| 3 / 4 | #267 | ClaimPanel、CouponShopPanel、MarioHUD |
| 4 / 4 | #268 | App 整合、state machine 接線、Twitch entrypoint 保留 |

### Slice 1 (#265) 詳細 Scope

**搬移內容**

- `src/assets/` — logo PNG
- `src/styles/fonts.css`、`src/styles/index.css` — pixel UI 主題、字型
- `src/i18n/locales/` — en / zh-TW / zh-CN 的 loading / login / hud / nonViewer / coupon / error keys
- `src/extension/couponCatalog.ts` — 靜態 demo coupon catalog
- `src/extension/storage.ts` — Chrome storage 為主、localStorage 為 fallback mirror 的 demo state 儲存層
- `src/extension/types.ts` — HUD demo state 型別與清洗函式
- `.github/workflows/ci.yml` — 新增 `workflow-regression` job 以驗證 CI YAML 正確性（在 frontend Docker container 範圍外獨立執行）

**Storage 設計**

Chrome storage 為主要儲存，localStorage 作為持續同步的 fallback mirror：

- `saveDemoState()` Chrome 寫入成功後同步 mirror 到 localStorage（確保 Chrome read 失敗時仍可回復最新狀態）
- `saveDemoState()` Chrome 寫入失敗時 fallback 直接寫入 localStorage
- `loadDemoState()` Chrome 無資料時讀 legacy localStorage 並嘗試 migrate；migrate 成功後清除 legacy key

**明確不含**

- 不含 UI 元件（LoginScreen / HUD / ClaimPanel / CouponShopPanel）
- 不改 App 進入點或路由邏輯
- 不移除 `extensions/tachigo-demo-sidepanel/`

---

## 5. 與現有命名的關係

本 decision 只處理產品 runtime 與 migration 邊界，不直接表示下列命名在本輪就要同步變更：

- `extension_jwt`
- `/extension/*` routes
- `ExtensionService`
- 其他既有 contract / schema / Swagger 用語

這些內容若要調整，應另開獨立的 contract / refactor scope 處理。
