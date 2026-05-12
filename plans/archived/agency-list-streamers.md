狀態：已完成

# Agency 查詢旗下 streamer — GET /agencies/:id/streamers

refs #107

## 背景

#18 規劃了 Agency 管理系統，`GET /agencies/:id/streamers` 路由已掛但回傳 501。`agency_streamers` 關係表已存在，需要實作查詢邏輯。

## 架構決策

- `AgencyService.ListStreamers` 直接查 `agency_streamers` 表，回傳 `[]models.AgencyStreamer`
- RBAC：agency 只能查自己（`id` == 自己的 user ID），admin 可查任何人
- Response 只回 `channel_id` + `user_id`，不回傳其他敏感欄位

## 待實作 checklist

- [x] `AgencyService.ListStreamers(agencyID uuid.UUID) ([]models.AgencyStreamer, error)`
- [x] `AgencyHandler.ListStreamers` — `GET /api/v1/agencies/:id/streamers`（agency or admin）
- [x] Router 取代 501 stub
- [x] Handler 測試（admin 成功、agency 查自己成功、agency 查他人 403、無效 UUID 400、未登入 401）

## 驗證方式

- `go build ./...` 通過
- `docker compose run --no-deps --rm app go test ./...` 全部通過
