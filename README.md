# tachigo

Chrome sidepanel extension + Web3 rewards platform.
Viewers spend Bits to earn on-chain tokens; streamers manage rewards from the dashboard.

## Structure

```
tachigo/
├── backend/      # Go API (Gin + GORM + PostgreSQL)
├── tachimint/    # Chrome sidepanel frontend (React + TypeScript + Vite)
└── dashboard/    # Admin dashboard (React + TypeScript + Vite)
```

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
Fill in the secrets in `backend/.env` before using OAuth or Twitch Extension features.

On Windows PowerShell, you can generate the local env files with:

```powershell
./scripts/setup-env.ps1
```

## Development

```bash
docker compose up --build     # start all services (foreground — see logs)
docker compose up -d --build  # start in background
docker compose down           # stop all services
docker compose logs -f        # tail all logs
```

`make` is still available as a convenience on macOS/Linux, but it is not required.

### Backend (`backend/`)

- Hot reload via [air](https://github.com/air-verse/air) — save any `.go` file to rebuild
- Swagger docs regenerated automatically on each build (`swag init`)
- Tests use SQLite in-memory — no Postgres required

```bash
docker compose run --no-deps --rm app go test ./...
```

### Frontend (`tachimint/`)

- Hot reload via Vite HMR
- current migration direction is Chrome sidepanel runtime
- Twitch identity / extension auth related flows are still retained during the migration stage

```bash
docker compose run --no-deps --rm frontend npm run build   # production build
```

## Environment variables

Copy the examples and fill in your secrets:

```bash
cp backend/.env.example backend/.env
cp tachimint/.env.example tachimint/.env
cp dashboard/.env.example dashboard/.env
```

Windows PowerShell:

```powershell
./scripts/setup-env.ps1
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
For the current frontend migration decision, see [docs/tachimint-chrome-sidepanel-migration.md](docs/tachimint-chrome-sidepanel-migration.md).

## Documentation

- [docs/architecture.md](docs/architecture.md) — system architecture
- [docs/claude-codex-cheatsheet.md](docs/claude-codex-cheatsheet.md) — quick reference for Claude Code + Codex collaboration
- [docs/claude-codex-workflow.md](docs/claude-codex-workflow.md) — full workflow guide for low-token Claude Code usage
- [docs/pr-scope-policy.md](docs/pr-scope-policy.md) — PR 邊界規則、required checks、scope police 設定
- [CLAUDE.md](CLAUDE.md) — repo-specific collaboration rules and command entry points

## CI

GitHub Actions 目前分兩段：

- `PR Scope Police` 先檢查 PR 邊界；超大包或跨 scope PR 會先被擋下
- `CI` 會直接出現在 PR 上，但會先經過輕量 `Scope gate`；只有 scope 合格才會跑 backend / frontend / dashboard 的重型 job

在受保護分支上的 CI：

- **Backend tests** — `go test ./...` inside the dev Docker image
- **Frontend build** — `npm run build` inside the frontend Docker image
