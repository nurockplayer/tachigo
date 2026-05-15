# Extension 導航架構 + 功能 MVP — 實作計劃

狀態：規劃中

設計來源：[docs/superpowers/specs/2026-05-16-extension-navigation-and-mvp-design.md](../docs/superpowers/specs/2026-05-16-extension-navigation-and-mvp-design.md)（v4 定稿，通過三輪 Codex review）

---

## 背景

把 `apps/extension` 從 demo-state 原型重構成可擴充的兩層導航骨架，並接出功能 MVP：
Twitch 登入（自動建 tachigo 帳號）→ 螃蟹單角色挖礦累點 → DB-only claim 成 $TACHI →
DB-only 兌換 Tachiya 折價券。詳細設計、決策依據、Codex review 處置見 spec。

## 架構決策（摘要，完整見 spec）

- 兩層導航：`scene` 狀態機 + `overlayStack` 疊層，自訂 `NavigationProvider`，不引入 router。
- MVP 只支援 Twitch 站內 panel，登入走 `onAuthorized` extension JWT。
- MVP $TACHI 經濟全 DB-only（claim / redeem 不上鏈、不需 wallet）。
- 流式佈局（寬 320–430、彈性高）。
- 後端 contract-first：B1→B2→B3 須先 merge 進 `develop`，前端 F 系列才開。

## PR 拆分總表

| # | Issue 標題 | 依賴 | 預估行數 |
|---|---|---|---|
| B1 | `[backend] extension JWT 登入自動建帳號` | — | ~250 |
| B2 | `[backend] MVP DB-only $TACHI claim 路徑` | B1 | ~300 |
| B3 | `[backend] MVP DB-only coupon 兌換 + redemption ledger` | B2 | ~350 |
| F1 | `[frontend] extension 導航骨架` | — | ~380（可拆 F1a/F1b） |
| F2 | `[frontend] entry 主視覺畫面` | F1 | ~200 |
| F3 | `[frontend] Login with Twitch + identity share` | F1, B1 | ~350 |
| F4 | `[frontend] 真實挖礦（螃蟹單角色）` | F1, B1 | ~380（可拆 a–d） |
| F5 | `[frontend] 真實點數 claim` | F1, B2 | ~250 |
| F6 | `[frontend] 真實 Tachiya 折價券兌換` | F1, B3 | ~280 |

B1/F1 可並行開工（F1 無後端依賴）。F2–F6 需 F1 + 對應後端 merge 後才開。

---

## B1 — extension JWT 登入自動建帳號

**目標**：`LoginWithExtension` 查無 Twitch 連結時自動建 tachigo 帳號。

**檔案**
- Modify：`services/api/internal/services/extension_service.go`（`LoginWithExtension`）
- 參照：`services/api/internal/services/auth_service.go`（`upsertOAuthUser` find-or-create）
- Test：`services/api/internal/services/extension_service_test.go`

**待實作 checklist**
- [ ] `claims.UserID` 為空（未分享 identity）→ 維持回 `ErrInvalidExtJWT`
- [ ] 查無 `AuthProvider{ProviderTwitch, claims.UserID}` → 在單一 transaction 內
      find-or-create `User` + `AuthProvider`，再發 token pair
- [ ] username 用 deterministic 規則（`twitch_<userID>`）或保持 nil；不可用 opaque id
- [ ] 並發登入：unique constraint 衝突後重新查既有 user（conflict recovery）
- [ ] 確認 fresh dev DB 會建立 `auth_providers(provider, provider_id)` partial unique
      index（來自 migration `014_auth_provider_partial_unique.sql`）
- [ ] 補 concurrent login 測試

**驗證**：`docker compose run --no-deps --rm app go test ./...`；新使用者首次登入成功建帳；
並發登入不產生重複 user / provider。

---

## B2 — MVP DB-only $TACHI claim 路徑

**目標**：不需 wallet、不上鏈的 claim path。

**檔案**
- Create：`services/api/internal/models/tachi_balance_transaction.go`
- Modify：`services/api/internal/services/claim_service.go`（新增 DB-only path）、
  `cmd/server/main.go`（AutoMigrate 註冊新 model）、`router/router.go`（若新增路由）
- 參照：`claim_service.go` 既有 `FOR UPDATE` 鎖帳、`finalizeClaim` upsert
- Test：`services/api/internal/services/claim_service_test.go`

**待實作 checklist**
- [ ] 新 model `TachiBalanceTransaction`：`user_id` / `delta` / `source`(enum:
      `claim_db` / `redeem_db` / 預留 on-chain) / `balance_after` / `reference_type` /
      `reference_id` / `created_at`
- [ ] 新增 DB-only claim：**單一 DB transaction 內**原子完成——鎖 `points_ledgers`、
      扣 `spendable_balance`、寫 `points_transactions`、upsert `tachi_balances`、
      寫 `tachi_balance_transactions`
