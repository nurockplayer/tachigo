package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type dashboardEnv struct {
	*testEnv
}

func newDashboardTestEnv(t *testing.T) *dashboardEnv {
	t.Helper()

	base := newTestEnv(t)
	configSvc := services.NewChannelConfigService(base.db)
	configH := handlers.NewChannelConfigHandler(configSvc)

	v1 := base.router.Group("/api/v1")
	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.JWTAuth(base.authSvc))
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer))
	{
		dashboard.PUT("/channels/:channel_id/config", configH.UpdateChannelConfig)
	}

	return &dashboardEnv{testEnv: base}
}

func (e *dashboardEnv) tokenForRole(t *testing.T, role models.UserRole) string {
	t.Helper()

	email := string(role) + "_dashboard@example.com"
	username := string(role) + "_dashboard"
	password := "password123"

	user, _, err := e.authSvc.Register(services.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("register(%s): %v", role, err)
	}

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

func updateChannelConfig(t *testing.T, router *gin.Engine, token, channelID, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboard/channels/"+channelID+"/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUpdateChannelConfig_NoToken(t *testing.T) {
	env := newDashboardTestEnv(t)

	w := updateChannelConfig(t, env.router, "", "channel_123", `{"seconds_per_point":45}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateChannelConfig_ViewerForbidden(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":45}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateChannelConfig_StreamerAllowed(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":45}`)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	if resp["success"] != true {
		t.Fatalf("expected success=true, got %#v", resp["success"])
	}

	data, _ := resp["data"].(map[string]interface{})
	config, _ := data["config"].(map[string]interface{})
	if config["channel_id"] != "channel_123" {
		t.Fatalf("channel_id: want channel_123, got %v", config["channel_id"])
	}
	if config["seconds_per_point"] != float64(45) {
		t.Fatalf("seconds_per_point: want 45, got %v", config["seconds_per_point"])
	}

	var saved models.ChannelConfig
	if err := env.db.First(&saved, "channel_id = ?", "channel_123").Error; err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if saved.SecondsPerPoint != 45 {
		t.Fatalf("saved seconds_per_point: want 45, got %d", saved.SecondsPerPoint)
	}
}

func TestUpdateChannelConfig_AdminCanUpdateExistingConfig(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if err := env.db.Create(&models.ChannelConfig{
		ChannelID:       "channel_123",
		SecondsPerPoint: 60,
	}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":90}`)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var saved models.ChannelConfig
	if err := env.db.First(&saved, "channel_id = ?", "channel_123").Error; err != nil {
		t.Fatalf("load updated config: %v", err)
	}
	if saved.SecondsPerPoint != 90 {
		t.Fatalf("updated seconds_per_point: want 90, got %d", saved.SecondsPerPoint)
	}
}

func TestUpdateChannelConfig_RejectsInvalidBody(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":0}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}
