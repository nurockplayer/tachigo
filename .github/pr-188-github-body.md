## 什麼改動

- Chrome extension demo sidepanel：Coupon 兌換商城（`CouponShopPanel`）、HUD 商城入口、以 demo TCG 餘額兌換流程。
- 多語系字串與 coupon 流程測試；後續依 review 調整：同步路徑原子扣款、已兌換 id 提升至 `App` 並寫入 demo state、`coupon` 商品文案與返回鍵 i18n、補成功／重複兌換測試。

## 為什麼

- 實作 extension coupon shop 示範 UI 與互動；對齊規格來源 issue。

## Release Context

- Release type：n/a

## Scope 對齊

- Source of truth: #15
- Depends on PR: none
- Backend contract already in develop:
  - [x] yes
  - [ ] no
- If no, this PR is:
  - [ ] stacked on dependency branch
  - [ ] intentionally blocked until dependency merges
- 本 PR 是否完全在 source of truth 範圍內？
  - [x] 是
  - [ ] 否，已另開 issue / PR 處理超出部分
- 本 PR 明確不做：
  - 不串接正式 backend API、不變更 production Twitch extension 行為。
  - 不擴張到 `tachimint/`、`dashboard/`、`backend/` 等其他 product surface。
  - 不把 `tachigo-ui/` 或其他未於本 PR 列出的目錄納入範圍。

## 超出範圍內容

- 無

## 測試方式

- [x] 本地測試過
- [x] 有寫 / 更新測試
  - `cd extensions/tachigo-demo-sidepanel && pnpm test:run && pnpm build`

## 備註

- refs #15
