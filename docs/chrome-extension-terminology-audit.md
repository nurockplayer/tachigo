# Chrome Extension 名詞盤點

> 用途：盤點 repo 內 `Chrome Extension`、`Twitch Extension` 與泛用 `extension` 命名的混用情況。
> 狀態：盤點文件，不是 migration spec，也不是「已完成 Chrome Extension 轉換」的宣告。
> 最後更新：2026-04-10

---

## 1. 目前可確認的 source of truth

依目前 repo 既有文件與程式碼現況，可先確認以下事實：

1. `tachimint` 的可運作實作目前仍是 **Twitch-hosted extension runtime**
2. 前端與後端仍依賴 `window.Twitch.ext`、`extension_jwt`、Twitch helper script 等流程
3. 若要正式改成 Chrome Extension，必須先有獨立的架構決策或 migration spec

因此，這份文件的用途是「盤點與拆題」，不是把 Chrome Extension 寫成既定現況。

---

## 2. 名詞層次拆分

目前 repo 內實際混在一起的是三種不同層次：

| 類別 | 代表什麼 | 目前狀態 |
|---|---|---|
| 產品形式 | 使用者最終安裝與使用的前端形態 | 尚未有已定案、可實作的 Chrome Extension migration spec |
| 現行 runtime | 現在程式真正依賴的執行環境 | Twitch-hosted extension |
| 模組 / API 命名 | `/extension/*`、`ExtensionService`、`extension_jwt` 等名稱 | 已存在於前後端契約與程式結構中 |

---

## 3. 盤點結果

### A. 已明確依賴 Twitch runtime 的地方

這些項目不是單純改字詞就能處理：

- [tachimint/index.html](../tachimint/index.html)
  - 仍載入 Twitch Extension Helper
- [tachimint/src/mock/twitch-ext.ts](../tachimint/src/mock/twitch-ext.ts)
  - 本地開發使用 `window.Twitch.ext` mock
- [tachimint/src/types/twitch-ext.d.ts](../tachimint/src/types/twitch-ext.d.ts)
  - 型別直接綁定 Twitch helper
- [tachimint/src/services/api.ts](../tachimint/src/services/api.ts)
  - 使用 `extension_jwt`、`loginWithTwitchExtension`

### B. 實作層泛用 `extension` 命名

這些名稱未必錯，但要先區分它們是模組名還是產品名：

- backend `ExtensionService`
- backend `/extension/*` routes
- `ExtJWTAuth`
- request body `extension_jwt`

如果只是要改產品描述，不需要在同一張 PR 內一併重命名這些程式項目。

### C. 文件中容易造成誤解的地方

目前 repo 內多數架構與設計文件仍以 Twitch Extension 為現況描述，這與程式 reality 一致。
若未來要引入 Chrome Extension 的產品方向，應明確標示為「未來規劃」或「待定 migration」，不能直接覆寫現況描述。

---

## 4. 建議拆票方式

### 第一類：文件 truth 校正

目標：
- 明確標示目前是 Twitch-hosted implementation
- 避免把尚未定案的 Chrome Extension migration 寫成既定事實

適合放進同一張 docs PR 的內容：
- README 說明補強
- 名詞盤點文件
- 未來 migration 需要回答的問題列表

### 第二類：Chrome Extension migration spec

這必須是獨立 issue / spec：
- 身份來源是否仍依賴 Twitch
- `extension_jwt` 的替代方案是什麼
- Bits / viewer context / broadcaster context 如何取得
- `window.Twitch.ext` mock 與 hosted 測試流程如何替換

### 第三類：程式碼命名清理

這是另外一張 refactor / contract 導向 PR：
- `ExtensionService`
- `ExtJWTAuth`
- `extension_jwt`
- Swagger description / schema wording

---

## 5. 後續需要明確回答的問題

在開始任何 Chrome Extension migration 前，至少要先定義：

1. `tachimint` 現在是否仍以 Twitch-hosted extension 為唯一可運作實作？
2. Chrome Extension 是已定案產品方向，還是僅為探索中的可能形態？
3. 若要遷移，誰提供 viewer identity、channel context、Bits 相關能力？
4. 哪些文件是新的 source of truth？

在這些問題被正式回答前，建議 repo 文件保持「現況真實」優先。
