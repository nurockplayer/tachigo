# Codebase Structure

## Directory Layout

```
tachigo/                          # Monorepo root
├── backend/                      # Go API server
│   ├── cmd/server/               # main entry point
│   ├── internal/
│   │   ├── config/               # App config (env vars, JWT secrets)
│   │   ├── database/             # GORM DB init, migrations runner
│   │   ├── handlers/             # HTTP handlers (Gin)
│   │   ├── middleware/           # Auth JWT middleware, CORS
│   │   ├── models/               # GORM model structs
│   │   ├── router/               # Route registration
│   │   └── services/             # Business logic
│   ├── migrations/               # Raw SQL migration files
│   ├── docs/                     # Swagger/OpenAPI generated docs
│   ├── Dockerfile
│   ├── Makefile
│   ├── go.mod
│   └── .air.toml                 # Hot reload config
│
├── tachimint/                    # Twitch Extension frontend (React + TypeScript)
│   ├── src/
│   │   ├── components/           # UI components
│   │   ├── hooks/                # Custom React hooks
│   │   ├── services/             # API client calls
│   │   ├── mock/                 # Mock data for dev/testing
│   │   ├── types/                # Shared TypeScript types
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── dist/                     # Build output (committed for Twitch CDN)
│   ├── package.json
│   └── vite.config.ts
│
├── dashboard/                    # Admin dashboard frontend (React + TypeScript)
│   ├── src/
│   │   ├── components/           # UI components (shadcn/ui based)
│   │   ├── pages/                # Route-level page components
│   │   ├── services/             # API client, auth service
│   │   ├── lib/                  # Utility functions
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
│
├── docs/                         # Architecture & design docs
│   ├── architecture.md
│   ├── feature-discussion.md
│   ├── watch-to-points-design.md
│   ├── uuid-v7.md
│   └── sequence-diagram.md
│
├── plans/                        # Implementation plan docs per feature
│   ├── dashboard-auth.md
│   ├── dashboard-skeleton.md
│   ├── refine-dashboard-mvp.md
│   ├── uuid-v7-migration.md
│   └── watch-points-channel-config.md
│
├── scripts/                      # Setup scripts
├── Makefile                      # Top-level dev commands (make dev, make down)
├── docker-compose.yml            # Production-like service orchestration
├── docker-compose.override.yml   # Dev overrides (hot reload, exposed ports)
└── CLAUDE.md                     # Claude Code guidelines
```

## Key Locations

### Backend

| Path | Purpose |
|------|---------|
| `backend/cmd/server/` | `main.go` — wire app, run HTTP server |
| `backend/internal/config/config.go` | Load env vars, JWT secrets, DB DSN |
| `backend/internal/database/` | `db.go` — GORM init; `migrations.go` — auto-migrate |
| `backend/internal/router/router.go` | Route grouping and middleware attachment |
| `backend/internal/middleware/auth.go` | JWT validation, claims extraction |
| `backend/internal/models/` | One file per entity (user, auth_provider, etc.) |
| `backend/internal/handlers/` | One file per handler group + `_test.go` sibling |
| `backend/internal/services/` | One file per service + `_test.go` sibling |
| `backend/migrations/` | Numbered SQL files (001–004) |

### Frontend (shared pattern — tachimint & dashboard)

| Path | Purpose |
|------|---------|
| `src/main.tsx` | React app entry point |
| `src/App.tsx` | Router configuration |
| `src/services/` | API client functions, auth state management |
| `src/components/` | Reusable UI components |
| `src/pages/` (dashboard only) | Page-level route components |
| `src/hooks/` (tachimint only) | Custom React hooks |

## Naming Conventions

### Go (Backend)

- **Files**: `snake_case.go` — e.g. `points_service.go`, `auth_handler.go`
- **Test files**: sibling pattern — `points_service_test.go` next to `points_service.go`
- **Packages**: flat, domain-named — `handlers`, `services`, `models`, `middleware`
- **Types**: `PascalCase` — `PointsService`, `UserRole`, `TxSource`
- **Methods**: `PascalCase` for exported, `camelCase` for internal

### TypeScript (Frontend)

- **Components**: `PascalCase.tsx` — e.g. `LoginPage.tsx`, `PointsBalance.tsx`
- **Services**: `camelCase.ts` — e.g. `auth.ts`, `api.ts`
- **Hooks**: `use` prefix — e.g. `usePoints.ts`
- **Types**: `PascalCase` interfaces — e.g. `AuthUser`, `ChannelConfig`

## Migration Numbering

SQL migrations use 3-digit prefix: `001_init.sql`, `002_email_auth.sql`, etc.

Note: `004_channel_config.sql` and `004_rbac_roles.sql` share the same prefix — likely applied in the same release. Future migrations should use `005_` onwards.

## Build Artifacts

- `tachimint/dist/` — committed to repo; deployed to Twitch CDN
- `backend/tmp/` — air hot-reload build cache; not committed
- `dashboard/` build output — not committed; served via Docker
