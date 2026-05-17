# tachigo API

`services/api` 是 tachigo 的 Go 後端服務，負責 auth、Twitch / Google OAuth、Twitch Extension auth、Watch Points、TACHI claim / spend、Streamer / Agency dashboard API、Airdrop 與 Raffle 流程。

## 快速啟動

需求：

- Go：以 [`go.mod`](go.mod) 為準，目前是 Go 1.25.0
- Docker / Docker Compose：建議用來啟動 PostgreSQL 與完整本機 stack
- `make`：repo root 與本目錄都有 Makefile

從 repo root 啟動完整 stack：

```bash
make dev
```

常用 root 指令：

```bash
make setup  # 建立本機 .env，並設定 git hooks
make dev    # docker compose up --build
make up     # docker compose up --build -d
make down   # 停止 stack
make logs   # 追蹤所有服務 log
```

後端啟動後：

- API health check：<http://localhost:8080/health>
- Swagger UI：<http://localhost:8080/swagger/index.html>
- API base path：`/api/v1`

## 只啟動後端

如果只想跑後端服務，可以在 `services/api` 目錄操作：

```bash
cd services/api
cp .env.example .env
make db-up
make run
```

`make db-up` 會用 Docker 啟動一個 `tachigo-postgres` container，host port 是 `5432`，對應 `.env.example` 內的預設 `DATABASE_URL`。

常用後端指令：

```bash
make run    # atlas migrate apply, then go run ./cmd/server
make build  # go build -o bin/tachigo ./cmd/server
make test   # go test ./... -v
make tidy   # go mod tidy
make db-up  # 啟動單獨 PostgreSQL
make db-down
```

使用 root `docker compose` 時，API container 會連到 compose network 內的 `postgres:5432`；PostgreSQL 對 host 暴露在 `localhost:5433`。

## 環境變數

本機開發先複製範本：

```bash
cp services/api/.env.example services/api/.env
```

`.env.example` 目前包含：

| 變數 | 用途 |
| --- | --- |
| `PORT` | API server port，預設 `8080` |
| `APP_ENV` | 執行環境；`development` 會放寬 production secret validation |
| `DATABASE_URL` | PostgreSQL DSN；root compose 會覆寫成 container network DSN |
| `JWT_ACCESS_SECRET` | access token 簽章 secret，production 必須至少 32 字元 |
| `JWT_REFRESH_SECRET` | refresh token 簽章 secret，必須和 access secret 不同 |
| `JWT_ACCESS_TTL_MINUTES` | access token 有效分鐘數 |
| `JWT_REFRESH_TTL_DAYS` | refresh token 有效天數 |
| `TWITCH_CLIENT_ID` | Twitch OAuth app client id |
| `TWITCH_CLIENT_SECRET` | Twitch OAuth app client secret |
| `TWITCH_REDIRECT_URL` | Twitch OAuth callback URL |
| `TWITCH_EXTENSION_SECRET` | Twitch Extension dashboard 內的 base64 extension secret |
| `TACHIYA_INTERNAL_SHARED_SECRET` | tachiya -> tachigo server-to-server request shared secret |
| `TACHIYA_BASE_URL` | tachiya internal API base URL |
| `GOOGLE_CLIENT_ID` | Google OAuth client id |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |
| `GOOGLE_REDIRECT_URL` | Google OAuth callback URL |
| `TACHI_CONTRACT_ADDRESS` | Sepolia 上的 TachiToken contract address |
| `SEPOLIA_SIGNER_KEY` | 後端送鏈上交易用的 Sepolia testnet signer private key；不可 commit 真實值 |
| `ALLOWED_ORIGINS` | CORS allowlist，使用逗號分隔 |

`config.Load()` 也支援以下變數；需要 email 或前端連結行為時可以加入 `.env`：

| 變數 | 用途 |
| --- | --- |
| `SMTP_HOST` / `SMTP_PORT` | 寄信服務 host / port |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | SMTP credential |
| `SMTP_FROM` | 系統信件寄件者 |
| `FRONTEND_URL` | email link 使用的前端 base URL |

## Swagger

Swagger 入口：

```text
http://localhost:8080/swagger/index.html
```

Swagger source 由 handler annotations 與 `cmd/server/main.go` 的 API metadata 產生，輸出在 [`docs/`](docs/)。

在 Docker dev container 內，Air 會依照 [`.air.toml`](.air.toml) 的 `pre_cmd` 自動執行：

```bash
swag init -g cmd/server/main.go --output docs --quiet
```

如果在本機直接修改 Swagger annotation，也可以在 `services/api` 手動執行同一個指令。

## 資料庫與 migrations

Atlas owns runtime schema changes. Server startup no longer runs schema DDL; schema changes must enter through [`migrations/`](migrations/) and be applied with Atlas.

Current paths:

- Docker entrypoint applies `atlas migrate apply` before starting `air` or `/tachigo`.
- `make run` depends on `make migrate`; use `make run-no-migrate` only when intentionally reusing an already-migrated database.
- `cmd/loader` uses `internal/schema.AtlasSchemaModels()` as the GORM schema source for Atlas.
- Legacy raffle claim token hashing remains a data-only startup repair; do not add schema DDL to server bootstrap.

若需要在乾淨 local PostgreSQL 上重播目前 migration directory，可以使用：

```bash
cd services/api
atlas migrate apply --dir file://migrations --url "$ATLAS_DATABASE_URL"
```

注意：正式環境變更要依當前 deploy workflow / runbook 的 Atlas migration strategy 執行，不要讓 API binary 重新取得 schema DDL ownership。

## 分層結構

主要目錄：

| 路徑 | 說明 |
| --- | --- |
| `cmd/server` | API server entrypoint、config load、DB setup、service wiring |
| `cmd/loader` | Atlas external schema loader |
| `internal/config` | env parsing、production secret validation |
| `internal/database` | GORM / PostgreSQL connection |
| `internal/router` | Gin router、middleware、route grouping、Swagger route |
| `internal/handlers` | HTTP request / response layer |
| `internal/services` | domain logic 與 transaction boundaries |
| `internal/models` | GORM models 與 persisted schema shape |
| `internal/schema` | Atlas loader 使用的 GORM model list |
| `migrations` | Atlas-owned SQL migration directory |
| `docs` | generated Swagger artifacts |

通常修改順序：

1. Model / schema shape：先確認 issue 是否真的允許 schema change。
2. Service：放 domain logic、交易與錯誤處理。
3. Handler / router：只做 HTTP boundary、auth / role middleware 與 DTO mapping。
4. Tests：依風險補 service / handler / workflow regression tests。

## 測試

後端 unit tests 預設多數使用 SQLite in-memory；schema / migration validation 以及 PostgreSQL-specific constraints、partial indexes、Atlas loader 相關變更必須走 PostgreSQL path。

常用指令：

```bash
cd services/api
go test ./...
go test ./internal/services -count=1
go test ./internal/handlers -count=1
```

從 repo root 用 Docker 跑：

```bash
# SQLite-backed unit tests only
docker compose run --no-deps --rm app go test ./...

# Integration / PostgreSQL-related tests
docker compose run --rm app go test -tags integration ./...
```

新增 API endpoint 時，請同步檢查 handler tests、router auth / role boundary、Swagger annotation，以及 `services/api/docs/` 產物是否需要更新。
