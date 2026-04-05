# External Integrations

**Analysis Date:** 2026-04-04

## APIs & External Services

**OAuth 2.0 Providers:**
- Twitch OAuth - User authentication via Twitch account
  - SDK/Client: `golang.org/x/oauth2` (Go), HTTP calls (frontend)
  - Auth endpoints: `https://id.twitch.tv/oauth2/authorize`, `https://id.twitch.tv/oauth2/token`
  - User info: `https://api.twitch.tv/helix/users`
  - Implementation: `backend/internal/services/auth_service.go` (OAuth2 config setup)
  - Frontend: tachimint uses Twitch Extension JWT instead of standard OAuth
  - Env vars: `TWITCH_CLIENT_ID`, `TWITCH_CLIENT_SECRET`, `TWITCH_REDIRECT_URL`

- Google OAuth - User authentication via Google account
  - SDK/Client: `golang.org/x/oauth2/google`
  - Auth endpoints: Google Cloud OAuth2
  - User info: `https://www.googleapis.com/oauth2/v3/userinfo`
  - Implementation: `backend/internal/services/auth_service.go`
  - Env vars: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`

**Twitch Extension Integration:**
- Extension JWT verification - Validates signed JWTs from Twitch Extension SDK
  - Implementation: `backend/internal/services/extension_service.go` (VerifyExtJWT)
  - Secret: Base64-encoded `TWITCH_EXTENSION_SECRET` from Twitch Extension dashboard
  - Used for: Viewer authentication within extension panel
  - Claims extracted: `opaque_user_id`, `user_id`, `channel_id`, `role`

- Bits Transaction Receipts - Verification of Bits purchases in extension
  - Implementation: `backend/internal/services/extension_service.go` (VerifyReceiptJWT)
  - Payload: Transaction ID, SKU, amount, type ("bits")
  - Handled in: `backend/internal/handlers/extension_handler.go`

## Data Storage

**Databases:**
- PostgreSQL 16-alpine
  - Connection: Environment variable `DATABASE_URL` (DSN format: `host=... user=... password=... dbname=... port=... sslmode=...`)
  - Default dev: `host=postgres user=postgres password=postgres dbname=tachigo port=5432 sslmode=disable`
  - ORM: GORM v1.30.0 with `gorm.io/driver/postgres`
  - Connection pooling: 25 max open, 10 max idle connections
  - Tables: users, auth_providers, refresh_tokens, web3_nonces, email_verifications, password_resets, channel_configs, points_ledgers, points_transactions, watch_sessions, watch_stats, broadcast_stats
  - Enums: `user_role` (viewer, streamer, admin)

- SQLite (testing only)
  - Driver: `gorm.io/driver/sqlite`
  - Used: In-memory database for unit tests
  - Does NOT require PostgreSQL for test execution

**File Storage:**
- Local filesystem only - No cloud storage integration detected
- Docker volumes: `postgres_data` for persistent database storage

**Caching:**
- None detected - No Redis, Memcached, or similar caching layer

## Authentication & Identity

**Auth Provider:**
- Custom + OAuth2
  - Email/Password: Custom bcrypt-hashed password storage in users table
  - Implementation: `backend/internal/services/auth_service.go`
  - JWT tokens: Access token (15 min TTL), Refresh token (30 day TTL)
  - Secrets: `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` (32+ chars recommended)

- OAuth2 Flows:
  - Twitch & Google OAuth redirect flow with callback URL
  - Twitch Extension JWT direct verification (no redirect needed)

**Middleware:**
- Auth middleware: `backend/internal/middleware/auth.go`
  - Validates Bearer tokens from Authorization header
  - Extracts user ID and role from JWT claims
- Extension auth: `backend/internal/middleware/ext_auth.go`
  - Validates Twitch Extension JWT
- CORS middleware: `backend/internal/middleware/cors.go`
  - Configurable via `ALLOWED_ORIGINS` env var
  - Default (dev): `http://localhost:3000,http://localhost:5173,http://localhost:5174`

