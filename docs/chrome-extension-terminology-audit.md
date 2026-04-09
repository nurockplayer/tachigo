# Chrome Extension 名詞清理盤點

> 用途：釐清本專案中 `Chrome Extension` 與 `Twitch Extension` 的混用情況，避免產品定位、技術實作與歷史命名混為一談。
> 狀態：盤點文件，供後續拆 issue / PR 使用。
> 最後更新：2026-04-09

---

## 1. 結論摘要

本專案目前需要明確區分三件事：

1. **產品定位**
   `tachimint` 的產品方向應統一表述為 **Chrome Extension**

2. **現有實作命名**
   程式碼中仍保留許多 `extension`、`Twitch Extension`、`extension_jwt`、`ExtensionService` 等名稱，這些多半反映的是既有 API 與驗證流程，而不是產品形式

3. **外部平台依賴**
   某些程式碼與 mock 仍直接依賴 `window.Twitch.ext` 或 Twitch 提供的 JWT / helper，這已超出單純文件命名問題，屬於實作與架構層的議題

簡單說：

- 文件中的產品描述應優先統一成 `Chrome Extension`
- API / service / payload 命名是否要跟著改，應另開 issue 評估
- 若未來真的不再依賴 Twitch Extension runtime，則需要獨立整理 migration 計畫

---

## 2. 已完成的文件層修正

以下文件已先統一產品定位為 `Chrome Extension`：

- [docs/architecture.md](architecture.md)
- [docs/feature-discussion.md](feature-discussion.md)
- [docs/tokenomics.md](tokenomics.md)
- [docs/watch-to-points-design.md](watch-to-points-design.md)
- [tachimint/README.md](../tachimint/README.md)

以下文件已在獨立 PR 處理：

- [docs/sequence-diagram.md](sequence-diagram.md) — 另拆 PR 處理（流程圖描述現況需單獨評估）
- [docs/extension-ui-prompts.md](extension-ui-prompts.md) — 已拆至 #154（新增 UI 設計提示詞）

---

## 3. 目前仍殘留的命名類型

### A. 文件 / 說明文字仍帶有 Twitch Extension 表述

這類通常可以視為下一輪文件清理範圍：

- [backend/.env.example](../backend/.env.example)
  `# Twitch Extension`
- [backend/cmd/server/main.go](../backend/cmd/server/main.go)
  swagger description 仍寫 `Twitch extension + Web3 rewards platform`
- [backend/docs/swagger.yaml](../backend/docs/swagger.yaml)
  description / summary 仍有 `Twitch Extension JWT`
- [backend/docs/swagger.json](../backend/docs/swagger.json)
  description / summary 仍有 `Twitch Extension JWT`
- [backend/docs/docs.go](../backend/docs/docs.go)
  產生出的 swagger 內容仍有 `Twitch Extension JWT`

### B. 實作層泛用 `extension` 命名

這類不一定要立刻改，因為它們可能只是模組名，而不是產品定位錯誤：

- [backend/internal/services/extension_service.go](../backend/internal/services/extension_service.go)
  `ExtensionService`
- [backend/internal/handlers/extension_handler.go](../backend/internal/handlers/extension_handler.go)
  `ExtensionHandler`
- [backend/internal/router/router.go](../backend/internal/router/router.go)
  `/extension/*` 路由群組
- [backend/internal/middleware/ext_auth.go](../backend/internal/middleware/ext_auth.go)
  `ExtJWTAuth`

這一層的問題不是 `extension` 這個字，而是裡面是否還把「產品 = Twitch Extension」寫死。

### C. Twitch-specific payload / runtime 命名

這類已不是單純改字詞，而是與實際串接方式有關：

- [tachimint/src/services/api.ts](../tachimint/src/services/api.ts)
  `extension_jwt`、`loginWithTwitchExtension`
- [tachimint/src/mock/twitch-ext.ts](../tachimint/src/mock/twitch-ext.ts)
  `window.Twitch.ext` mock
- [tachimint/src/types/twitch-ext.d.ts](../tachimint/src/types/twitch-ext.d.ts)
  Twitch helper type declarations
- [tachimint/index.html](../tachimint/index.html)
  `Twitch Extension Helper`
- [backend/internal/services/extension_service.go](../backend/internal/services/extension_service.go)
  `Twitch Extension JWT`
- [backend/internal/handlers/extension_handler.go](../backend/internal/handlers/extension_handler.go)
  request body `extension_jwt`

這些名稱反映的不只是文案，而是目前系統仍假設某種 Twitch helper / JWT 存在。

---

## 4. 建議的拆分方式

### 第一層：文件與對外說明清理

目標：

- 所有產品描述統一為 `Chrome Extension`
- README / architecture / tokenomics / swagger description 不再誤導

適合項目：

- `docs/*`
- `README`
- swagger summary / description
- `.env.example` 註解

風險：

- 低
- 主要是文字修正

### 第二層：程式碼命名清理

目標：

- 評估是否將 `ExtensionService`、`extension_jwt`、`loginWithTwitchExtension` 等名稱改為較中性的命名

適合項目：

- handler / service / middleware 名稱
- request / response payload 欄位
- swagger schema 名稱

風險：

- 中
- 可能牽涉前後端契約、測試、文件同步

### 第三層：架構與平台依賴清理

目標：

- 釐清 `tachimint` 是否仍依賴 Twitch 提供的 helper、JWT 與嵌入環境
- 若產品已改為 Chrome Extension，定義新的身份來源與授權流程

適合項目：

- `window.Twitch.ext` 替代方案
- `extension_jwt` 來源替代方案
- auth/login 流程重設計
- bits / Twitch-specific capabilities 的新邊界

風險：

- 高
- 這是實作與產品邏輯變更，不應和單純文案 PR 混在一起

---

## 5. 建議後續 issue 題目

可拆成以下幾個獨立 issue：

1. `docs: replace remaining Twitch Extension product wording with Chrome Extension`
   範圍：文件、README、swagger description、`.env.example` 註解

2. `refactor: audit extension naming in backend/frontend interfaces`
   範圍：`ExtensionService`、`ExtJWTAuth`、`extension_jwt`、`loginWithTwitchExtension`

3. `research: define auth/runtime model for Chrome Extension version of tachimint`
   範圍：Twitch helper、JWT 來源、Chrome Extension 身份模型、相容策略

4. `docs: define terminology policy for Chrome Extension vs Twitch integration`
   範圍：整理一份術語表，說明哪些詞代表產品、哪些詞代表串接來源、哪些詞是 legacy 命名

---

## 6. 術語建議

建議未來統一使用：

| 類別 | 建議用詞 | 說明 |
|---|---|---|
| 產品形式 | `Chrome Extension` | 指使用者實際安裝與使用的前端產品 |
| Twitch 串接 | `Twitch integration` | 指 Twitch API、身份、Bits、直播資料等外部依賴 |
| 前端模組 | `extension` | 可保留作為中性模組名稱 |
| 舊驗證名稱 | `legacy extension_jwt` | 若短期內不能改欄位名，可在文件中這樣註記 |
| 舊 helper | `legacy Twitch helper` | 指 `window.Twitch.ext` 相關相容層 |

避免再直接把以下兩者混成同一件事：

- `Chrome Extension`
- `Twitch Extension`

前者是產品形式，後者若仍存在，只能代表歷史命名或特定串接機制。