- [ ] 餘額不足回明確錯誤、不扣款
- [ ] 與既有 on-chain `Claim` 並存，MVP 走新 path
- [ ] swagger annotation + `swag init`（若新增/改路由）
- [ ] 單元測試：原子性、餘額不足、帳本記錄正確

**驗證**：後端測試全綠；claim 後 `spendable_balance` 減、`tachi_balances.balance` 增、
`tachi_balance_transactions` 有對應稽核列。

---

## B3 — MVP DB-only coupon 兌換 + redemption ledger

**目標**：不需 wallet、不上鏈的兌券 path。

**檔案**
- Create：`services/api/internal/models/coupon_redemption.go`、
  `services/api/internal/handlers/coupon_handler.go`
- Modify：`services/api/internal/services/spend_service.go`（新增 DB-only path）、
  `cmd/server/main.go`、`router/router.go`
- Test：`spend_service_test.go`、handler 測試

**待實作 checklist**
- [ ] 新 model `CouponRedemption`：`user_id` / `coupon_id` / `amount` / `status`
      (`pending`/`reserved`/`completed`/`failed`) / `voucher_code` / `idempotency_key`；
      `idempotency_key` 或 `(user_id, coupon_id)` 加 unique constraint
- [ ] status flow：先 DB 建 `pending/reserved` redemption + 扣 `tachi_balances.balance`
      + 寫 `tachi_balance_transactions(source=redeem_db)` → call Tachiya → 成功補
      `voucher_code` 標 `completed`；失敗回補餘額或標可重試
- [ ] 同券重複兌換被 unique constraint 擋下、不重複扣款
- [ ] endpoint 用產品語意（如 `POST /extension/coupons/redeem`）
- [ ] swagger annotation + `swag init`
- [ ] 單元測試：idempotency、Tachiya 失敗補償、餘額不足

**驗證**：後端測試全綠；兌換成功有 `coupon_redemptions` 紀錄與 voucher；重複兌換被擋；
Tachiya 失敗不出現「已扣款無 voucher」。

**B3 計劃須先拍板**（spec §9）：coupon 兌換頻率政策（一生一次 / 每日 / idempotency key）、
Tachiya 成功但 DB 失敗時的 source of truth。

---

## F1 — extension 導航骨架

**目標**：兩層導航 `NavigationProvider`，既有畫面接線，新畫面 placeholder。

**檔案**
- Create：`apps/extension/src/app/navigation/{types,NavigationProvider,useNavigation}.ts(x)`、
  `SceneRenderer.tsx`、`OverlayHost.tsx`、各 placeholder 元件
- Modify：`apps/extension/src/app/App.tsx`、`apps/extension/src/extension/{types,storage}.ts`
- Test：navigation reducer 測試、storage 遷移測試

**待實作 checklist**（spec §5、§6）
- [ ] `types.ts`：`Scene` / `Overlay` / `OverlayEntry`（discriminated union）/ `NavState`
- [ ] `NavigationProvider`（context + reducer）：`goScene` / `pushOverlay` /
      `popOverlay` / `closeAllOverlays` / `setFlag`
- [ ] push 去重：kind + params 皆同且在頂端才去重；`goScene` 清 `overlayStack`
- [ ] 開機分流：依 `flags.hasCompletedLogin`（首次 `entry`、回訪 `loading`→`mining`）；
      後端 401 清 `hasCompletedLogin`
- [ ] `SceneRenderer` + `OverlayHost`（backdrop / 返回 = `popOverlay`）
- [ ] 改寫 `App.tsx`：移除扁平 `screen` state；App shell 改流式（320–430、彈性高）
- [ ] `storage.ts`：移除 `DemoState.screen`、新增 `flags`；key 升 `app-state.v3`、舊資料 sanitize
- [ ] 既有畫面接線：Login/Loading/Mario→scene；Claim/Coupon/Raffle→overlay
- [ ] 8 個新畫面 placeholder（流式、i18n key 或標 dev-only）
- [ ] `menu` 齒輪 hub：真按鈕 `pushOverlay` 到子面板
- [ ] dev 導航列以 `import.meta.env.DEV` gate
- [ ] reducer 測試（轉場、疊層、去重含 params、清疊層、401 清 flag）+ storage 遷移測試

**內部可拆**：F1a（navigation reducer + types + storage + 測試）、
F1b（SceneRenderer/OverlayHost + App 接線 + shell 流式 + placeholder + dev gate）。

**驗證**：`pnpm test`；dev 導航列走遍所有 scene/overlay；production build 無 dev 列。

---

## F2 — entry 主視覺畫面

**目標**：01_First page 靜態主視覺 + press-to-enter。

