# Chrome Extension 名詞盤點

> 用途：盤點 repo 內 `Chrome Extension`、`Twitch Extension` 與泛用 `extension` 命名的混用情況。
> 狀態：歷史盤點文件；Chrome sidepanel migration 方向已定案，但本文件本身不是完整實作 spec。
> 最後更新：2026-04-16
> 最後校正：2026-05-05（#490 docs root audit）

---

## 1. 目前可確認的 source of truth

依目前 repo 既有文件、程式碼現況與已定案 migration decision，可先確認以下事實：

1. `tachimint` 的既有可運作實作是 **Twitch-hosted extension runtime**
2. `tachimint` 的新方向已定為 **Chrome sidepanel extension runtime**
3. 本階段前端與後端仍依賴 `window.Twitch.ext`、`extension_jwt`、Twitch helper script 等流程
4. 本輪 migration 的 decision source of truth 為 [docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md](2026-04-16-tachimint-chrome-sidepanel-migration.md)

因此，這份文件的用途是說明術語與現況的落差，協助後續 migration 拆題；它不是完整 migration spec。

---

## 2. 本輪術語收斂範圍

目前 repo 內實際混在一起的是三種不同層次：

| 類別 | 代表什麼 | 目前狀態 |
|---|---|---|
| 產品形式 | 使用者最終安裝與使用的前端形態 | 已定案為 Chrome sidepanel extension |
| 現行 runtime | 現在程式真正依賴的執行環境 | Twitch-hosted extension，將分階段遷移 |
| 模組 / API 命名 | `/extension/*`、`ExtensionService`、`extension_jwt` 等名稱 | 已存在於前後端契約與程式結構中 |

以下文件已在本輪 docs 收斂中處理：

- [docs/architecture.md](../architecture.md)
- [docs/feature-discussion.md](../feature-discussion.md)
- [docs/tokenomics.md](../tokenomics.md)
- [docs/watch-to-points-design.md](../watch-to-points-design.md)
- [apps/extension/README.md](../../apps/extension/README.md)

以下文件已另拆處理，不混在本 PR：

- [docs/sequence-diagram.md](../sequence-diagram.md) - 另拆 PR 處理，流程圖描述現況需單獨評估
- [docs/extension-ui-prompts.md](../extension-ui-prompts.md) - 已拆至 #154，避免超出 terminology cleanup / audit 邊界

---

## 3. 盤點結果

### A. 已明確依賴 Twitch runtime 的地方

這些項目不是單純改字詞就能處理：

- [apps/extension/index.html](../../apps/extension/index.html)
  - 仍載入 Twitch Extension Helper
- [apps/extension/src/mock/twitch-ext.ts](../../apps/extension/src/mock/twitch-ext.ts)
  - 本地開發使用 `window.Twitch.ext` mock
- [apps/extension/src/types/twitch-ext.d.ts](../../apps/extension/src/types/twitch-ext.d.ts)
  - 型別直接綁定 Twitch helper
- [apps/extension/src/services/api.ts](../../apps/extension/src/services/api.ts)
  - 使用 `extension_jwt`、`loginWithTwitchExtension`

### B. 實作層泛用 `extension` 命名

這些名稱未必錯，但要先區分它們是模組名還是產品名：

- backend `ExtensionService`
- backend `/extension/*` routes
- `ExtJWTAuth`
- request body `extension_jwt`

如果只是要改產品描述，不需要在同一張 PR 內一併重命名這些程式項目。

### C. 文件中容易造成誤解的地方

目前 repo 內多數架構與設計文件仍以 Twitch Extension 為現況描述，這反映既有程式 reality。
但 migration 方向現在已定案，因此後續文件應逐步收斂為：

- 既有 runtime 現況：仍有 Twitch-hosted 遺留
- 產品方向：`tachimint` 遷移為 Chrome sidepanel
- 本階段邊界：仍沿用 Twitch identity 與既有 backend contract

---

## 4. 建議拆票方式

### 第一類：文件 truth 校正

目標：
- 明確標示既有實作仍含 Twitch-hosted 遺留
- 明確標示 Chrome sidepanel migration 已是既定方向

適合放進同一張 docs PR 的內容：
- README 說明補強
- 名詞盤點文件更新
- migration decision doc

### 第二類：Chrome Extension migration implementation

這必須是獨立 frontend PR：
- sidepanel runtime 骨架
- 新 app shell / UI 導入
- Twitch auth / heartbeat / claim 邏輯接線
- 舊殼與 `extensions/` cleanup

### 第三類：程式碼命名清理

這是另外一張 refactor / contract 導向 PR：
- `ExtensionService`
- `ExtJWTAuth`
- `extension_jwt`
- Swagger description / schema wording

---

## 5. 已回答與待後續處理的問題

本輪已回答：

1. `apps/extension/` 保留，作為唯一正式 extension 前端 surface
2. Chrome sidepanel 是已定案產品方向
3. 本階段仍沿用 Twitch identity 與既有 backend contract
4. 新的 migration decision source of truth 為 [docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md](2026-04-16-tachimint-chrome-sidepanel-migration.md)

仍待後續 implementation PR 處理：

1. sidepanel runtime 內如何承接既有 Twitch auth/context 流程
2. demo state 如何逐步替換為正式 product state
3. 哪些 Twitch-hosted 遺留檔案可以在最後 cleanup PR 刪除
