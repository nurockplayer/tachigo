# Testing Patterns

## Backend (Go)

### Framework & Runner

- Standard library `testing` package
- `go test ./...` — run all tests
- In Docker: `docker compose run --no-deps --rm app go test ./...`
- No external test framework (no testify, no gomock)

### Test DB Strategy

Tests use **in-memory SQLite** via `gorm.io/driver/sqlite` — not PostgreSQL.

```go
// backend/internal/services/testutil_test.go
func newTestDB(t *testing.T) *gorm.DB {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    db.AutoMigrate(&models.User{}, &models.AuthProvider{}, ...)
    return db
}
```

This means:
- Tests run fast without Docker dependency
- PostgreSQL-specific behavior (UUID types, JSON ops) is not tested
- Risk: SQLite/PostgreSQL divergence can mask migration issues

### Test File Organization

Sibling files in the same package:

```
backend/internal/services/
├── points_service.go
├── points_service_test.go      # same package (services)
├── watch_service.go
├── watch_service_test.go
└── testutil_test.go            # shared helpers (newTestDB, seedViewer, seedStreamer)

backend/internal/handlers/
├── auth_handler.go
├── auth_handler_test.go
└── testutil_test.go            # shared Gin test router setup
```

### Test Helper Patterns

```go
// Seed helpers create test fixtures
func seedViewer(t *testing.T, svc *PointsService) uuid.UUID
func seedStreamer(t *testing.T, svc *PointsService, channelID string) uuid.UUID

// Service constructors accept *gorm.DB for injection
func NewPointsService(db *gorm.DB, watchSvc *WatchService) *PointsService
```

### Handler Tests

Handler tests use `httptest.NewRecorder()` and a test Gin engine:

```go
// Typical handler test pattern
func TestGetUser(t *testing.T) {
    db := newTestDB(t)
    router := setupTestRouter(db)
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/users/me", nil)
    req.Header.Set("Authorization", "Bearer "+testJWT)
    router.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}
```

### Test Coverage

**Covered:**
- `services/points_service_test.go` — earn, spend, balance queries, dual-ledger
- `services/watch_service_test.go` — session start/end, watch time accumulation
- `services/auth_service_test.go` — Twitch JWT validation, token refresh
- `services/email_auth_service_test.go` — email OTP flow
- `services/user_service_test.go` — user CRUD
- `services/address_service_test.go` — address management
- `handlers/auth_handler_test.go` — login/logout/refresh endpoints
- `handlers/rbac_handler_test.go` — role assignment
- `handlers/channel_config_handler_test.go` — channel config CRUD
- `middleware/auth_test.go` — JWT middleware validation

**Not covered:**
- Dashboard auth persistence (in-memory token on page refresh)
- Watch service concurrency (simultaneous session starts)
- Email SMTP integration (unit-mocked only)
- Admin-only routes under real RBAC enforcement

## Frontend

### Framework

Neither `tachimint` nor `dashboard` have test files currently.

- No Vitest, Jest, or React Testing Library configured
- `tachimint/src/mock/` contains mock API response data used for manual dev testing only

### Recommended Setup (not yet implemented)

```
Vitest + React Testing Library
pnpm add -D vitest @testing-library/react @testing-library/user-event jsdom
```

## CI

`.github/workflows/ci.yml` — runs on push/PR:

```yaml
# Runs backend Go tests
docker compose run --no-deps --rm app go test ./...
```

Frontend linting/build validation may also be included — check `ci.yml` for current steps.

## Key Risks

1. **SQLite ≠ PostgreSQL** — UUID v7 columns, JSON operators, and index behavior differ. A passing SQLite test does not guarantee PostgreSQL compatibility.
2. **No frontend tests** — dashboard auth flow, API error states, and UI interactions are not automatically verified.
3. **No E2E tests** — full Twitch Extension → backend flow is only tested manually.
