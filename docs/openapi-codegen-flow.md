# OpenAPI TypeScript Codegen Flow

**Source of truth**: #401
**Status**: proposed rollout
**Last reviewed**: 2026-05-13

## 目的

Backend API contract 應該從後端 OpenAPI schema 產生，不應在 Go 與
TypeScript 兩邊長期手動同步 DTO。

本文件定義 frontend-facing TypeScript contract 的預期 rollout 形狀。這份
文件尚未引入 generated artifacts，也不改 runtime API 行為。

## 目前來源

後端 Swagger / OpenAPI artifact 由 `swag init` 從 Go handler annotations
與 API metadata 產生：

```bash
cd services/api
swag init -g cmd/server/main.go --output docs --quiet
```

目前已 commit 的輸出是：

- `services/api/docs/swagger.json`
- `services/api/docs/swagger.yaml`
- `services/api/docs/docs.go`

目前這些 artifact 仍是 Swagger 2.0（`"swagger": "2.0"`），不是 OpenAPI
3.x。這會影響工具選型：目前 `openapi-typescript` 7.x 目標是 OpenAPI
3.0 / 3.1 schema。後續 implementation PR 必須明確加入 Swagger 2.0 到
OpenAPI 3.x 的轉換步驟，或先把後端輸出改成 OpenAPI 3.x，再接上 type
generation。

在 compatibility step 於 CI 被證明可行前，不要直接把最新版本 type generator
指向 `swagger.json`。

## 目標 packages

用分離 package 維持 generated types 與手寫 client logic 的 ownership 邊界：

| Package | 內容 | 規則 |
|---|---|---|
| `packages/shared-types` | 從後端 schema 產生的 TypeScript declarations | Generated files 不手改。 |
| `packages/api-client` | 給 extension 與 dashboard 使用的薄 typed fetch wrapper | 手寫、runtime-light、framework-agnostic。 |

第一個 package 落地時，workspace 應納入 `packages/*`。

## Regeneration flow

最終實作應提供穩定的 root scripts：

```bash
pnpm api:types:generate
pnpm api:types:check
```

預期行為：

- `api:types:generate` 從已 commit 的後端 schema artifact 重新產生 shared
  type package。
- `api:types:check` 重新產生到暫存位置，或以 generator 的 check mode 執行；
  若 committed generated files 發生 drift，指令應失敗。
- Schema generation 仍由 backend workflow 擁有：若 handler annotations 改變，
  同一個 backend PR 必須先 commit `swag init` output，再更新 frontend type
  generation。

## Client 邊界

第一版 typed client 應刻意維持薄層：

- 預設使用 native Fetch API shape。
- 明確處理 auth token injection、base URL 選擇與 JSON parsing。
- 本 issue 不產生 React Query hooks。
- 不強迫 extension 與 dashboard 共用 UI 或 state management code。

若 dashboard 後續有具體需求，可以在 `packages/api-client` 之上另包
dashboard-specific hooks。

## CI guardrail

CI drift check 應在 generated artifacts 已 commit 後才加入：

1. 從乾淨 checkout 執行 schema / type generation command。
2. 若 generated package files 與 committed files 不一致，CI 失敗。
3. Check 只負責 contract drift；除非 PR 觸碰相關 surface，否則不應順手 rebuild
   unrelated frontend bundles。

## Rollout slices

1. 先記錄本 flow 與 Swagger 2.0 compatibility constraint。
2. 新增 `packages/shared-types` 與 pinned generation / conversion command。
3. 新增 `packages/api-client`，只放最小 typed fetch wrapper。
4. 一次導入一個 frontend surface。
5. Generated artifacts 穩定後，再加 CI drift protection。

每個 slice 都應可獨立 review。除非 reviewer 明確同意該 scope，否則不要把
package scaffolding、frontend migration 與 CI policy changes 合併在同一個 PR。
