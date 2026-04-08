package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type airdropEnv struct {
	*testEnv
}

func newAirdropTestEnv(t *testing.T) *airdropEnv {
	t.Helper()

	base := newTestEnv(t)
	watchSvc := services.NewWatchService(base.db)
	pointsSvc := services.NewPointsService(base.db, watchSvc)
	configSvc := services.NewChannelConfigService(base.db)
	streamerSvc := services.NewStreamerService(base.db, pointsSvc)
	agencySvc := services.NewAgencyService(base.db)
	airdropSvc := services.NewAirdropService(base.db, pointsSvc, configSvc)
	airdropH := handlers.NewAirdropHandler(airdropSvc, agencySvc, streamerSvc)

	v1 := base.router.Group("/api/v1")
	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.JWTAuth(base.authSvc))
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency))
	{
		dashboard.POST("/channels/:channel_id/airdrop", airdropH.Airdrop)
	}

	return &airdropEnv{testEnv: base}
}

func airdropRequest(t *testing.T, router *gin.Engine, token, channelID, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboard/channels/"+channelID+"/airdrop", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func seedActiveViewerSession(t *testing.T, env *airdropEnv, channelID string, accumulatedSeconds int64) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	if err := env.db.Exec(
		`INSERT INTO users (id, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, 'viewer', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		userID,
	).Error; err != nil {
		t.Fatalf("seed viewer: %v", err)
	}
	if err := env.db.Exec(
		`INSERT INTO watch_sessions (
			id, user_id, channel_id, accumulated_seconds, rewarded_seconds,
			last_heartbeat_at, click_cooldown_until, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, 0, CURRENT_TIMESTAMP, '1970-01-01 00:00:00', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), userID, channelID, accumulatedSeconds,
	).Error; err != nil {
		t.Fatalf("seed watch session: %v", err)
	}
	return userID
}

func seedAgencyChannel(t *testing.T, env *airdropEnv, email, channelID string) {
	t.Helper()

	var user models.User
	if err := env.db.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("load agency user: %v", err)
	}
	if err := env.db.Create(&models.AgencyStreamer{
		AgencyID:  user.ID,
		ChannelID: channelID,
	}).Error; err != nil {
		t.Fatalf("seed agency ownership: %v", err)
	}
}

func TestAirdropHandler_Unauthenticated(t *testing.T) {
	env := newAirdropTestEnv(t)

	w := airdropRequest(t, env.router, "", "channel_123", `{"amount":100}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAirdropHandler_ViewerForbidden(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleViewer)

	w := airdropRequest(t, env.router, token, "channel_123", `{"amount":100}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAirdropHandler_StreamerNonOwnedForbidden(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleStreamer)

	w := airdropRequest(t, env.router, token, "channel_other", `{"amount":100}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAirdropHandler_AgencyNonOwnedForbidden(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleAgency)
	seedAgencyChannel(t, env, "agency_dashboard@example.com", "channel_owned")

	w := airdropRequest(t, env.router, token, "channel_other", `{"amount":100}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAirdropHandler_AmountMustBePositive(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleAdmin)

	w := airdropRequest(t, env.router, token, "channel_live", `{"amount":0}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAirdropHandler_NoActiveViewers(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleAdmin)

	w := airdropRequest(t, env.router, token, "channel_empty", `{"amount":100}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "no active viewers") {
		t.Fatalf("want no active viewers error, got %s", w.Body.String())
	}
}

func TestAirdropHandler_NormalAirdropAdmin(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleAdmin)
	seedActiveViewerSession(t, env, "channel_live", 30)
	seedActiveViewerSession(t, env, "channel_live", 90)

	w := airdropRequest(t, env.router, token, "channel_live", `{"amount":200,"note":"campaign"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"affected_count":2`) {
		t.Fatalf("want affected_count=2, got %s", w.Body.String())
	}
}

func TestAirdropHandler_StreamerOwnedChannelAllowed(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleStreamer)
	seedOwnedStreamerChannel(t, &dashboardEnv{testEnv: env.testEnv}, "streamer_dashboard@example.com", "channel_owned")
	seedActiveViewerSession(t, env, "channel_owned", 120)

	w := airdropRequest(t, env.router, token, "channel_owned", `{"amount":100}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Fatalf("want success response, got %s", w.Body.String())
	}
}

func TestAirdropHandler_AgencyOwnedChannelAllowed(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleAgency)
	seedAgencyChannel(t, env, "agency_dashboard@example.com", "channel_agency_owned")
	seedActiveViewerSession(t, env, "channel_agency_owned", 60)

	w := airdropRequest(t, env.router, token, "channel_agency_owned", `{"amount":100}`)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"affected_count":1`) {
		t.Fatalf("want affected_count=1, got %s", w.Body.String())
	}
}

func TestAirdropHandler_DailyLimitExceeded_ReturnsRemaining(t *testing.T) {
	env := newAirdropTestEnv(t)
	token := (&dashboardEnv{testEnv: env.testEnv}).tokenForRole(t, models.RoleAdmin)
	seedActiveViewerSession(t, env, "channel_limited", 60)

	// Set a tight daily limit.
	if err := env.db.Exec(
		`INSERT INTO channel_configs (channel_id, seconds_per_point, multiplier, daily_airdrop_limit, created_at, updated_at)
		 VALUES ('channel_limited', 60, 1, 500, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	).Error; err != nil {
		t.Fatalf("seed channel config: %v", err)
	}

	// First airdrop takes 400 of 500.
	w := airdropRequest(t, env.router, token, "channel_limited", `{"amount":400}`)
	if w.Code != http.StatusOK {
		t.Fatalf("first airdrop want 200, got %d: %s", w.Code, w.Body.String())
	}

	// Second airdrop of 200 exceeds the remaining 100.
	w = airdropRequest(t, env.router, token, "channel_limited", `{"amount":200}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "daily airdrop limit exceeded") {
		t.Fatalf("want limit exceeded error, got %s", body)
	}
	if !strings.Contains(body, `"remaining":100`) {
		t.Fatalf("want remaining=100 in data, got %s", body)
	}
}
