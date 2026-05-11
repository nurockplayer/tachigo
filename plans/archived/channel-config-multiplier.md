# channel-config-multiplier 實作計畫

狀態：已完成

refs #68

## 背景

`WatchService.Heartbeat` 以 `seconds_per_point` 控制發點速率（預設 60 秒 1 點），已在 #59 完成。`ChannelConfigService` 基礎實作已在 #73 完成。

本 PR 在現有基礎上新增 `multiplier` 欄位，讓 Streamer / Admin 可針對各頻道調整挖礦倍率，同時新增 `GET /dashboard/channels/:channel_id/config` 端點與完整的 ownership 驗證。

## 架構決策

### 發點公式

```
basePoints    = floor(delta_seconds / seconds_per_point)
pointsToAward = basePoints * multiplier
newRewarded   = basePoints * seconds_per_point   // 記錄 raw seconds，不受 multiplier 影響
```

`newRewarded` 故意不乘 multiplier，原因：若之後 config 被修改，不會造成下次 heartbeat 的 pending seconds 計算錯誤（drift）。

### ownership 驗證（authorizeChannelAccess）

- Admin → 直接通過
- Agency → 501（MVP 暫不實作）
- Streamer → 查 `streamers` 表 `OwnsChannel(userID, channelID)`，不擁有 → 403
- 其他 → 403

`streamers` 表只能透過 `Register`（已驗 auth_provider Twitch 對應）寫入，確保授權鏈完整。

### UpdateChannelConfig upsert 語意

`seconds_per_point=0` 或 `multiplier=0` 表示「保留舊值」，service 層做值保護，避免誤傳 0 清空設定。

## 待實作 checklist

- [x] `backend/migrations/007_channel_config_multiplier.sql` — ALTER TABLE 加 `multiplier` 欄位
- [x] `models.ChannelConfig` 加 `Multiplier int64` 欄位
- [x] `ChannelConfigService.Get` — 取得單筆 config
- [x] `ChannelConfigService.EffectiveMultiplier` — fallback 1
- [x] `ChannelConfigService.UpdateChannelConfig` — 支援同時更新 `seconds_per_point` + `multiplier`
- [x] `WatchService.Heartbeat` — 套用 multiplier 公式，`getSecondsPerPoint` 改為 `getChannelConfig`
- [x] `ChannelConfigHandler.GetChannelConfig` — 新增 GET endpoint
- [x] `ChannelConfigHandler.UpdateChannelConfig` — 支援 multiplier
- [x] `ChannelConfigHandler.authorizeChannelAccess` — Streamer ownership 驗證
- [x] Router — dashboard group 新增 GET `/channels/:channel_id/config`

## 驗證方式

```bash
docker compose run --no-deps --rm app go test ./internal/...
```

測試涵蓋：
- `channel_config_service_test.go` — Get、EffectiveMultiplier（default/override）、UpdateChannelConfig
- `watch_service_test.go` — MultiplierApplied（3x → 3 points）、DefaultMultiplier（1x → 1 point）
- `channel_config_handler_test.go` — Admin/Streamer GET/PUT、ownership 邊界（forbidden/allowed）
