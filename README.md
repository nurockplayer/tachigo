# tachigo

Chrome sidepanel extension + Web3 rewards platform.
Viewers spend Bits to earn on-chain tokens; streamers manage rewards from the dashboard.

## Structure

```
tachigo/
├── apps/
│   ├── dashboard/    # Streamer / agency admin dashboard (React + TypeScript + Vite)
│   └── extension/    # Chrome sidepanel frontend (React + TypeScript + Vite)
├── contracts/        # Foundry smart contracts for TACHI token flows
├── deployments/      # Chain deployment metadata
├── design/           # Product and UI design artifacts
├── docs/             # Architecture, policies, decisions, and AI workflow docs
├── infra/            # Repo automation scripts and git hooks
├── plans/            # Planning notes and implementation breakdowns
├── services/
│   └── api/          # Go API (Gin + GORM + PostgreSQL)
└── docker-compose.yml
```

## Documentation

| Area | Link |
|------|------|
| System architecture | [docs/architecture.md](docs/architecture.md) |
| Auth baseline | [docs/auth-architecture.md](docs/auth-architecture.md) |
| Backend permissions | [docs/backend-permissions.md](docs/backend-permissions.md) |
| Chrome sidepanel migration | [docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md](docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md) |
| PR scope policy | [docs/pr-scope-policy.md](docs/pr-scope-policy.md) |
| Dependency update policy | [docs/dependabot-update-policy.md](docs/dependabot-update-policy.md) |
| Dependency inventory policy | [docs/dependency-inventory-policy.md](docs/dependency-inventory-policy.md) |
| Contracts gas snapshot policy | [docs/contracts-gas-snapshot-policy.md](docs/contracts-gas-snapshot-policy.md) |
| Repo automation | [infra/README.md](infra/README.md) |
| AI collaboration | [docs/ai/README.md](docs/ai/README.md) |
| Agent rules | [CLAUDE.md](CLAUDE.md), [AGENTS.md](AGENTS.md) |

## Features

- **Watch Points** — viewers earn off-chain points from watch heartbeat activity.
- **TACHI claim / spend flows** — backend services track spendable balances and bridge selected flows to on-chain mint / burn behavior.
- **Twitch Extension / Chrome sidepanel** — `apps/extension` hosts the migration-stage viewer experience with Twitch identity and extension auth compatibility.
- **Streamer / agency dashboard** — `apps/dashboard` provides the management surface for channels, streamers, and operational settings.
- **Agency and channel configuration** — backend services model agency ownership, streamer relationships, and per-channel reward settings.
- **Airdrop and raffle systems** — backend jobs and APIs support viewer rewards, campaign-style interactions, and result delivery.
- **Security and dependency operations** — CI includes scope gates, backend scanners, dependency inventory reports, and PR review automation.

## Quick start

**Prerequisites:** Docker, Docker Compose

```bash
git clone <repo>
cd tachigo
docker compose up --build
```

| Service  | URL                                      |
| -------- | ---------------------------------------- |
| Backend  | http://localhost:8080                    |
| Swagger  | http://localhost:8080/swagger/index.html |
| Frontend | http://localhost:5173                    |
| Postgres | localhost:5433                           |

If you want local `.env` files, copy the examples first. Docker Compose can still start without them because the env files are optional.
Fill in the secrets in `services/api/.env` before using OAuth or Twitch Extension features.

On Windows PowerShell, you can generate the local env files with:

```powershell
./infra/scripts/setup-env.ps1
```

## Development

```bash
docker compose up --build     # start all services (foreground — see logs)
docker compose up -d --build  # start in background
docker compose down           # stop all services
docker compose logs -f        # tail all logs
```

`make` is still available as a convenience on macOS/Linux, but it is not required.

### Backend (`services/api/`)

- Hot reload via [air](https://github.com/air-verse/air) — save any `.go` file to rebuild
- Swagger docs regenerated automatically on each build (`swag init`)
- Tests use SQLite in-memory — no Postgres required

```bash
docker compose run --no-deps --rm app go test ./...
```

### Frontend (`apps/extension/`)

- Hot reload via Vite HMR
- current migration direction is Chrome sidepanel runtime
- Twitch identity / extension auth related flows are still retained during the migration stage

```bash
docker compose run --no-deps --rm frontend npm run build   # production build
```

## Environment variables

Copy the examples and fill in your secrets:

```bash
cp services/api/.env.example services/api/.env
cp apps/extension/.env.example apps/extension/.env
cp apps/dashboard/.env.example apps/dashboard/.env
```

Windows PowerShell:

```powershell
./infra/scripts/setup-env.ps1
```

Key backend variables:

| Variable                  | Description                            |
| ------------------------- | -------------------------------------- |
| `TWITCH_CLIENT_ID`        | From dev.twitch.tv/console             |
| `TWITCH_CLIENT_SECRET`    | From dev.twitch.tv/console             |
| `TWITCH_EXTENSION_SECRET` | Extension secret (base64)              |
| `GOOGLE_CLIENT_ID`        | From Google Cloud Console              |
| `GOOGLE_CLIENT_SECRET`    | From Google Cloud Console              |
| `JWT_ACCESS_SECRET`       | Random string ≥ 32 chars               |
| `JWT_REFRESH_SECRET`      | Random string ≥ 32 chars               |
| `TACHI_CONTRACT_ADDRESS`  | Sepolia TachiToken contract address    |
| `SEPOLIA_SIGNER_KEY`      | Backend signer key for TACHI mint/burn |

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full system diagram.
For the frontend migration decision record, see [docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md](docs/history/2026-04-16-tachimint-chrome-sidepanel-migration.md).

## CI

GitHub Actions 目前分兩段：

- `PR Scope Police` 先檢查 PR 邊界；超大包或跨 scope PR 會先被擋下
- `CI` 會直接出現在 PR 上，但會先經過輕量 `Scope gate`；只有 scope 合格才會跑 backend / frontend / dashboard 的重型 job

在受保護分支上的 CI：

- **Backend tests** — `go test ./...` inside the dev Docker image
- **Frontend build** — `npm run build` inside the frontend Docker image
