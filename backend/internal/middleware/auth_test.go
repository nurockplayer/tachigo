package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

const testAccessSecret = "test-access-secret-at-least-32-chars!"

func init() {
	gin.SetMode(gin.TestMode)
}

// newAuthSvc returns an AuthService backed by a nil DB.
// ValidateAccessToken only uses the JWT secret, so no DB is needed for these tests.
func newAuthSvc() *services.AuthService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  testAccessSecret,
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
	}
	return services.NewAuthService(nil, cfg)
}

// mintToken creates a signed JWT access token for the given role without touching the DB.
func mintToken(t *testing.T, role models.UserRole) string {
	t.Helper()
	claims := services.Claims{
		UserID: "00000000-0000-0000-0000-000000000001",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "00000000-0000-0000-0000-000000000001",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testAccessSecret))
	if err != nil {
		t.Fatalf("mintToken: %v", err)
	}
	return token
}

// buildRouter creates a minimal gin router with JWTAuth + RequireRole protecting GET /protected.
func buildRouter(authSvc *services.AuthService, allowed ...models.UserRole) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	protected := r.Group("/protected")
	protected.Use(middleware.JWTAuth(authSvc))
	protected.Use(middleware.RequireRole(allowed...))
	protected.GET("", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	return r
}

func doGET(r *gin.Engine, token string) int {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w.Code
}

// ── Unit tests: RequireRole ────────────────────────────────────────────────

func TestRequireRole_AllowsMatchingRole(t *testing.T) {
	authSvc := newAuthSvc()
	router := buildRouter(authSvc, models.RoleAdmin)
	token := mintToken(t, models.RoleAdmin)

	if got := doGET(router, token); got != 200 {
		t.Fatalf("expected 200, got %d", got)
	}
}

func TestRequireRole_ForbidsNonMatchingRole(t *testing.T) {
	authSvc := newAuthSvc()
	router := buildRouter(authSvc, models.RoleAdmin)
	token := mintToken(t, models.RoleViewer)

	if got := doGET(router, token); got != 403 {
		t.Fatalf("expected 403, got %d", got)
	}
}

func TestRequireRole_AllowsAnyOfMultipleRoles(t *testing.T) {
	authSvc := newAuthSvc()
	router := buildRouter(authSvc, models.RoleAgency, models.RoleAdmin)

	for _, role := range []models.UserRole{models.RoleAgency, models.RoleAdmin} {
		token := mintToken(t, role)
		if got := doGET(router, token); got != 200 {
			t.Fatalf("role %s: expected 200, got %d", role, got)
		}
	}
}

func TestRequireRole_ForbidsOtherRolesWhenMultipleAllowed(t *testing.T) {
	authSvc := newAuthSvc()
	router := buildRouter(authSvc, models.RoleAgency, models.RoleAdmin)

	for _, role := range []models.UserRole{models.RoleViewer, models.RoleStreamer} {
		token := mintToken(t, role)
		if got := doGET(router, token); got != 403 {
			t.Fatalf("role %s: expected 403, got %d", role, got)
		}
	}
}

func TestRequireRole_Returns401WithoutToken(t *testing.T) {
	authSvc := newAuthSvc()
	router := buildRouter(authSvc, models.RoleAdmin)

	if got := doGET(router, ""); got != 401 {
		t.Fatalf("expected 401, got %d", got)
	}
}

func TestRequireRole_AllRolesCovered(t *testing.T) {
	// Ensure all four defined roles behave as expected against a single-role gate.
	authSvc := newAuthSvc()
	router := buildRouter(authSvc, models.RoleAdmin)

	cases := []struct {
		role models.UserRole
		want int
	}{
		{models.RoleViewer, 403},
		{models.RoleStreamer, 403},
		{models.RoleAgency, 403},
		{models.RoleAdmin, 200},
	}
	for _, tc := range cases {
		token := mintToken(t, tc.role)
		if got := doGET(router, token); got != tc.want {
			t.Errorf("role %s: expected %d, got %d", tc.role, tc.want, got)
		}
	}
}
