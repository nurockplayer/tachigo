# tachimint

`tachimint` 是 `tachigo` repo 內唯一正式維護的前端 surface。

目前已定案的方向是：

- 前端 runtime 逐步遷移為 Chrome sidepanel extension
- 本階段仍沿用 Twitch identity / extension auth 相關流程
- backend contract 仍沿用既有 API

完整決策請見 [../docs/tachimint-chrome-sidepanel-migration.md](../docs/tachimint-chrome-sidepanel-migration.md)。

## 目前定位

這個目錄是後續 migration 的收斂目標，不會與 `extensions/tachigo-demo-sidepanel/` 長期並存成兩個正式產品入口。

`extensions/tachigo-demo-sidepanel/` 在這個階段的角色是 migration source：

- 提供 sidepanel runtime 方向
- 提供新 app shell / UI 參考
- 不作為長期保留的正式前端 surface

## 本輪 migration 邊界

這個 migration decision 目前只固定以下原則：

- 保留 `tachimint/` 目錄名稱
- runtime 遷移到 Chrome sidepanel
- 暫時保留 Twitch identity 與既有 extension auth 相關流程
- 暫時沿用既有 backend contract

這個階段明確不做：

- 不重做 backend API contract
- 不另建新的 viewer identity 系統
- 不同步擴張到 `backend/` 或 `dashboard/`

## 後續實作方向

後續 frontend PR 預期按這個順序收斂：

1. 建立 sidepanel runtime 骨架
2. 導入新的 app shell 與 UI
3. 把 Twitch auth、heartbeat、claim 等既有邏輯接回
4. 最後清掉過時舊殼與 `extensions/` 遺留