**檔案**：Create `apps/extension/src/app/scenes/EntryScene.tsx`；Modify placeholder 換實作。

**待實作 checklist**
- [ ] 靜態主視覺（鯨魚 + 水底場景 + TACHIGO logo），流式佈局
- [ ] 點任意處 → `goScene('login')`
- [ ] i18n 文案

**驗證**：`pnpm test`；首次開啟顯示 entry、點擊進 login。

---

## F3 — Login with Twitch + identity share

**目標**：02 畫面，Login with Twitch 主路徑。

**檔案**
- Modify：`apps/extension/src/app/components/LoginScreen.tsx`、
  `apps/extension/src/hooks/useTwitch.ts`、`apps/extension/src/types/twitch-ext.d.ts`、
  `apps/extension/src/services/api.ts`

**待實作 checklist**
- [ ] 「Login with Twitch」按鈕 → 取得 extension JWT → 呼叫 B1 自動建帳號 → 抓 `channelId`
- [ ] 補 Twitch helper action 型別（`requestIdShare`）到 `twitch-ext.d.ts`
- [ ] `useTwitch` 登入失敗分類：identity 未分享 / JWT 無效 / 後端不可用，各對應 UX
- [ ] 成功設 `flags.hasCompletedLogin = true` → `goScene('loading')`
- [ ] 帳密表單 / Sign Up / Forgot Password render 但不接線
- [ ] LoginScreen 內部 `width: 320` 改流式
- [ ] 元件測試

**驗證**：`pnpm test`；按 Login with Twitch → 引導分享 identity → 自動建帳號 → 進 loading。

---

## F4 — 真實挖礦（螃蟹單角色）

**目標**：`mining` 接真實後端，汰除 demo state。

**檔案**：Modify `apps/extension/src/app/components/MarioHUD.tsx`、相關 hooks、
`apps/extension/src/extension/types.ts`（移除 `HudDemoState`）

**待實作 checklist**
- [ ] 接真實 heartbeat / click / points / balance（`useHeartbeat` / `useClickBoost` / `useTPoint`）
- [ ] 單一螃蟹角色、無進化、無其他角色
- [ ] 汰除 `HudDemoState` 與 demo 累積邏輯
- [ ] capybara 美術換螃蟹（正式美術未到位前用 placeholder）
- [ ] MarioHUD 內部 `width: 320` 改流式
- [ ] 顯示正在觀看的頻道（`channelId`）
- [ ] 元件測試

**內部可拆**：(a) balance/點數顯示接線、(b) click/heartbeat 互動、(c) 移除 `HudDemoState`、
(d) 螃蟹美術 + 流式。

**驗證**：`pnpm test`；掛機 / 點擊真實累點，重開數值一致。

---

## F5 — 真實點數 claim

**目標**：`claim` overlay 接 B2 DB-only claim path。

**檔案**：Modify `apps/extension/src/app/components/ClaimPanel.tsx`、`services/api.ts`

**待實作 checklist**
- [ ] 接 B2 DB-only claim endpoint → $TACHI
- [ ] 統一走 `runWithAuthRecovery`
- [ ] 餘額不足顯示明確錯誤、不扣款
- [ ] ClaimPanel 內部 `width: 320` 改流式
- [ ] 移除 demo CPC→TCG 換算
- [ ] 元件測試

**驗證**：`pnpm test`；claim 成功 $TACHI 增加；餘額不足有錯誤。

---

## F6 — 真實 Tachiya 折價券兌換

**目標**：`shop` overlay 接 B3 DB-only 兌換 path。

**檔案**：Modify `apps/extension/src/app/components/CouponShopPanel.tsx`、
`apps/extension/src/services/api.ts`（`redeemCoupon`）

**待實作 checklist**
- [ ] 接 B3 `POST /extension/coupons/redeem`
- [ ] `redeemCoupon()` 改用 tachigo access token + `runWithAuthRecovery`，移除手動 Bearer
- [ ] 只開放 Tachiya 商城折價券
- [ ] 重複兌換 / 餘額不足 / Tachiya 失敗的 UX
- [ ] CouponShopPanel 內部 `width: 320` 改流式
- [ ] 移除 demo catalog 依賴
- [ ] 元件測試

**驗證**：`pnpm test`；兌換成功扣 $TACHI、得 voucher；重複兌換被擋。

---

## 端到端驗證（MVP 全部 merge 後）

對照 spec §10：首次 entry→login→自動建帳號→mining；回訪略過 entry/login；
真實累點；DB-only claim / 兌券；錯誤路徑（後端不可用、identity 未分享、餘額不足、
重複兌換、Tachiya 失敗、並發登入）。

## 置後 / #710

UI/UX 畫面（settings / account / onboarding / menu 視覺等）、#710 海洋角色系統、
裝備欄 — 見 spec §8，不在本計劃。
