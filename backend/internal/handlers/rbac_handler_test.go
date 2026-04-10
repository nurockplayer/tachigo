package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

// rbacEnv wraps testEnv and exposes a router with RBAC-protected routes.
type rbacEnv struct {
	*testEnv
}

func newRBACTestEnv(t *testing.T) *rbacEnv {
	t.Helper()
	base := newTestEnv(t)

	// Attach RBAC-protected stubs on top of the existing router.
	v1 := base.router.Group("/api/v1")

	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(base.authSvc))
	{
		agencies.POST("", middleware.RequireRole(models.RoleAdmin), func(c *gin.Context) {
			c.JSON(501, gin.H{"error": "not implemented"})
		})
		agencies.PUT("/:id/settings",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) },
		)
		agencies.GET("/:id/streamers",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) },
		)
	}

	events := v1.Group("/events")
	events.Use(middleware.JWTAuth(base.authSvc))
	events.Use(middleware.RequireRole(models.RoleStreamer, models.RoleAgency, models.RoleAdmin))
	{
		events.POST("/create", func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) })
		events.POST("/:id/settle", func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) })
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(base.authSvc))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.GET("/users", func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) })
	}

	return &rbacEnv{base}
}

// tokenForRole registers a user, updates their role in the DB, then logs in to
// obtain a JWT that carries the requested role in its claims.
func (e *rbacEnv) tokenForRole(t *testing.T, role models.UserRole) string {
	t.Helper()

	email := string(role) + "_user@example.com"
	username := string(role) + "_user"
	password := "password123"

	user, _, err := e.authSvc.Register(services.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("register(%s): %v", role, err)
	}

	// Update role directly in DB, then login so the JWT carries the new role.
	if err := e.db.Model(user).Update("role", role).Error; err != nil {
		t.Fatalf("set role(%s): %v", role, err)
	}

	_, tokens, err := e.authSvc.Login(services.LoginInput{
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("login(%s): %v", role, err)
	}
	return tokens.AccessToken
}

func doRequest(router *gin.Engine, method, path, token string) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(w, req)
	return w.Code
}

// ── GET /admin/users ──────────────────────────────────────────────────────────

// viewer token → 403
func TestRBAC_AdminUsers_ViewerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/admin/users", token); got != http.StatusForbidden {
		t.Errorf("viewer: want 403, got %d", got)
	}
}

// streamer token → 403
func TestRBAC_AdminUsers_StreamerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/admin/users", token); got != http.StatusForbidden {
		t.Errorf("streamer: want 403, got %d", got)
	}
}

// agency token → 403
func TestRBAC_AdminUsers_AgencyForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAgency)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/admin/users", token); got != http.StatusForbidden {
		t.Errorf("agency: want 403, got %d", got)
	}
}

// admin token → 501（handler 尚未實作，但通過授權）
func TestRBAC_AdminUsers_AdminAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/admin/users", token); got != http.StatusNotImplemented {
		t.Errorf("admin: want 501, got %d", got)
	}
}

// 無 token → 401
func TestRBAC_AdminUsers_NoToken(t *testing.T) {
	env := newRBACTestEnv(t)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/admin/users", ""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
}

// ── POST /agencies ────────────────────────────────────────────────────────────

// 無 token → 401
func TestRBAC_CreateAgency_NoToken(t *testing.T) {
	env := newRBACTestEnv(t)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/agencies", ""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
}

// viewer → 403
func TestRBAC_CreateAgency_ViewerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/agencies", token); got != http.StatusForbidden {
		t.Errorf("viewer: want 403, got %d", got)
	}
}

// admin → 501（通過授權）
func TestRBAC_CreateAgency_AdminAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/agencies", token); got != http.StatusNotImplemented {
		t.Errorf("admin: want 501, got %d", got)
	}
}

// ── POST /events/create ───────────────────────────────────────────────────────

// 無 token → 401
func TestRBAC_CreateEvent_NoToken(t *testing.T) {
	env := newRBACTestEnv(t)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/create", ""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
}

// viewer → 403
func TestRBAC_CreateEvent_ViewerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/create", token); got != http.StatusForbidden {
		t.Errorf("viewer: want 403, got %d", got)
	}
}

// streamer → 501（通過授權）
func TestRBAC_CreateEvent_StreamerAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/create", token); got != http.StatusNotImplemented {
		t.Errorf("streamer: want 501, got %d", got)
	}
}

// agency → 501（通過授權）
func TestRBAC_CreateEvent_AgencyAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAgency)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/create", token); got != http.StatusNotImplemented {
		t.Errorf("agency: want 501, got %d", got)
	}
}

