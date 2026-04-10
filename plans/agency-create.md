狀態：已完成

# Agency 建立 — POST /agencies（admin only）

refs #106

## 背景

#18 規劃了 Agency 管理系統，`POST /agencies` 路由已掛但回傳 501。需要實作 admin 建立 agency 帳號的功能。

## 架構決策

- 不新增獨立的 agency profile 表，直接利用 `users.role = 'agency'`
- 不設初始密碼（`password_hash = nil`），agency 帳號建立後由 admin 通知當事人走 forgot-password 流程自行設定
- Response 只回 `id` + `name`，不回傳任何憑證

## 待實作 checklist

- [x] `AgencyService.Create(name, email string) (*models.User, error)`
  - [x] email 重複時回 `ErrAgencyEmailTaken`
- [x] `AgencyHandler.Create` — `POST /api/v1/agencies`（admin only）
- [x] Router 取代 501 stub
- [x] `main.go` 初始化並注入
- [x] Handler 測試（成功、重複 email、無效 body、非 admin 403）

## 驗證方式

- `go build ./...` 通過
- `docker compose run --no-deps --rm app go test ./...` 全部通過
