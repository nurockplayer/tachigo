package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

func newStreamerDashboardEnv(t *testing.T) *dashboardEnv {
	t.Helper()

	base := newTestEnv(t)
	watchSvc := services.NewWatchService(base.db)
	pointsSvc := services.NewPointsService(base.db, watchSvc)
	streamerSvc := services.NewStreamerService(base.db, pointsSvc)
	streamerH := handlers.NewStreamerHandler(streamerSvc)
	configSvc := services.NewChannelConfigService(base.db)
	configH := handlers.NewChannelConfigHandler(configSvc)

	v1 := base.router.Group("/api/v1")
	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.JWTAuth(base.authSvc))
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer))
	{
		dashboard.POST("/streamers/register", streamerH.Register)
		dashboard.GET("/streamers/channels", streamerH.ListChannels)
		dashboard.GET("/channels/:channel_id/stats", streamerH.GetChannelStats)
		dashboard.PUT("/channels/:channel_id/config", configH.UpdateChannelConfig)
	}

	return &dashboardEnv{testEnv: base}
}

func dashboardRequest(t *testing.T, router *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestStreamerRegister_AndListChannels(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers/register", token, `{"channel_id":"ch_123","display_name":"測試實況主"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("register want 200, got %d: %s", w.Code, w.Body.String())
	}

	w = dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/channels", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	channels, _ := data["channels"].([]interface{})
	if len(channels) != 1 {
		t.Fatalf("want 1 channel, got %d", len(channels))
	}
	channel, _ := channels[0].(map[string]interface{})
	if channel["channel_id"] != "ch_123" {
		t.Fatalf("channel_id: want ch_123, got %v", channel["channel_id"])
	}
}

func TestStreamerStats_ForbiddenForViewer(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	token := env.tokenForRole(t, models.RoleViewer)

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/channels/ch_123/stats", token, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStreamerStats_OK(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)

	var streamer models.User
	if err := env.db.Where("email = ?", "streamer_dashboard@example.com").First(&streamer).Error; err != nil {
		t.Fatalf("load streamer: %v", err)
	}
	if err := env.db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), streamer.ID, models.ProviderTwitch, "ch_123",
	).Error; err != nil {
		t.Fatalf("seed auth provider: %v", err)
	}
	if err := env.db.Create(&models.BroadcastTimeLog{
		StreamerID: streamer.ID,
		ChannelID:  "ch_123",
		Seconds:    33,
		RecordedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/channels/ch_123/stats", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	stats, _ := data["stats"].(map[string]interface{})
	if stats["daily_seconds"] != float64(33) {
		t.Fatalf("daily_seconds: want 33, got %v", stats["daily_seconds"])
	}
}
