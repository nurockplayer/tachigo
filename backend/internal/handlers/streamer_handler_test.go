package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

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

	v1 := base.router.Group("/api/v1")
	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.JWTAuth(base.authSvc))
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency))
	{
		dashboard.POST("/streamers", middleware.RequireRole(models.RoleAdmin), streamerH.Create)
		dashboard.GET("/streamers", middleware.RequireRole(models.RoleAgency, models.RoleAdmin), streamerH.List)
		dashboard.GET("/streamers/:streamer_id/stats",
			middleware.RequireRole(models.RoleStreamer, models.RoleAgency, models.RoleAdmin),
			streamerH.GetStats)
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

func createDashboardUser(t *testing.T, env *dashboardEnv, role models.UserRole, prefix string) (models.User, string) {
	t.Helper()

	email := prefix + "@example.com"
	username := prefix
	password := "password123"

	user, _, err := env.authSvc.Register(services.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("register %s: %v", prefix, err)
	}
	if err := env.db.Model(user).Update("role", role).Error; err != nil {
		t.Fatalf("set role %s: %v", prefix, err)
	}

	_, tokens, err := env.authSvc.Login(services.LoginInput{Email: email, Password: password})
	if err != nil {
		t.Fatalf("login %s: %v", prefix, err)
	}
	return *user, tokens.AccessToken
}

func seedTwitchProviderForUser(t *testing.T, env *dashboardEnv, userID uuid.UUID, twitchLogin string) {
	t.Helper()
	if err := env.db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), userID, models.ProviderTwitch, twitchLogin,
	).Error; err != nil {
		t.Fatalf("seed auth provider: %v", err)
	}
}

func seedStreamerRow(t *testing.T, env *dashboardEnv, userID uuid.UUID, agencyUserID *uuid.UUID, twitchLogin string) *models.Streamer {
	t.Helper()
	streamer := &models.Streamer{
		UserID:       userID,
		AgencyUserID: agencyUserID,
		TwitchLogin:  twitchLogin,
	}
	if err := env.db.Create(streamer).Error; err != nil {
		t.Fatalf("seed streamer: %v", err)
	}
	return streamer
}

func TestCreate_AdminOK(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "streamer_admin")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_target")
	agencyUser, _ := createDashboardUser(t, env, models.RoleAgency, "agency_target")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "target_login")

	body := `{"user_id":"` + streamerUser.ID.String() + `","agency_user_id":"` + agencyUser.ID.String() + `","twitch_login":"target_login"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", adminToken, body)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamer := data["streamer"].(map[string]interface{})
	if streamer["user_id"] != streamerUser.ID.String() {
		t.Fatalf("user_id mismatch: %v", streamer["user_id"])
	}
	if streamer["agency_user_id"] != agencyUser.ID.String() {
		t.Fatalf("agency_user_id mismatch: %v", streamer["agency_user_id"])
	}
	if streamer["twitch_login"] != "target_login" {
		t.Fatalf("twitch_login mismatch: %v", streamer["twitch_login"])
	}
}

func TestCreate_StreamerForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, streamerToken := createDashboardUser(t, env, models.RoleStreamer, "forbidden_streamer")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_target_s")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "target_login")

	body := `{"user_id":"` + streamerUser.ID.String() + `","twitch_login":"target_login"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", streamerToken, body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_AgencyForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, agencyToken := createDashboardUser(t, env, models.RoleAgency, "forbidden_agency")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_target_a")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "target_login")

	body := `{"user_id":"` + streamerUser.ID.String() + `","twitch_login":"target_login"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", agencyToken, body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestList_AgencySeesOwnOnly(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	agencyA, agencyToken := createDashboardUser(t, env, models.RoleAgency, "agency_a")
	agencyB, _ := createDashboardUser(t, env, models.RoleAgency, "agency_b")
	streamerA1, _ := createDashboardUser(t, env, models.RoleStreamer, "agency_a_streamer1")
	streamerA2, _ := createDashboardUser(t, env, models.RoleStreamer, "agency_a_streamer2")
	streamerB1, _ := createDashboardUser(t, env, models.RoleStreamer, "agency_b_streamer1")

	seedStreamerRow(t, env, streamerA1.ID, &agencyA.ID, "a1_login")
	seedStreamerRow(t, env, streamerA2.ID, &agencyA.ID, "a2_login")
	seedStreamerRow(t, env, streamerB1.ID, &agencyB.ID, "b1_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers", agencyToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != 2 {
		t.Fatalf("want 2 streamers, got %d", len(streamers))
	}
	for _, item := range streamers {
		streamer := item.(map[string]interface{})
		if streamer["agency_user_id"] != agencyA.ID.String() {
			t.Fatalf("unexpected agency_user_id: %v", streamer["agency_user_id"])
		}
	}
}

func TestList_AdminSeesAll(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "list_admin")
	agencyA, _ := createDashboardUser(t, env, models.RoleAgency, "list_agency_a")
	agencyB, _ := createDashboardUser(t, env, models.RoleAgency, "list_agency_b")
	streamerA, _ := createDashboardUser(t, env, models.RoleStreamer, "list_streamer_a")
	streamerB, _ := createDashboardUser(t, env, models.RoleStreamer, "list_streamer_b")

	seedStreamerRow(t, env, streamerA.ID, &agencyA.ID, "admin_a_login")
	seedStreamerRow(t, env, streamerB.ID, &agencyB.ID, "admin_b_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers", adminToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != 2 {
		t.Fatalf("want 2 streamers, got %d", len(streamers))
	}
}

func TestGetStats_StreamerOwnOK(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	streamerUser, streamerToken := createDashboardUser(t, env, models.RoleStreamer, "stats_streamer_self")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "self_login")
	streamer := seedStreamerRow(t, env, streamerUser.ID, nil, "self_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", streamerToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStats_StreamerOtherForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	otherStreamer, _ := createDashboardUser(t, env, models.RoleStreamer, "stats_streamer_other")
	seedTwitchProviderForUser(t, env, otherStreamer.ID, "other_login")
	streamer := seedStreamerRow(t, env, otherStreamer.ID, nil, "other_login")
	_, streamerToken := createDashboardUser(t, env, models.RoleStreamer, "stats_streamer_requester")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", streamerToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStats_AgencyOwnOK(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	agency, agencyToken := createDashboardUser(t, env, models.RoleAgency, "stats_agency_self")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "stats_agency_streamer")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "agency_login")
	streamer := seedStreamerRow(t, env, streamerUser.ID, &agency.ID, "agency_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", agencyToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStats_AgencyOtherForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	otherAgency, _ := createDashboardUser(t, env, models.RoleAgency, "stats_agency_other")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "stats_agency_other_streamer")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "other_agency_login")
	streamer := seedStreamerRow(t, env, streamerUser.ID, &otherAgency.ID, "other_agency_login")
	_, agencyToken := createDashboardUser(t, env, models.RoleAgency, "stats_agency_requester")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", agencyToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStats_AdminAllOK(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "stats_admin")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "stats_admin_streamer")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "admin_login")
	streamer := seedStreamerRow(t, env, streamerUser.ID, nil, "admin_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", adminToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStats_NotFound(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "stats_not_found_admin")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+uuid.New().String()+"/stats", adminToken, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}
