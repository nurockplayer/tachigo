# Agency Onboarding Recovery Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 補齊 agency onboarding partial success 情境的補救入口、狀態可見性與可觀測性。

**Architecture:** 新增 `GET /agencies/:id`（回傳 agency profile + `onboarding_complete` 衍生狀態）與 `POST /agencies/:id/resend-setup`（admin 重送密碼設定信），onboarding 狀態由 `password_hash IS NULL` 衍生，不新增 schema 欄位。改善既有 Create handler 的 log 訊息為結構化格式。

**Tech Stack:** Go 1.22、Gin、GORM、SQLite (test)、`services.EmailAuthService.ForgotPassword`

**refs #146**

---

## 檔案異動清單

| 動作 | 路徑 | 說明 |
|---|---|---|
| Modify | `backend/internal/services/agency_service.go` | 新增 `GetByID`，回傳 agency user 與 onboarding 狀態 |
| Modify | `backend/internal/handlers/agency_handler.go` | 新增 `Get`、`ResendSetup` handler；改善 Create log |
| Modify | `backend/internal/handlers/agency_handler_test.go` | 新增對應測試 |
| Modify | `backend/internal/router/router.go` | 掛載兩條新路由 |

---

## Task 1：AgencyService.GetByID

**Files:**
- Modify: `backend/internal/services/agency_service.go`

- [ ] **Step 1：寫失敗測試**

在 `backend/internal/services/agency_service_test.go` 新增：

```go
func TestAgencyService_GetByID_Found(t *testing.T) {
    db := setupTestDB(t)
    svc := NewAgencyService(db)

    id := uuid.New()
    name := "test-agency"
    email := "ta@example.com"
    if err := db.Exec(
        `INSERT INTO users (id, username, email, role, is_active, email_verified, password_hash, created_at, updated_at)
         VALUES (?, ?, ?, 'agency', 1, 1, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
        id, name, email,
    ).Error; err != nil {
        t.Fatalf("seed: %v", err)
    }

    user, complete, err := svc.GetByID(id)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.ID != id {
        t.Fatalf("expected id %v, got %v", id, user.ID)
    }
    if complete {
        t.Fatal("expected onboarding_complete=false when password_hash IS NULL")
    }
}

func TestAgencyService_GetByID_Complete(t *testing.T) {
    db := setupTestDB(t)
    svc := NewAgencyService(db)

    id := uuid.New()
    name := "done-agency"
    email := "done@example.com"
    if err := db.Exec(
        `INSERT INTO users (id, username, email, role, is_active, email_verified, password_hash, created_at, updated_at)
         VALUES (?, ?, ?, 'agency', 1, 1, 'hashed', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
        id, name, email,
    ).Error; err != nil {
        t.Fatalf("seed: %v", err)
    }

    _, complete, err := svc.GetByID(id)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !complete {
        t.Fatal("expected onboarding_complete=true when password_hash IS NOT NULL")
    }
}

func TestAgencyService_GetByID_NotFound(t *testing.T) {
    db := setupTestDB(t)
    svc := NewAgencyService(db)

    _, _, err := svc.GetByID(uuid.New())
    if !errors.Is(err, ErrAgencyNotFound) {
        t.Fatalf("expected ErrAgencyNotFound, got %v", err)
    }
}
```

- [ ] **Step 2：跑測試確認失敗**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -run TestAgencyService_GetByID -v
```

預期：FAIL（method 不存在）

- [ ] **Step 3：實作 GetByID**

在 `backend/internal/services/agency_service.go` 新增：

```go
// GetByID returns the agency user and whether onboarding is complete.
// onboardingComplete is true when the agency has set a password (password_hash IS NOT NULL).
// Returns ErrAgencyNotFound if no user with the given id and role=agency exists.
func (s *AgencyService) GetByID(id uuid.UUID) (*models.User, bool, error) {
    var user models.User
    if err := s.db.Where("id = ? AND role = ?", id, models.RoleAgency).First(&user).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, false, ErrAgencyNotFound
        }
        return nil, false, err
    }
    onboardingComplete := user.PasswordHash != nil
    return &user, onboardingComplete, nil
}
```

- [ ] **Step 4：跑測試確認通過**

```bash
docker compose run --no-deps --rm app go test ./internal/services/... -run TestAgencyService_GetByID -v
```

預期：PASS

- [ ] **Step 5：Commit**

```bash
git add backend/internal/services/agency_service.go backend/internal/services/agency_service_test.go
git commit -m "feat: add AgencyService.GetByID with onboarding status