// admin → 501（通過授權）
func TestRBAC_CreateEvent_AdminAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/create", token); got != http.StatusNotImplemented {
		t.Errorf("admin: want 501, got %d", got)
	}
}

// ── POST /agencies（補完角色覆蓋）────────────────────────────────────────────

// streamer → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_CreateAgency_StreamerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/agencies", token); got != http.StatusForbidden {
		t.Errorf("streamer: want 403, got %d", got)
	}
}

// agency → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_CreateAgency_AgencyForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAgency)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/agencies", token); got != http.StatusForbidden {
		t.Errorf("agency: want 403, got %d", got)
	}
}

// ── PUT /agencies/:id/settings ────────────────────────────────────────────────

// 無 token → 401
func TestRBAC_UpdateAgencySettings_NoToken(t *testing.T) {
	env := newRBACTestEnv(t)

	if got := doRequest(env.router, http.MethodPut, "/api/v1/agencies/1/settings", ""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
}

// viewer → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_UpdateAgencySettings_ViewerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	if got := doRequest(env.router, http.MethodPut, "/api/v1/agencies/1/settings", token); got != http.StatusForbidden {
		t.Errorf("viewer: want 403, got %d", got)
	}
}

// streamer → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_UpdateAgencySettings_StreamerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	if got := doRequest(env.router, http.MethodPut, "/api/v1/agencies/1/settings", token); got != http.StatusForbidden {
		t.Errorf("streamer: want 403, got %d", got)
	}
}

// agency → 501（通過授權）
func TestRBAC_UpdateAgencySettings_AgencyAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAgency)

	if got := doRequest(env.router, http.MethodPut, "/api/v1/agencies/1/settings", token); got != http.StatusNotImplemented {
		t.Errorf("agency: want 501, got %d", got)
	}
}

// admin → 501（通過授權）
func TestRBAC_UpdateAgencySettings_AdminAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if got := doRequest(env.router, http.MethodPut, "/api/v1/agencies/1/settings", token); got != http.StatusNotImplemented {
		t.Errorf("admin: want 501, got %d", got)
	}
}

// ── GET /agencies/:id/streamers ───────────────────────────────────────────────

// 無 token → 401
func TestRBAC_ListAgencyStreamers_NoToken(t *testing.T) {
	env := newRBACTestEnv(t)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/agencies/1/streamers", ""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
}

// viewer → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_ListAgencyStreamers_ViewerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/agencies/1/streamers", token); got != http.StatusForbidden {
		t.Errorf("viewer: want 403, got %d", got)
	}
}

// streamer → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_ListAgencyStreamers_StreamerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/agencies/1/streamers", token); got != http.StatusForbidden {
		t.Errorf("streamer: want 403, got %d", got)
	}
}

// agency → 501（通過授權）
func TestRBAC_ListAgencyStreamers_AgencyAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAgency)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/agencies/1/streamers", token); got != http.StatusNotImplemented {
		t.Errorf("agency: want 501, got %d", got)
	}
}

// admin → 501（通過授權）
func TestRBAC_ListAgencyStreamers_AdminAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if got := doRequest(env.router, http.MethodGet, "/api/v1/agencies/1/streamers", token); got != http.StatusNotImplemented {
		t.Errorf("admin: want 501, got %d", got)
	}
}

// ── POST /events/:id/settle ───────────────────────────────────────────────────

// 無 token → 401
func TestRBAC_SettleEvent_NoToken(t *testing.T) {
	env := newRBACTestEnv(t)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/1/settle", ""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
}

// viewer → 403（有 token 但角色不符，應回 403 而非 401）
func TestRBAC_SettleEvent_ViewerForbidden(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/1/settle", token); got != http.StatusForbidden {
		t.Errorf("viewer: want 403, got %d", got)
	}
}

// streamer → 501（通過授權）
func TestRBAC_SettleEvent_StreamerAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/1/settle", token); got != http.StatusNotImplemented {
		t.Errorf("streamer: want 501, got %d", got)
	}
}

// agency → 501（通過授權）
func TestRBAC_SettleEvent_AgencyAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAgency)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/1/settle", token); got != http.StatusNotImplemented {
		t.Errorf("agency: want 501, got %d", got)
	}
}

// admin → 501（通過授權）
func TestRBAC_SettleEvent_AdminAllowed(t *testing.T) {
	env := newRBACTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if got := doRequest(env.router, http.MethodPost, "/api/v1/events/1/settle", token); got != http.StatusNotImplemented {
		t.Errorf("admin: want 501, got %d", got)
	}
}
