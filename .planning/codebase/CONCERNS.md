# Codebase Concerns

## Technical Debt

### Dashboard Auth — In-Memory Token Storage (HIGH)

**File:** `dashboard/src/services/auth.ts`

Access tokens are stored only in memory (React state / module variable). Page refresh loses the session entirely — users must log in again.

- Acknowledged as MVP limitation in `plans/dashboard-auth.md`
- Blocks stable dashboard usage in multi-tab / reload scenarios
- Fix: use `localStorage` or `sessionStorage` with refresh token rotation

### UUID v7 Migration — Partial (MEDIUM)

**Files:** `backend/migrations/003_watch_points.sql`, `docs/uuid-v7.md`

UUID v7 is implemented for watch-related tables but not applied to core tables (`users`, `auth_providers`). Creates inconsistency:

- Some tables use UUID v7 (ordered, time-sortable)
- Core tables remain UUID v4 (random)
- `plans/uuid-v7-migration.md` documents the incomplete migration plan

### Duplicate Migration Prefix (LOW)

**Files:** `backend/migrations/004_channel_config.sql`, `backend/migrations/004_rbac_roles.sql`

Two migrations share prefix `004_`. GORM auto-migrate may apply both, but manual migration runners or tools may conflict. Future migrations should start at `005_`.

---

## Known Issues

### Silent Failure in AddBroadcastTime (MEDIUM)

**File:** `backend/internal/services/points_service.go` (approximate)

`AddBroadcastTime()` returns `nil` when the channel has no registered streamer, with no log entry or metric. Callers cannot distinguish "no streamer" from "success."

### Authorization Gap — Channel Config Ownership (MEDIUM)

**File:** `backend/internal/handlers/channel_config_handler.go`

The channel config API allows any user with `streamer` role to edit any channel config — there is no ownership check verifying the requesting streamer owns that channel.

---

## Security Considerations

### Default JWT Secrets in Config (HIGH — if deployed without override)

**File:** `backend/internal/config/config.go`

Default values `"change-me-*-secret"` are present for JWT signing keys. No startup validation verifies that env vars were actually set. A misconfigured deployment would use predictable secrets.

Mitigation: ensure `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET` are required env vars with startup panic if unset.

### No Dashboard Auth Persistence = Forced Re-Login (MEDIUM)

Related to the in-memory token concern above. While not a direct security hole, the lack of refresh token persistence means users frequently re-authenticate, potentially weakening token hygiene practices.

### RBAC Permission Matrix Undefined (MEDIUM)

**File:** `backend/internal/middleware/auth.go`, handler RBAC checks

RBAC roles (`viewer`, `streamer`, `admin`) exist in the DB, but the full permission matrix (which role can do what) is not documented or centrally enforced. Individual handlers check roles ad-hoc, making it easy to miss an endpoint.

---

## Performance Bottlenecks

### Missing Indexes on broadcast_time_logs (MEDIUM)

**File:** `backend/migrations/003_watch_points.sql`

Queries filtering by `(viewer_id, channel_id, period)` on `broadcast_time_logs` lack explicit composite indexes. At scale, these queries will do full table scans.

### No Caching on Balance Query (LOW)

`PointsBalance` is computed from the dual-ledger on every request. No Redis or in-process caching layer. Acceptable at MVP scale but will degrade under concurrent viewers.

### Connection Pool Not Tuned (LOW)

**File:** `backend/internal/database/db.go`

GORM connection pool uses Go defaults. No `SetMaxOpenConns`, `SetMaxIdleConns`, or `SetConnMaxLifetime` configured. May cause connection exhaustion under load.

---

## Fragile Areas

### Stale Watch Session Cleanup

Watch sessions that don't cleanly close (client disconnect, crash) are not automatically expired. There is no background job or TTL to clean up orphaned sessions. Stale sessions can inflate watch-time points.

### Email Auth — Untested in Production

Email OTP flow is unit-tested with mocked SMTP. No integration test or staging verification confirms real emails are delivered. SMTP config errors would silently fail OTP sends.

### tachimint dist/ Committed to Repo

`tachimint/dist/` is committed as a build artifact. This creates:
- Large diffs on every frontend build
- Risk of stale dist if build step is skipped before commit
- Merge conflicts on `dist/assets/` hashes

---

## Scaling Limits

- No queue/worker for async tasks (points calculation is synchronous in request path)
- No rate limiting on auth endpoints (OTP spam, token refresh abuse)
- Single PostgreSQL instance — no read replica or sharding strategy

---

## Missing Critical Features

- Dashboard: no real-time updates (polling or WebSocket) for live stream data
- Backend: no audit log for admin actions (role changes, config edits)
- Backend: no soft-delete pattern — records are hard-deleted

---

## Test Coverage Gaps

- Dashboard: zero automated tests
- Watch service: no concurrency tests (parallel session starts for same viewer/channel)
- Email service: no real SMTP integration test
- Admin endpoints: RBAC enforcement not verified in handler tests