refs #146

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 2：GET /agencies/:id handler

**Files:**
- Modify: `backend/internal/handlers/agency_handler.go`
- Modify: `backend/internal/handlers/agency_handler_test.go`

- [ ] **Step 1：寫失敗測試**

在 `backend/internal/handlers/agency_handler_test.go` 中，先在 `newAgencyTestEnv` 路由組加掛新路由（Task 4 前先在 test helper 裡掛），或建立獨立 helper。在此新增：

```go
func newAgencyTestEnvWithGet(t *testing.T) (*testEnv, http.Handler) {
    t.Helper()
    env := newTestEnv(t)
    agencySvc := services.NewAgencyService(env.db)
    agencyH := handlers.NewAgencyHandler(agencySvc, env.emailAuthSvc)

    r := env.router
    v1 := r.Group("/api/v1")
    agencies := v1.Group("/agencies")
    agencies.Use(middleware.JWTAuth(env.authSvc))
    agencies.POST("", middleware.RequireRole(models.RoleAdmin), agencyH.Create)
    agencies.GET("/:id", middleware.RequireRole(models.RoleAgency, models.RoleAdmin), agencyH.Get)
    agencies.PUT("/:id/settings", middleware.RequireRole(models.RoleAgency, models.RoleAdmin), agencyH.UpdateSettings)
    agencies.GET("/:id/streamers", middleware.RequireRole(models.RoleAgency, models.RoleAdmin), agencyH.ListStreamers)
    return env, r
}

func TestAgencyHandler_Get_ReturnsProfile(t *testing.T) {
    env, r := newAgencyTestEnvWithGet(t)
    agencyID := seedAgencyUser(t, env.db, "agency-get", "agency-get@example.com")

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String(), nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
    resp := parseBody(t, w.Body.Bytes())
    data := resp["data"].(map[string]interface{})
    if data["id"] == nil {
        t.Fatal("expected id in response")
    }
    if data["email"] != "agency-get@example.com" {
        t.Fatalf("expected email agency-get@example.com, got %v", data["email"])
    }
    // seedAgencyUser inserts with no password_hash → onboarding_complete must be false
    if data["onboarding_complete"] != false {
        t.Fatalf("expected onboarding_complete=false, got %v", data["onboarding_complete"])
    }
}

func TestAgencyHandler_Get_NotFound(t *testing.T) {
    _, r := newAgencyTestEnvWithGet(t)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/agencies/"+uuid.NewString(), nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAgencyHandler_Get_AgencyCanQueryOwn(t *testing.T) {
    env, r := newAgencyTestEnvWithGet(t)
    agencyID := seedAgencyUser(t, env.db, "agency-self-get", "agency-self-get@example.com")

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String(), nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, agencyID, models.RoleAgency))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAgencyHandler_Get_AgencyCannotQueryOthers(t *testing.T) {
    env, r := newAgencyTestEnvWithGet(t)
    agencyID := seedAgencyUser(t, env.db, "agency-other-get", "agency-other-get@example.com")

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String(), nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, uuid.New(), models.RoleAgency))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusForbidden {
        t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
    }
}
```

- [ ] **Step 2：跑測試確認失敗**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... -run TestAgencyHandler_Get -v
```

預期：FAIL（`agencyH.Get` 不存在）

- [ ] **Step 3：實作 Get handler**

在 `backend/internal/handlers/agency_handler.go` 新增 response 型別與 handler：

```go
type getAgencyResponse struct {
    ID                 uuid.UUID `json:"id"`
    Name               string    `json:"name"`
    Email              string    `json:"email"`
    OnboardingComplete bool      `json:"onboarding_complete"`
}

