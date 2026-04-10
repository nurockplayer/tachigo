# Architecture

**Analysis Date:** 2026-04-04

## Pattern Overview

**Overall:** Three-tier distributed architecture with specialized frontends and a centralized Go backend serving multiple client types (Twitch Extension, Admin Dashboard, and internal services).

**Key Characteristics:**
- Decoupled frontend clients consuming REST API
- Service-oriented backend design with dependency injection
- Per-channel points ledger model supporting dual-balance accounting (cumulative total + spendable balance)
- Stateless JWT authentication with refresh token rotation
- Database-driven configuration for watch-time reward calculations

## Layers

**Client Layer:**
- Purpose: Specialized user interfaces for different stakeholder roles
- Location: `tachimint/` (Twitch Extension), `dashboard/` (Admin Management), `tachiya` (E-commerce, external)
- Contains: React components, hooks, service integrations, authentication state
- Depends on: Go backend REST API `/api/v1`, OAuth providers (Twitch, Google, Web3/SIWE)
- Used by: End users (viewers, streamers, agencies)

**API Layer (Go):**
- Purpose: RESTful HTTP API gateway accepting requests from multiple clients
- Location: `backend/cmd/server/main.go`, `backend/internal/router/`
- Contains: Route definitions, handler wiring, middleware stack (CORS, JWT auth, logging)
- Depends on: Handler layer, service layer, database connection
- Used by: All frontend clients

**Handler Layer:**
- Purpose: HTTP request/response translation; delegates business logic to services
- Location: `backend/internal/handlers/`
- Contains: Request unmarshaling, response formatting, basic input validation
- Depends on: Service layer, models
- Used by: Router (wired via dependency injection)
- Key handlers:
  - `auth_handler.go` — OAuth flows (Twitch, Google, Web3), JWT token refresh
  - `watch_handler.go` — Extension watch session lifecycle, heartbeat
  - `points_handler.go` — Balance retrieval, transaction history
  - `extension_handler.go` — Bits completion callback, extension JWT verification
  - `user_handler.go` — User profile, address management
  - `rbac_handler_test.go` — Role-based access control (admin, streamer, viewer)

**Service Layer:**
- Purpose: Business logic and state management; repository pattern for data access
- Location: `backend/internal/services/`
- Contains: Use case implementations, transactional operations, integration logic
- Depends on: Database, models, external services (OAuth providers, SMTP)
- Key services:
  - `auth_service.go` — User registration, login, token generation (JWT/refresh), OAuth delegation
  - `points_service.go` — Points ledger queries, deduction with ACID guarantees, balance calculations
  - `watch_service.go` — Watch session management (start/end), watch-time stats tracking
  - `extension_service.go` — Bits completion, extension JWT verification
  - `user_service.go` — Profile management, address operations
  - `email_auth_service.go` — Email verification, password reset

**Model Layer:**
- Purpose: Data structure definitions and GORM ORM mapping
- Location: `backend/internal/models/`
- Contains: User, PointsLedger, PointsTransaction, WatchSession, AuthProvider entities
- Depends on: GORM, PostgreSQL enum types
- Key models:
  - `user.go` — UserRole enum (viewer/streamer/agency/admin), soft-delete support
  - `points.go` — PointsLedger (per user+channel), PointsTransaction with TxSource enum
  - `watch_session.go` — WatchSession (is_active flag for partial unique index), WatchTimeStat, BroadcastTimeStat
  - `email_auth.go` — EmailVerification, PasswordReset lifecycle entities

**Data Layer:**
- Purpose: PostgreSQL persistence, connection pooling, migrations
- Location: `backend/internal/database/`, `backend/migrations/`
- Contains: GORM database connection, raw SQL migrations
- Depends on: PostgreSQL driver, GORM
- Features:
  - Max open connections: 25, idle: 10 (see `backend/internal/database/db.go`)
  - Partial unique index on watch_sessions (user_id, channel_id, is_active=true) for single-session per viewer
  - Partial unique index on points_ledgers (user_id, channel_id)
  - Manual enum type creation before AutoMigrate (`backend/cmd/server/main.go` lines 37-44)

**Configuration Layer:**
- Purpose: Environment-driven configuration loading
- Location: `backend/internal/config/config.go`
- Contains: Server, Database, JWT, OAuth, SMTP, App settings
- Depends on: OS environment, godotenv
- Pattern: Single Config struct with typed sub-configs, getEnv() with fallback defaults

## Data Flow

**Extension Watch-Time Points Flow:**

1. Viewer connects to Twitch stream with tachimint extension loaded
2. Extension calls `POST /api/v1/extension/auth/login` with OAuth token from Twitch (exchange for tachigo JWT)
3. Extension polls `POST /api/v1/extension/watch/start` to begin watch session
4. Every heartbeat period (e.g., 30s), extension calls `POST /api/v1/extension/watch/heartbeat`
5. `WatchService.Heartbeat()` calculates elapsed time, awards points if threshold met
6. `PointsService` atomically updates PointsLedger: spendable_balance ↑, cumulative_total ↑
7. Entry created in PointsTransaction with TxSource="watch_time", linked to WatchSessionID
8. Extension calls `POST /api/v1/extension/watch/end` to terminate session
9. Viewer fetches balance via `GET /api/v1/extension/watch/balance` (WatchService.GetBalance)
10. Dashboard queries `GET /api/v1/users/me/points` (PointsService.GetBalance)

**Bits Redemption Flow:**

