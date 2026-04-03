# tachigo

Twitch extension + Web3 rewards platform.
Viewers spend Bits to earn on-chain tokens; streamers manage rewards from the dashboard.

## Structure

```
tachigo/
├── backend/      # Go API (Gin + GORM + PostgreSQL)
└── tachimint/    # Twitch Extension frontend (React + TypeScript + Vite)
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
- `window.Twitch.ext` is mocked in dev mode — no Twitch Developer Rig needed
- Open http://localhost:5173 to see the extension panel

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

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full system diagram.

## CI

GitHub Actions runs on every push/PR to `main`:

- **Backend tests** — `go test ./...` inside the dev Docker image
- **Frontend build** — `npm run build` inside the frontend Docker image