func (h *AgencyHandler) Get(c *gin.Context) {
    agencyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        badRequest(c, "invalid agency id")
        return
    }

    claims := middleware.MustClaims(c)
    if claims.Role == models.RoleAgency && claims.UserID != agencyID.String() {
        c.JSON(http.StatusForbidden, Response{Success: false, Error: "forbidden"})
        return
    }

    user, complete, err := h.agencySvc.GetByID(agencyID)
    if err != nil {
        if errors.Is(err, services.ErrAgencyNotFound) {
            notFound(c, "agency not found")
            return
        }
        log.Printf("agency get: unexpected error for id=%s: %v", agencyID, err)
        internal(c)
        return
    }

    name := ""
    if user.Username != nil {
        name = *user.Username
    }
    email := ""
    if user.Email != nil {
        email = *user.Email
    }

    ok(c, getAgencyResponse{
        ID:                 user.ID,
        Name:               name,
        Email:              email,
        OnboardingComplete: complete,
    })
}
```

- [ ] **Step 4：跑測試確認通過**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... -run TestAgencyHandler_Get -v
```

預期：PASS

- [ ] **Step 5：Commit**

```bash
git add backend/internal/handlers/agency_handler.go backend/internal/handlers/agency_handler_test.go
git commit -m "feat: add GET /agencies/:id with onboarding_complete status

refs #146

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3：POST /agencies/:id/resend-setup handler

**Files:**
- Modify: `backend/internal/handlers/agency_handler.go`
- Modify: `backend/internal/handlers/agency_handler_test.go`

- [ ] **Step 1：寫失敗測試**

在 `agency_handler_test.go` 的 `newAgencyTestEnvWithGet` 加掛 resend 路由，或建立完整版 helper（含所有路由）。建議直接把 `newAgencyTestEnvWithGet` 改名並補上 resend 路由：

```go
// 將 newAgencyTestEnvWithGet 改為 newFullAgencyTestEnv，補上 resend 路由：
agencies.POST("/:id/resend-setup", middleware.RequireRole(models.RoleAdmin), agencyH.ResendSetup)
```

新增測試：

```go
func TestAgencyHandler_ResendSetup_Success(t *testing.T) {
    env, r := newFullAgencyTestEnv(t)
    agencyID := seedAgencyUser(t, env.db, "agency-resend", "agency-resend@example.com")

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/agencies/"+agencyID.String()+"/resend-setup", nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }

    // A password_resets token must be written
    var count int64
    if err := env.db.Table("password_resets").
        Where("email = ?", "agency-resend@example.com").
        Count(&count).Error; err != nil {
        t.Fatalf("query password_resets: %v", err)
    }
    if count != 1 {
        t.Fatalf("expected 1 password_resets row, got %d", count)
    }
}

func TestAgencyHandler_ResendSetup_NotFound(t *testing.T) {
    _, r := newFullAgencyTestEnv(t)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/agencies/"+uuid.NewString()+"/resend-setup", nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAgencyHandler_ResendSetup_RequiresAdmin(t *testing.T) {
    env, r := newFullAgencyTestEnv(t)
    agencyID := seedAgencyUser(t, env.db, "agency-resend-auth", "agency-resend-auth@example.com")

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/agencies/"+agencyID.String()+"/resend-setup", nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, agencyID, models.RoleAgency))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusForbidden {
        t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
    }
}

func TestAgencyHandler_ResendSetup_InvalidID(t *testing.T) {
    _, r := newFullAgencyTestEnv(t)

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/agencies/not-a-uuid/resend-setup", nil)
    req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
    }
}
```

- [ ] **Step 2：跑測試確認失敗**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... -run TestAgencyHandler_ResendSetup -v
```