1. Streamer's Twitch Bits transaction triggers extension callback
2. Extension calls `POST /api/v1/extension/bits/complete` with bits amount
3. `ExtensionHandler.BitsComplete()` verifies JWT signature (ExtensionSecret)
4. `PointsService.DeductPoints()` creates PointsTransaction with TxSource="bits" (note: no WatchSessionID)
5. Note field records bits amount for audit trail

**Authentication Flow:**

1. User initiates OAuth via dashboard: `GET /api/v1/auth/twitch` (or `/google`, `/web3/nonce`)
2. Browser redirected to provider; user authorizes
3. Provider redirects to callback endpoint with auth code
4. `AuthHandler.TwitchCallback()` (or equivalent) exchanges code for provider access token
5. `AuthService` queries provider API for user email, creates or links User record
6. `AuthService.Login()` generates JWT pair: AccessToken (15 min) + RefreshToken (30 days)
7. RefreshToken persisted in database; AccessToken sent to client
8. Client attaches AccessToken in `Authorization: Bearer <token>` header
9. `JWTAuth` middleware validates and extracts Claims (UserID, Role) for every request

**State Management:**

- **Stateless server:** No session storage; all auth state encoded in JWT
- **Client-side state:** Access token cached in memory/localStorage; refresh token stored securely
- **DB state:** RefreshTokens persisted to prevent replay; points/watch data permanently logged

## Key Abstractions

**PointsBalance:**
- Purpose: Read model representing a viewer's points in a channel
- Examples: `backend/internal/services/points_service.go` lines 19-23
- Pattern: Value object with CumulativeTotal and SpendableBalance fields; hydrated from PointsLedger

**TokenPair:**
- Purpose: Encapsulate JWT and refresh token pair returned at login
- Examples: `backend/internal/services/auth_service.go` lines 38-42
- Pattern: Value object with AccessToken, RefreshToken, ExpiresIn; sent in login response

**Claims:**
- Purpose: JWT payload structure for authentication context
- Examples: `backend/internal/services/auth_service.go` lines 44-48
- Pattern: Extends jwt.RegisteredClaims with UserID and Role for role-based access control

**UserRole:**
- Purpose: Enum constraining user types (viewer, streamer, agency, admin)
- Examples: `backend/internal/models/user.go` lines 10-17
- Pattern: String-based enum mapped to PostgreSQL user_role ENUM type

**TxSource:**
- Purpose: Enum indicating transaction origin (bits, watch_time, spend)
- Examples: `backend/internal/models/points.go` lines 11-17
- Pattern: String-based enum mapped to varchar(50) for forward compatibility

## Entry Points

**Backend Server:**
- Location: `backend/cmd/server/main.go`
- Triggers: Docker container startup or `go run ./cmd/server`
- Responsibilities:
  - Load config from .env
  - Connect to PostgreSQL
  - Create ENUM types (user_role)
  - Run GORM AutoMigrate on all models
  - Create manual partial indexes (watch_sessions, points_ledgers)
  - Wire all services with dependency injection
  - Initialize router with CORS config
  - Listen on :8080 (or PORT env var)

**Dashboard Frontend:**
- Location: `dashboard/src/main.tsx` → `dashboard/src/App.tsx`
- Triggers: Browser load of http://localhost:5174 (dev)
- Responsibilities:
  - Initialize React 19 app with React Router
  - Mount ProtectedRoute wrapper (checks localStorage for JWT)
  - Render Layout with sidebar navigation (Dashboard, Streamers, Transactions, Settings)
  - Route requests to pages (DashboardPage, StreamersPage, etc.)

**Twitch Extension:**
- Location: `tachimint/src/main.tsx` → `tachimint/src/App.tsx`
- Triggers: Twitch extension panel loaded on streamer/viewer's page
- Responsibilities:
  - Initialize Twitch extension context via `window.Twitch.ext.onContext()`
  - Branch rendering: broadcaster view vs. viewer view
  - Initialize hooks: useTwitch (auth), useHeartbeat (watch-time), useBits (redemption)
  - Render balance display with animation feedback

## Error Handling

**Strategy:** Explicit error types in service layer; HTTP status codes in handlers

**Patterns:**
- Service-level error definitions: `ErrUserNotFound`, `ErrInvalidToken`, `ErrInsufficientBalance` (see `backend/internal/services/auth_service.go` lines 26-36, `backend/internal/services/points_service.go` lines 14-16)
- Handler maps domain errors to HTTP status: 400 Bad Request, 401 Unauthorized, 409 Conflict
- Database errors wrapped with context; transactions rolled back on error
- Frontend errors caught in API service layer with retry logic for network failures

## Cross-Cutting Concerns

**Logging:**
- Backend: Gin middleware logs every HTTP request with response time
- Database: GORM logger set to Info level; logs SQL queries
- Frontend: Console logs in hooks for auth state, balance updates

**Validation:**
- Backend: Handler layer unmarshals JSON → Go struct with struct tags (gorm/json)
- Database: GORM model validation (required fields, enum types enforced at DB level)
- Frontend: React client-side validation on LoginPage before API calls

**Authentication:**
- Backend: JWTAuth middleware extracts Bearer token, validates signature, populates context
- Extension: Verifies Twitch Extension JWT (ExtensionSecret in config)
- Frontend: ProtectedRoute component redirects unauthenticated users to LoginPage

**CORS:**
- Backend: CORS middleware configured with ALLOWED_ORIGINS env var (defaults to localhost:3000, localhost:5173)
- Allows credentials: true for cookie-based auth (if needed in future)

---

*Architecture analysis: 2026-04-04*
