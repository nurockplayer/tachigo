# Tachigo Extension — 導航架構 + 功能 MVP 設計

狀態：定稿 v4（已通過三輪 Codex review，第三輪結論「可開工」，**尚未 commit / push**）
日期：2026-05-16
相關：[discussion #710](https://github.com/nurockplayer/tachigo/discussions/710)、`plans/ocean-mining-character-system-phase1.md`

> v3 變更摘要：第二輪 Codex review 指出 `claim` 與 `redeem` 一樣需要 wallet +
> 上鏈（`claim_service.go` 做 on-chain mint）。經對照程式碼驗證屬實。已新增
> 後端 PR、補上 $TACHI 交易帳本與兌換 idempotency 設計、修正開機分流 UX、
> 釐清流式佈局的拆分範圍。完整處置見第 11 段。

---

## 1. 背景與目標

`apps/extension` 目前是以 demo state 驅動的原型：畫面用扁平的 `DemoScreen`
字串 enum 切換，資料來自 `loadDemoState` / `saveDemoState`（chrome storage 假資料）。

本設計做兩件事：

1. **導航骨架**：把扁平 screen enum 重構成兩層導航（全螢幕 scene + 疊層 overlay），
   讓後續十幾個畫面能逐張掛上、不互相牽動。
2. **功能 MVP**：把核心循環接成真實後端——Twitch 登入（自動建 tachigo 帳號）→
   實際挖礦累積點數 → claim 成 $TACHI → 用 $TACHI 換 Tachiya 商城折價券。汰除 demo state。

MVP 定位是「**最簡可用版本，先做內部測試**」：只用螃蟹單一角色、無進化、無其他角色。

**運行型態（已拍板）**：MVP **只支援 Twitch 站內 extension**，登入一律走
`onAuthorized` extension JWT。standalone（popup / sidepanel）與手機端置後，
MVP 不做 OAuth redirect 登入路徑。

**設計尺寸（已拍板）**：採**流式響應式佈局**，不寫死 px——寬度設計基準 **360**、
下限 **320**（吃得下 Twitch Panel 型 extension）、上限約 **430**（大尺寸手機）；
高度永遠流式（`min-height` + 內容可捲動）。實作用 `%` / `flex` / `clamp()`。
這樣同一份程式碼在 Twitch panel、瀏覽器側邊欄、手機 360–430 都不需重切。

**MVP $TACHI 經濟（已拍板）**：claim 與 redeem 在 MVP **皆走 DB-only 路徑**，
不需 Web3 wallet、不上鏈。on-chain mint / burn 是後續迭代，不在 MVP。

非目標：#710 海洋角色系統不在 MVP，獨立 track 排在 MVP 之後；本設計只負責「預留位置」。

---

## 2. 現況

- `apps/extension/src/app/App.tsx`：扁平 `DemoScreen = 'login'|'loading'|'hud'|'claim'|'coupon'|'raffle'`，
  條件渲染，底部一排 **永遠 render** 的 dev 導航鈕（非 dev-only）。frame 寫死 320。
- 既有畫面元件：`LoginScreen`、`LoadingScreen`、`MarioHUD`（挖礦主頁，capybara）、
  `ClaimPanel`、`CouponShopPanel`、`RaffleResultPanel`、`LanguageSwitcher`。
  → `ClaimPanel` 仍是 demo CPC→TCG 換算；`CouponShopPanel` 用 demo catalog。
  → `LoginScreen` / `MarioHUD` / `ClaimPanel` 等內部都寫死 `width: 320`（不只 App frame）。
- 既有 hooks：`useTwitch`（`onAuthorized` JWT → 後端換 token；目前把所有 login
  失敗都壓成 `Backend unavailable`，未區分 identity 未分享 / JWT 無效）、
  `useHeartbeat`、`useClickBoost`、`useTPoint`、`useRaffleResult`。
- 後端（`services/api/internal/`）關鍵事實（已對照程式碼確認）：
  - `extension_service.go` `LoginWithExtension`：**不會建帳號**，查不到 `ProviderTwitch`
    連結回 `ErrUserNotFound`；`claims.UserID` 為空（未分享 identity）回 `ErrInvalidExtJWT`。
  - `auth_service.go` `upsertOAuthUser`（OAuth callback 路線）：**會** find-or-create user。
    `users.email` / `username` 皆可為 nil。`auth_providers(provider, provider_id)` 的
    unique index 主要在 migration 建立，AutoMigrate 未必明確建出（須確認 fresh DB）。
  - `claim_service.go` `Claim`：`resolveWalletAddress` → `MintBroadcastOnChain` →
    `WaitMintReceiptOnChain` → 寫 `tachi_balances.balance`。**需 wallet + 上鏈 mint**。
  - `spend_service.go` `Redeem`：`resolveWalletAddress` + `BurnOnChain`。**需 wallet + 上鏈 burn**。
  - `models/tachi_balance.go` `TachiBalance`：單列 aggregate（`UserID uniqueIndex` + `Balance`），
    **無 $TACHI 交易流水、無 coupon 兌換記錄**。
  - 路由：`/extension/auth/login`、`/extension/watch/*`、`/api/v1/users/me/points/claim`、
    `/api/v1/spend/redeem`。
- 前端 `redeemCoupon()` 繞過 `runWithAuthRecovery`、手動塞 `Authorization: Bearer ${token}`。
- Figma `Tachigo Prototype v1`：01 主視覺、02 登入、02-1 註冊、02-2 忘記密碼、
  03 角色選擇、04 挖礦主頁、05 claim、06 coupon market。

---

## 3. 設計決策：兩層導航（不引入 router）

- **A（採用）兩層導航**：`scene` 狀態機 + `overlayStack` 疊層。自訂 `NavigationProvider`
  （context + reducer），不引入 router lib。
- B Hash router：Twitch Extension iframe 無有意義 URL / history，疊層無法乾淨對應 route。
- C 擴充扁平 enum：無疊層、無 stack、無返回，面板長到 10+ 會腐爛。

採用 A：貼合無 URL 環境、輕量、返回鍵語意清楚、好持久化、好測試。

---

## 4. 畫面盤點

### Scenes（全螢幕流程）

| scene | 對應 | 現況 | MVP? |
|---|---|---|---|
| `entry` | 程式入口主視覺（press-to-enter） | 🆕 新增 | ✅ |
| `login` | 登入（Login with Twitch 為主） | 改造 `LoginScreen` | ✅ |
| `loading` | 認證 / 串接畫面（含 error / retry） | 沿用 `LoadingScreen`（擴充狀態） | ✅ |
| `character-select` | 角色選擇畫面 | 🆕 placeholder | ⏸ #710 |
| `mining` | 挖礦主頁面（螃蟹） | 沿用 `MarioHUD` | ✅ |

### Overlays（疊在當前 scene 上）

| overlay | 對應 | 現況 | MVP? |
|---|---|---|---|
| `claim` | 點數 claim → $TACHI | 沿用 `ClaimPanel`（需接線） | ✅ |
| `shop` | Tachiya 折價券商店 | 沿用 `CouponShopPanel`（需接線） | ✅ |
| `raffle-result` | 抽獎結果（需 `raffleId` param） | 沿用 `RaffleResultPanel` | ✅（既有） |
| `menu` | 齒輪 hub 選單 | 🆕 骨架接線、視覺置後 | ⏸ |
| `account` | 帳號角色資訊 | 🆕 placeholder | ⏸ |
| `settings` | 語言/畫面/音效/特效/HUD 開關 | 🆕 placeholder | ⏸ |
| `character-switch` | 角色變換（≈ #710 CharacterMenu） | 🆕 placeholder | ⏸ #710 |
| `collection` | 圖鑑 | 🆕 placeholder | ⏸ #710 |
| `missions` | 任務 | 🆕 placeholder | ⏸ #710 Phase 2 |
| `equipment` | 裝備欄 | 🆕 placeholder | 🧊 Icebox |
| `onboarding` | 首次 mining 新手導覽 | 🆕 placeholder | ⏸ |

### 已確認的設計細節

- `character-select` (scene) 與 `character-switch` (overlay) 共用 `CharacterPicker` 元件。
- `entry` 是品牌主視覺，點任意處 → `login`；之後可做動態版（Icebox）。
- `login` scene 內含登入 / 註冊 / 忘記密碼三子畫面，由 scene 元件 local state 管，
  不佔全域 scene。MVP 只有「Login with Twitch」功能可用；其餘先 render、不接線。
- 齒輪 = 選單 hub：點齒輪出 `menu` overlay，`pushOverlay` 到子面板。
- `mining` 頁的 claim / shop 入口在 MVP 用直接按鈕，不依賴齒輪 hub。

---

## 5. 導航骨架資料結構

```ts
// src/app/navigation/types.ts
type Scene   = 'entry' | 'login' | 'loading' | 'character-select' | 'mining'
type Overlay =
  | 'claim' | 'shop' | 'raffle-result' | 'menu'
  | 'account' | 'settings' | 'character-switch'
  | 'collection' | 'missions' | 'equipment' | 'onboarding'

type OverlayEntry =
  | { kind: 'raffle-result'; params: { raffleId: string } }
  | { kind: Exclude<Overlay, 'raffle-result'>; params?: undefined }

interface NavState {
  scene: Scene
  overlayStack: OverlayEntry[]          // 後進先出
  flags: {
    hasCompletedLogin: boolean          // 曾成功登入 → 開機可略過 entry/login
    onboardingVersion: number           // 版本化；> 已看版本才重跳 onboarding
    selectedCharacterOnce: boolean      // 預留給 #710
  }
}
```

- **`NavigationProvider`**（context + `useReducer`）提供
  `goScene` / `pushOverlay(kind, params?)` / `popOverlay` / `closeAllOverlays` / `setFlag`。
- **push 去重**：僅當 `kind` **且 `params` 皆相同**、且位於 stack 頂端時才不重複堆疊
  （防 menu 疊 menu）；同 `kind` 但 `params` 不同 → 視為新 entry 照常 push
  （例如 raffle A 已在頂端、push raffle B 不可被吃掉）。
- `goScene` 會清空 `overlayStack`。
- **渲染**：`App.tsx` → `<SceneRenderer>` 打底 + `<OverlayHost>` 由下往上疊；
  每個 overlay 自帶半透明 backdrop，點 backdrop / 返回鍵 = `popOverlay`。
- **持久化**：只把 `flags` 寫進 storage；`scene` / `overlayStack` 不持久化。
  storage key 從 `tachigo.sidepanel.demo-state.v2` 升為 `tachigo.sidepanel.app-state.v3`，
  舊資料 sanitize 後丟棄不相容欄位。

### 開機分流（auth 驅動，修正 v2 的 UX 矛盾）

`api.ts` 無持久後端 token（access token 只存 module global、`useTwitch` unmount
即清）。開機流程依 `flags.hasCompletedLogin` 分兩種：

- **首次使用（`hasCompletedLogin = false`）**：`scene = 'entry'`。背景仍等
  `onAuthorized` 預熱 extension JWT，但**不自動進 mining**——使用者必須走
  `entry` → `login` → 按「Login with Twitch」才轉場。即使 identity 已分享、
  背景已能登入，首次仍要看 entry/login（產品已確認首次需引導）。
- **回訪（`hasCompletedLogin = true`）**：`scene = 'loading'`，等 `onAuthorized`
  → 後端 extension 登入：成功 → `goScene('mining')`；失敗 → `loading` 顯示
  error / retry（沿用 `useTwitch` 15s retry），不退回 entry。
- 後端登入回 401 / B1 回錯 → `flags.hasCompletedLogin` 清為 false（列入 reducer
  / storage 測試）。
- `loading` scene 必須涵蓋 authorizing / error / retry 三種狀態。

### Scene 轉場規則（MVP）

| 從 | 事件 | 到 |
|---|---|---|
| `entry` | 點任意處 | `login` |
| `login` | Login with Twitch 成功（含後端 B1 自動建帳號） | `loading` → `mining`，並設 `hasCompletedLogin=true` |
| `loading` | extension 登入成功 | `mining` |
| `mining` | 點 claim / shop 按鈕 | `pushOverlay('claim' \| 'shop')` |
| `mining` | 點齒輪 | `pushOverlay('menu')` |
| 任一 overlay | backdrop / 返回 | `popOverlay` |

---

## 6. 骨架實作範圍（F1）

**會寫的：**

1. 新增 `src/app/navigation/`：`types.ts`、`NavigationProvider.tsx`、`useNavigation.ts`。
2. `SceneRenderer`、`OverlayHost` 兩個渲染元件。
3. 改寫 `App.tsx`：移除扁平 `screen` state，改用 `NavigationProvider`；
   **App 外層 shell** 改成流式（寬 320–430、彈性高）。
4. 擴充 `extension/storage.ts` / `types.ts`：移除 `DemoState.screen`，新增 `flags`；
   storage key 升 `v3`，舊資料 sanitize。
5. 既有畫面接線（內容不動）：`LoginScreen`→`login`、`LoadingScreen`→`loading`、
   `MarioHUD`→`mining`；`ClaimPanel`→`claim`、`CouponShopPanel`→`shop`、
   `RaffleResultPanel`→`raffle-result`（`onBack`→`popOverlay`，raffle 帶 `raffleId`）。
6. 新畫面建 placeholder（標題 + 返回鍵；文案走 i18n locale key 或標 dev-only）：
   `entry`、`character-select`、`account`、`settings`、`collection`、`missions`、
   `equipment`、`onboarding`。**新 placeholder 一律用流式佈局。**
7. `menu` 齒輪 hub：骨架就要能動——真按鈕，`pushOverlay` 到對應子面板。
8. dev 導航列：以 `import.meta.env.DEV` gate，production build 不 render（完成條件）。
9. 測試：reducer 單元測試（轉場、疊層、push 去重含 params、`goScene` 清疊層、
   401 清 `hasCompletedLogin`）+ storage 遷移測試。

**流式佈局的範圍界定**：F1 只負責 **App 外層 shell + 新 placeholder** 流式。
既有元件（`LoginScreen` / `MarioHUD` / `ClaimPanel` / `CouponShopPanel`）內部仍
寫死 `width: 320`；它們的流式改造**併入各自的 MVP PR**（login→F3、mining→F4、
claim→F5、shop→F6）一起做，不在 F1 retrofit 全部元件（避免 F1 爆量）。

**內部拆分建議**（避免單 PR 超過 CLAUDE.md 400 行軟門檻）：
- F1a：`navigation/` reducer + types + storage 遷移 + 測試。
- F1b：`SceneRenderer` / `OverlayHost` + `App.tsx` 接線 + shell 流式 + placeholder + dev gate。

**不在 F1：** 任何新畫面真實 UI / 內容、真實後端串接、既有元件內部流式改造、#710。

---

## 7. MVP 分期（後端 contract-first，再前端）

依 CLAUDE.md「PR 不得依賴未 merge 的 PR」，後端 PR 須先 merge 進 `develop`。

### 後端

| PR | 標題 | 內容 |
|---|---|---|
| B1 | `[backend] extension JWT 登入自動建帳號` | 改 `LoginWithExtension`：查無 `ProviderTwitch` 連結時，用 extension JWT 的 Twitch user_id（需 identity 已分享）自動 find-or-create tachigo user + `AuthProvider`（仿 `upsertOAuthUser`）。**完成條件**：以 transaction + unique constraint 衝突回復處理並發登入；補 concurrent login 測試；確認 fresh dev DB 會建立 `auth_providers(provider, provider_id)` partial unique index——它來自 migration `014_auth_provider_partial_unique.sql`（非 model tag），B1 須確認 runtime migration / fresh setup 真的會跑到它。匿名 user 的 username 用 deterministic 規則（如 `twitch_<userID>`）或保持 nil，不可拿 opaque id 當穩定公開名稱。 |
| B2 | `[backend] MVP DB-only $TACHI claim 路徑` | 新增**不需 wallet、不上鏈** 的 claim path：扣 `points_ledgers.spendable_balance`、增 `tachi_balances.balance`。新增**可稽核的 $TACHI 交易帳本** `tachi_balance_transactions`（user_id / delta / `source` / balance_after / `reference_type` / `reference_id` / created_at），claim 與日後 redeem 共用。**驗收條件**：鎖 `points_ledgers`、扣 `spendable_balance`、寫 `points_transactions`、upsert `tachi_balances`、寫 `tachi_balance_transactions` 必須在**單一 DB transaction 內原子完成**（參照 `claim_service.go` 既有 `FOR UPDATE` 鎖帳與 `finalizeClaim` upsert）。`source` 定成可稽核 enum（至少分 `claim_db` / `redeem_db` / 未來 on-chain）。與既有 on-chain `Claim` 並存，MVP 走新 path。 |
| B3 | `[backend] MVP DB-only coupon 兌換 + redemption ledger` | 新增不需 wallet、不上鏈 的兌券 path。新增 `coupon_redemptions` 表（user_id / coupon_id / amount / status / voucher_code / **idempotency_key**），以 unique constraint 擋同券重複扣款。**Tachiya 發券是外部副作用，無法與 DB transaction 原子化**——status flow：先在 DB 建 `pending/reserved` redemption + 扣 `tachi_balances.balance` + 寫 `tachi_balance_transactions(source=redeem_db)`，再 call Tachiya，成功補 `voucher_code` 並標 `completed`，失敗則回補餘額或標記可重試（B3 計劃須拍板）。endpoint 用產品語意（如 `/extension/coupons/redeem`）。 |

### 前端

| PR | 標題 | 內容 | 依賴 |
|---|---|---|---|
| F1 | `[frontend] extension 導航骨架` | 第 5、6 段（可內部拆 F1a/F1b） | — |
| F2 | `[frontend] entry 主視覺畫面` | 01_First page 靜態主視覺 + press-to-enter → `login`；流式 | F1 |
| F3 | `[frontend] Login with Twitch + identity share` | 02 畫面；「Login with Twitch」主路徑 → 取得 extension JWT → 呼叫 B1 自動建帳號 → 抓 `channelId`。**需補 Twitch helper action 型別（`requestIdShare`）、把 `useTwitch` 的 login 失敗分類（identity 未分享 / JWT 無效 / 後端不可用）並對應 UX**。帳密表單 / Sign Up / Forgot 先 render、不接線。元件流式改造一併做 | F1, B1 |
| F4 | `[frontend] 真實挖礦（螃蟹單角色）` | `mining` 接真實 heartbeat/click/points/balance；單一螃蟹、無進化；汰除 `HudDemoState`；capybara 美術換螃蟹（正式美術未到位前用 placeholder）；元件流式改造 | F1, B1 |
| F5 | `[frontend] 真實點數 claim` | `claim` overlay 接 B2 的 DB-only claim path → $TACHI；統一走 `runWithAuthRecovery`；元件流式改造 | F1, B2 |
| F6 | `[frontend] 真實 Tachiya 折價券兌換` | `shop` overlay 接 B3 的 DB-only 兌換 path；`redeemCoupon()` 改用 tachigo access token + `runWithAuthRecovery`，移除手動 Bearer；元件流式改造 | F1, B3 |

**F4 內部拆分建議**：(a) 真實 balance / 點數顯示接線、(b) click/heartbeat 互動 UI、
(c) 移除 `HudDemoState`、(d) 螃蟹美術替換 + 流式。

PR 順序即依賴順序；每張預估 < 400 行，超過於 issue 註明再拆。

---

## 8. 置後 / #710 同步 / Icebox

**⏸ 置後 — UI/UX 與帳號功能（MVP 內測後）**

`login` 帳密自訂 / 註冊 / 忘記密碼接線、`onboarding` 新手導覽、`menu` 齒輪 hub
視覺、`settings` 設定面板、`account` 帳號資訊面板、`entry` 動態化、
**$TACHI claim / redeem 的 on-chain mint / burn 版本**（MVP 是 DB-only）。

**🌊 #710 海洋角色系統 — 獨立 track，排 MVP 之後**

- MVP 鎖螃蟹單角色；#710 落地才啟用四角色、進化、buff，填上 `character-select`
  scene、`character-switch` / `collection` overlay、首次選角分流。
- `missions` 任務面板依 #710 Phase 2。
- 骨架的 `Scene` / `Overlay` enum 已預留位置。**`flags` 不承諾「#710 完全不動骨架」**
  ——#710 會需要 server-driven active character、ownership、cooldown 等狀態，
  屆時 `flags` 與 overlay params 會擴充。骨架保證的是「scene/overlay 不必新增、
  reducer 形狀穩定」。
- 既有計劃文件：`plans/ocean-mining-character-system-phase1.md`。

**🧊 Icebox**：`[discussion] 裝備欄系統`、`entry` 主視覺動態化。

---

## 9. 待確認問題

均屬 B1/B2/B3/F3 issue 階段可定的實作細節，不擋 spec 定稿與 writing-plans：

- [ ] B1：extension JWT payload 是否含 Twitch 顯示名稱 / email？若無，自動建立的
      user 用 `twitch_<userID>` 或保持 nil（issue 階段定）。
- [ ] B3：同一張 coupon 的兌換頻率政策——一生一次 / 每日一次 / 每個 idempotency key
      一次 / 依 Tachiya voucher rule？直接決定 `coupon_redemptions` unique 鍵設計。
- [ ] B2：`tachi_balance_transactions` 是否保留 `balance_before`（v4 只列 `balance_after`，
      可運作；補 `balance_before` 稽核體驗較佳）——B2 issue 階段定。
- [ ] B3：Tachiya voucher 發放成功但 DB 回寫失敗時，以 Tachiya 為準還是 tachigo DB
      為準？需定 rollback / pending / 補償策略。
- [ ] 使用者未同意 Twitch identity share 時，`login` 的 CTA 文案（F3 issue 定）。
- [ ] 螃蟹美術資產由設計提供；MVP 內測先用 placeholder。
- [ ] `extensions/tachigo-demo-sidepanel` 是否同步 / 廢棄，不在本範圍。

---

## 10. 驗證方式

**骨架（F1）**
1. reducer 單元測試：scene 轉場、疊層、push 去重（含 params 不同不去重）、
   `goScene` 清疊層、401 清 `hasCompletedLogin`。
2. storage 遷移測試：舊 `v2`（含 `screen`）可被 sanitize 成新 `v3`。
3. 手動：dev 導航列走遍所有 scene/overlay；placeholder 返回鍵正常；
   production build 不出現 dev 導航列。

**MVP（B1/B2/B3 + F2~F6）端到端**
4. 開啟 extension（首次）→ `entry` → 點擊 → `login`。
5. 按「Login with Twitch」→ 引導分享 identity → 後端 B1 自動建立 tachigo 帳號
   → `loading` → `mining`，`hasCompletedLogin` 設為 true。
6. 回訪重開 → 略過 entry/login，`loading` → `mining`。
7. `mining` 顯示正在觀看的頻道（`channelId`），螃蟹挖礦。
8. 掛機 / 點擊 → 真實點數累積（非 demo state），重開 app 數值一致。
9. 開 `claim` → 經 B2 DB-only path：`spendable_balance` 扣除、`tachi_balances.balance`
   增加、$TACHI 交易帳本有記錄；**不需 wallet、不上鏈**。
10. 開 `shop` → 經 B3 DB-only path 兌換 Tachiya 折價券 → $TACHI 扣除、
    `coupon_redemptions` 有記錄；不需 wallet、不上鏈。

**錯誤路徑（MVP 必測）**
11. 後端不可用 → `loading` 顯示錯誤 + 重試。
12. extension JWT 無效 / identity 未分享 → `login` 顯示對應分類 CTA（非籠統「Backend unavailable」）。
13. claim / 兌換時 $TACHI 餘額不足 → 面板顯示明確錯誤、不扣款。
14. 同一張 coupon 重複兌換 → `idempotency_key` / unique constraint 擋下、不重複扣款。
15. Tachiya API 發券失敗 → 依 B3 定的策略 rollback / pending，DB 不出現「已扣款但無 voucher」。
16. 並發 extension 登入（同一新使用者）→ B1 不建立重複 user / provider。

---

## 11. Codex review 處置紀錄

### 第一輪

| Codex 指出 | 驗證 | 處置 |
|---|---|---|
| Blocker：`/extension/auth/login` 不會自動建帳號 | 屬實 | 新增 B1 |
| Blocker：`spend/redeem` 需 wallet + 上鏈 burn | 屬實 | 新增 DB-only 兌換路徑（v3 為 B3） |
| Major：iframe OAuth redirect 不宜當主路徑 | 採納 | MVP 只走 `onAuthorized` JWT |
| Major：`overlayStack` 缺 params | 屬實 | 改 `OverlayEntry` discriminated union |
| Major：開機分流無持久 token | 屬實 | 改 auth 驅動分流 |
| Major：F1 / F4 PR 偏大 | 採納 | 加內部拆分建議 |
| Major：viewport 320 vs 360 | 屬實 | 改流式佈局（320–430） |
| Major：`flags` 太薄 | 採納 | 加 `onboardingVersion`；改寫骨架保證範圍 |
| Minor 多項 | 採納 | storage key v3、placeholder i18n、dev gate、錯誤路徑驗證 |

### 第二輪

| Codex 指出 | 驗證 | 處置 |
|---|---|---|
| Blocker：`claim` 同樣需 wallet + 上鏈 mint，F5 仍走不通 | 屬實（`claim_service.go` 做 on-chain mint） | claim 改 DB-only，新增 B2；MVP $TACHI 經濟明確全 DB-only（第 1 段） |
| Major：B2 缺 $TACHI 交易帳本與兌換記錄 | 屬實（`TachiBalance` 僅單列 aggregate） | B2 加 `tachi_balance_transactions`；B3 加 `coupon_redemptions` + `idempotency_key` + unique constraint |
| Major：B1 並發建帳 race / unique | 屬實 | B1 完成條件加 transaction + 衝突回復 + concurrent 測試 + fresh DB index 確認 |
| Major：開機分流與 entry/login UX 矛盾 | 採納 | 第 5 段依 `hasCompletedLogin` 分流：首次必看 entry/login |
| Major：OverlayEntry 去重會吃掉不同 params | 採納 | 去重改為 kind + params 皆同才去重 |
| Major：F1 仍偏大、元件內部寫死 320 | 屬實 | 第 6 段界定流式範圍：F1 只做 shell + placeholder，元件流式併入各 MVP PR |
| Major：F3 缺 `requestIdShare` 型別、登入錯誤未分類 | 屬實 | F3 加 Twitch helper 型別 + 錯誤分類 + 重新授權 UX |
| Minor：deterministic username、401 清 flag、endpoint 命名、Tachiya 失敗 rollback 驗證 | 採納 | 納入 B1 / 第 5 段 / B3 / 第 10 段 |

### 第三輪

結論：**無 blocker，可開工**。剩餘 major 為 B1/B2/B3 計劃文件須鎖死的驗收條件，
已硬化進第 7 段：

| Codex 指出 | 處置 |
|---|---|
| B2 須單一 DB transaction 原子完成 | B2 row 加「驗收條件」明列原子範圍與參照 `claim_service.go` |
| B3 Tachiya 外部副作用無法原子化，須定 status flow | B3 row 加 `pending/reserved → call Tachiya → completed/補償` 流程 |
| B1 fresh DB unique index 來自 migration `014_auth_provider_partial_unique.sql` | B1 row 補明 migration 來源與須確認 fresh setup 會跑到 |
| `tachi_balance_transactions` 應有可稽核 `source` enum + `reference_type`/`reference_id` | B2 row 採納，加入欄位 |