預期：FAIL

- [ ] **Step 3：實作 ResendSetup handler**

在 `backend/internal/handlers/agency_handler.go` 新增：

```go
func (h *AgencyHandler) ResendSetup(c *gin.Context) {
    agencyID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        badRequest(c, "invalid agency id")
        return
    }

    user, _, err := h.agencySvc.GetByID(agencyID)
    if err != nil {
        if errors.Is(err, services.ErrAgencyNotFound) {
            notFound(c, "agency not found")
            return
        }
        log.Printf("agency resend-setup get: id=%s err=%v", agencyID, err)
        internal(c)
        return
    }

    if err := h.emailAuthSvc.ForgotPassword(*user.Email); err != nil {
        if errors.Is(err, services.ErrPasswordResetEmailSend) {
            log.Printf("agency resend-setup: email delivery failed agency_id=%s err=%v", agencyID, err)
        } else {
            log.Printf("agency resend-setup: token write failed agency_id=%s err=%v", agencyID, err)
        }
        c.JSON(http.StatusInternalServerError, Response{Success: false, Error: "failed to send setup email"})
        return
    }

    log.Printf("agency resend-setup: password setup email sent agency_id=%s email=%s", agencyID, *user.Email)
    ok(c, gin.H{"message": "setup email sent"})
}
```

> **注意：** ResendSetup 與 Create 不同——這裡的失敗**應該回 500**，因為 admin 主動要求重送，失敗必須明確告知。

- [ ] **Step 4：跑測試確認通過**

```bash
docker compose run --no-deps --rm app go test ./internal/handlers/... -run TestAgencyHandler_ResendSetup -v
```

預期：PASS

- [ ] **Step 5：Commit**

```bash
git add backend/internal/handlers/agency_handler.go backend/internal/handlers/agency_handler_test.go
git commit -m "feat: add POST /agencies/:id/resend-setup for admin password reset retrigger

refs #146

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4：Wire 新路由 + 改善 Create log

**Files:**
- Modify: `backend/internal/router/router.go`
- Modify: `backend/internal/handlers/agency_handler.go`（log 格式調整）

- [ ] **Step 1：在 router 掛載兩條新路由**

在 `backend/internal/router/router.go` 的 agencies group 新增：

```go
// GET /agencies/:id — agency or admin
agencies.GET("/:id",
    middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
    agencyHandler.Get,
)
// POST /agencies/:id/resend-setup — admin only
agencies.POST("/:id/resend-setup",
    middleware.RequireRole(models.RoleAdmin),
    agencyHandler.ResendSetup,
)
```

- [ ] **Step 2：改善 Create 的 log 格式**

將 `agency_handler.go` 的 Create 中兩條 log 改為結構化格式：

```go
// 舊：
log.Printf("agency create: password setup email not delivered for user %s: %v", user.ID, err)
log.Printf("agency create: password reset token setup failed for user %s: %v", user.ID, err)

// 新：
log.Printf("agency create: email delivery failed agency_id=%s email=%s err=%v", user.ID, *user.Email, err)
log.Printf("agency create: token write failed agency_id=%s email=%s err=%v", user.ID, *user.Email, err)
```

- [ ] **Step 3：跑全部測試確認無 regression**

```bash
docker compose run --no-deps --rm app go test ./...
```

預期：所有測試 PASS

- [ ] **Step 4：Commit**

```bash
git add backend/internal/router/router.go backend/internal/handlers/agency_handler.go
git commit -m "chore: wire GET /agencies/:id and POST /agencies/:id/resend-setup routes; improve log format

refs #146

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 驗證清單

- [ ] `GET /agencies/:id`：admin 可查詢，agency 只能查自己，回傳 `onboarding_complete`
- [ ] `POST /agencies/:id/resend-setup`：admin only，成功寫入 `password_resets` row，失敗回 500
- [ ] 全部測試通過：`go test ./...`
- [ ] Create log 訊息含 `agency_id=` 可 grep 追蹤