**Wallet/Web3:**
- Ethereum integration: `github.com/ethereum/go-ethereum v1.17.2`
- Nonce generation for wallet signature verification: `backend/internal/models/web3_nonce.go` (model exists)
- Address storage: `backend/internal/models/address.go` (model exists)
- Actual Web3 signature verification: Not yet implemented in codebase (models present, no handler found)

## Monitoring & Observability

**Error Tracking:**
- None detected - No Sentry, Rollbar, or similar integration

**Logs:**
- Standard logging: Go `log` package and Gin framework logs
- Database logging: GORM Logger in Info mode (logs SQL queries)
- No structured logging or log aggregation detected

**Health Checks:**
- PostgreSQL: Docker healthcheck via `pg_isready`
- HTTP: Swagger docs at `GET /swagger/index.html` serve as implicit health indicator

## CI/CD & Deployment

**Hosting:**
- Docker containers (development via docker-compose)
- Production deployment: Not specified in codebase (app is containerized, ready for any Docker-capable platform)

**CI Pipeline:**
- GitHub Actions (`.github/workflows/` directory exists)
- Backend tests: `go test ./...` in dev Docker image
- Frontend build: `npm run build` in frontend Docker image
- Triggers: Every push/PR to `main` branch

## Environment Configuration

**Required env vars:**

*Backend (critical for OAuth/Twitch/Email):*
- `TWITCH_CLIENT_ID` - Twitch OAuth app registration
- `TWITCH_CLIENT_SECRET` - Twitch OAuth app secret
- `TWITCH_EXTENSION_SECRET` - Base64-encoded Twitch Extension secret
- `GOOGLE_CLIENT_ID` - Google OAuth app ID
- `GOOGLE_CLIENT_SECRET` - Google OAuth app secret
- `JWT_ACCESS_SECRET` - Random 32+ char string for access token signing
- `JWT_REFRESH_SECRET` - Random 32+ char string for refresh token signing
- `SMTP_HOST` / `SMTP_USERNAME` / `SMTP_PASSWORD` - Email server (optional, defaults to no-op)
- `DATABASE_URL` - PostgreSQL connection string

*Frontend:*
- `VITE_TACHIGO_API_URL` (tachimint) - Backend API URL (default: http://localhost:8080)
- `VITE_API_URL` (dashboard) - Backend API URL (default: http://localhost:8080)

**Secrets location:**
- Backend: `.env` file in `backend/` directory (git-ignored)
- Frontend: `.env` files in respective directories (git-ignored)
- Example templates: `.env.example` files in each directory
- Docker Compose: Loads env files from `backend/.env`, `tachimint/.env`, `dashboard/.env` (optional, with fallbacks)

## Webhooks & Callbacks

**Incoming:**
- Twitch OAuth callback: `GET /api/v1/auth/twitch/callback` - Receives authorization code
- Google OAuth callback: `GET /api/v1/auth/google/callback` - Receives authorization code
- No other webhook endpoints detected

**Outgoing:**
- None detected - System does not call external webhooks
- Email sending: One-way SMTP to configured mail server (optional)

## Data Flow Integration Points

**Authentication Flow:**
1. User initiates OAuth or email/password login
2. Backend creates JWT access + refresh tokens
3. Frontend stores access token in memory, refresh token in localStorage
4. All subsequent API requests include `Authorization: Bearer {access_token}` header

**Twitch Extension Flow:**
1. Twitch Extension SDK provides extension JWT to viewer
2. Frontend sends JWT to backend heartbeat/auth endpoints
3. Backend verifies JWT signature using `TWITCH_EXTENSION_SECRET`
4. Backend responds with user points balance
5. Frontend displays in extension panel

**Bits Transaction Flow:**
1. Viewer completes Bits purchase in extension
2. Twitch signs transaction receipt JWT
3. Frontend submits extension JWT + receipt to `/api/v1/extension/bits/complete`
4. Backend verifies both JWTs using extension secret
5. Points ledger updated

**Email Verification:**
- Backend sends HTML email via SMTP (if configured)
- Email contains verification link with token
- Frontend user clicks link, verification endpoint updates database
- Implementation: `backend/internal/services/email_auth_service.go`

---

*Integration audit: 2026-04-04*
