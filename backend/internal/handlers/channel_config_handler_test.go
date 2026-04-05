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
	watchSvc := services.NewWatchService(base.db)
	pointsSvc := services.NewPointsService(base.db, watchSvc)
	configSvc := services.NewChannelConfigService(base.db)
	streamerSvc := services.NewStreamerService(base.db, pointsSvc)
	configH := handlers.NewChannelConfigHandler(configSvc, streamerSvc)

	v1 := base.router.Group("/api/v1")
	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.JWTAuth(base.authSvc))
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency))
	{
		dashboard.GET("/channels/:channel_id/config", configH.GetChannelConfig)
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

func getChannelConfig(t *testing.T, router *gin.Engine, token, channelID string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/channels/"+channelID+"/config", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func seedOwnedStreamerChannel(t *testing.T, env *dashboardEnv, email, channelID string) {
	t.Helper()

	var user models.User
	if err := env.db.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("load user by email: %v", err)
	}
	if err := env.db.Create(&models.Streamer{
		UserID:      user.ID,
		ChannelID:   channelID,
		DisplayName: "Owned channel",
	}).Error; err != nil {
		t.Fatalf("seed streamer ownership: %v", err)
	}
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

func TestGetChannelConfig_OK(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if err := env.db.Create(&models.ChannelConfig{
		ChannelID:       "channel_123",
		SecondsPerPoint: 60,
		Multiplier:      3,
	}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w := getChannelConfig(t, env.router, token, "channel_123")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetChannelConfig_StreamerOwned(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)
	seedOwnedStreamerChannel(t, env, "streamer_dashboard@example.com", "channel_owned")

	w := getChannelConfig(t, env.router, token, "channel_owned")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetChannelConfig_StreamerForbidden(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	w := getChannelConfig(t, env.router, token, "channel_other")
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateChannelConfig_WithMultiplier(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":60,"multiplier":3}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var saved models.ChannelConfig
	if err := env.db.First(&saved, "channel_id = ?", "channel_123").Error; err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if saved.SecondsPerPoint != 60 || saved.Multiplier != 3 {
		t.Fatalf("unexpected saved config: %+v", saved)
	}
}

func TestUpdateChannelConfig_StreamerForbidden(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	w := updateChannelConfig(t, env.router, token, "channel_other", `{"seconds_per_point":60,"multiplier":2}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateChannelConfig_StreamerAllowed(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)
	seedOwnedStreamerChannel(t, env, "streamer_dashboard@example.com", "channel_123")

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":45}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var saved models.ChannelConfig
	if err := env.db.First(&saved, "channel_id = ?", "channel_123").Error; err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if saved.SecondsPerPoint != 45 || saved.Multiplier != 1 {
		t.Fatalf("unexpected saved config: %+v", saved)
	}
}

func TestUpdateChannelConfig_AdminCanUpdateExistingConfig(t *testing.T) {
	env := newDashboardTestEnv(t)
	token := env.tokenForRole(t, models.RoleAdmin)

	if err := env.db.Create(&models.ChannelConfig{
		ChannelID:       "channel_123",
		SecondsPerPoint: 60,
		Multiplier:      1,
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

	w := updateChannelConfig(t, env.router, token, "channel_123", `{"seconds_per_point":-1}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}
