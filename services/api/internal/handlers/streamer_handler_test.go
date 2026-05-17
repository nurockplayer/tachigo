package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type streamerHandlerContextKey struct{}

func installStreamerHandlerDBContextProbe(t *testing.T, db *gorm.DB, key, want any) func() int {
	t.Helper()

	var seen int
	name := "test:streamer_handler_db_context:" + uuid.NewString()
	probe := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Context != nil && tx.Statement.Context.Value(key) == want {
			seen++
		}
	}

	if err := db.Callback().Query().Before("gorm:query").Register(name+":query", probe); err != nil {
		t.Fatalf("register query context probe: %v", err)
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register(name+":raw", probe); err != nil {
		t.Fatalf("register raw context probe: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(name + ":query")
		_ = db.Callback().Raw().Remove(name + ":raw")
	})

	return func() int {
		return seen
	}
}

func newStreamerDashboardEnv(t *testing.T) *dashboardEnv {
	t.Helper()

	base := newTestEnv(t)
	watchSvc := services.NewWatchService(base.db)
	pointsSvc := services.NewPointsService(base.db, watchSvc)
	streamerSvc := services.NewStreamerService(base.db, pointsSvc)
	streamerH := handlers.NewStreamerHandler(streamerSvc)
	configSvc := services.NewChannelConfigService(base.db)
	configH := handlers.NewChannelConfigHandler(configSvc, streamerSvc)

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
		dashboard.POST("/streamers/register",
			middleware.RequireRole(models.RoleStreamer),
			streamerH.Register)
		dashboard.GET("/streamers/channels",
			middleware.RequireRole(models.RoleStreamer),
			streamerH.ListChannels)
		dashboard.GET("/channels/:channel_id/stats",
			middleware.RequireRole(models.RoleAdmin, models.RoleStreamer),
			streamerH.GetChannelStats)
		dashboard.GET("/channels/:channel_id/config",
			middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency),
			configH.GetChannelConfig)
		dashboard.PUT("/channels/:channel_id/config",
			middleware.RequireRole(models.RoleAdmin, models.RoleStreamer),
			configH.UpdateChannelConfig)
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

func dashboardRequestWithContext(
	t *testing.T,
	router *gin.Engine,
	ctx context.Context,
	method,
	path,
	token,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	if ctx == nil {
		ctx = context.Background()
	}
	req := httptest.NewRequestWithContext(ctx, method, path, bytes.NewBufferString(body))
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

func seedTwitchProvider(t *testing.T, env *dashboardEnv, email, channelID string) {
	t.Helper()
	var user models.User
	if err := env.db.Where("email = ?", email).First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if err := env.db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), user.ID, models.ProviderTwitch, channelID,
	).Error; err != nil {
		t.Fatalf("seed auth provider: %v", err)
	}
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

func seedTwitchProviderForUser(t *testing.T, env *dashboardEnv, userID uuid.UUID, channelID string) {
	t.Helper()
	if err := env.db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		uuid.New(), userID, models.ProviderTwitch, channelID,
	).Error; err != nil {
		t.Fatalf("seed auth provider: %v", err)
	}
}

func seedStreamerRow(t *testing.T, env *dashboardEnv, userID uuid.UUID, agencyUserID *uuid.UUID, channelID string) *models.Streamer {
	t.Helper()
	streamer := &models.Streamer{
		UserID:       userID,
		AgencyUserID: agencyUserID,
		ChannelID:    channelID,
	}
	if err := env.db.Create(streamer).Error; err != nil {
		t.Fatalf("seed streamer: %v", err)
	}
	return streamer
}

func TestStreamerRegister_AndListChannels(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)
	seedTwitchProvider(t, env, "streamer_dashboard@example.com", "ch_123")

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

func TestRegister_RejectsUnownedChannel(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	token := env.tokenForRole(t, models.RoleStreamer)
	// No auth_provider seeded — channel does not belong to this user.

	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers/register", token, `{"channel_id":"ch_someone_else","display_name":""}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
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
	// Register ownership so GetChannelStats ownership check passes.
	if err := env.db.Create(&models.Streamer{
		UserID:    streamer.ID,
		ChannelID: "ch_123",
	}).Error; err != nil {
		t.Fatalf("seed streamer: %v", err)
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

func TestGetChannelStats_ForbiddenForOtherStreamer(t *testing.T) {
	env := newStreamerDashboardEnv(t)

	// Streamer A registers ch_A.
	tokenA := env.tokenForRole(t, models.RoleStreamer)
	seedTwitchProvider(t, env, "streamer_dashboard@example.com", "ch_A")
	dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers/register", tokenA, `{"channel_id":"ch_A"}`)

	// Streamer B — registered with a different email.
	_, _ = env.registerUser(t, "streamer_b", "streamer_b@example.com", "password123")
	if err := env.db.Exec(`UPDATE users SET role = 'streamer' WHERE email = 'streamer_b@example.com'`).Error; err != nil {
		t.Fatalf("set role: %v", err)
	}
	// Re-login to get a token with updated role.
	_, tokens, err := env.authSvc.Login(services.LoginInput{Email: "streamer_b@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login streamer_b: %v", err)
	}
	tokenB := tokens.AccessToken

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/channels/ch_A/stats", tokenB, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_AdminOK(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "streamer_admin")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_target")
	agencyUser, _ := createDashboardUser(t, env, models.RoleAgency, "agency_target")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "target_ch_id")

	body := `{"user_id":"` + streamerUser.ID.String() + `","agency_user_id":"` + agencyUser.ID.String() + `","channel_id":"target_ch_id"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", adminToken, body)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var created models.Streamer
	if err := env.db.Where("user_id = ? AND channel_id = ?", streamerUser.ID, "target_ch_id").First(&created).Error; err != nil {
		t.Fatalf("load created streamer: %v", err)
	}
	if created.UserID != streamerUser.ID {
		t.Fatalf("created streamer user_id: want %s, got %s", streamerUser.ID, created.UserID)
	}
	if created.AgencyUserID == nil {
		t.Fatal("created streamer agency_user_id: want non-nil")
	}
	if *created.AgencyUserID != agencyUser.ID {
		t.Fatalf("created streamer agency_user_id: want %s, got %s", agencyUser.ID, *created.AgencyUserID)
	}
	if created.ChannelID != "target_ch_id" {
		t.Fatalf("created streamer channel_id: want target_ch_id, got %s", created.ChannelID)
	}
}

func TestCreate_StreamerForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, streamerToken := createDashboardUser(t, env, models.RoleStreamer, "forbidden_streamer")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_target_s")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "target_ch_id")

	body := `{"user_id":"` + streamerUser.ID.String() + `","channel_id":"target_ch_id"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", streamerToken, body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_AgencyForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, agencyToken := createDashboardUser(t, env, models.RoleAgency, "forbidden_agency")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_target_a")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "target_ch_id")

	body := `{"user_id":"` + streamerUser.ID.String() + `","channel_id":"target_ch_id"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", agencyToken, body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_RejectsAgencyUserIDForNonAgencyUser(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "admin_non_agency")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_non_agency_target")
	viewerUser, _ := createDashboardUser(t, env, models.RoleViewer, "viewer_not_agency")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "non_agency_target_ch")

	body := `{"user_id":"` + streamerUser.ID.String() + `","agency_user_id":"` + viewerUser.ID.String() + `","channel_id":"non_agency_target_ch"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", adminToken, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_RejectsUnknownAgencyUserID(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "admin_unknown_agency")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "streamer_unknown_agency_target")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "unknown_agency_target_ch")

	unknownAgencyID := uuid.New()
	body := `{"user_id":"` + streamerUser.ID.String() + `","agency_user_id":"` + unknownAgencyID.String() + `","channel_id":"unknown_agency_target_ch"}`
	w := dashboardRequest(t, env.router, http.MethodPost, "/api/v1/dashboard/streamers", adminToken, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestList_PassesRequestContextToStreamerQueries(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "list_context_admin")
	streamerUser, _ := createDashboardUser(t, env, models.RoleStreamer, "list_context_streamer")
	seedStreamerRow(t, env, streamerUser.ID, nil, "list_context_channel")
	key := streamerHandlerContextKey{}
	seen := installStreamerHandlerDBContextProbe(t, env.db, key, "streamer-list")

	w := dashboardRequestWithContext(
		t,
		env.router,
		context.WithValue(context.Background(), key, "streamer-list"),
		http.MethodGet,
		"/api/v1/dashboard/streamers",
		adminToken,
		"",
	)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if seen() == 0 {
		t.Fatal("expected List handler to pass request context to streamer GORM queries")
	}
}

func TestList_AgencySeesOwnOnly(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	agencyA, agencyToken := createDashboardUser(t, env, models.RoleAgency, "agency_a")
	agencyB, _ := createDashboardUser(t, env, models.RoleAgency, "agency_b")
	streamerA1, _ := createDashboardUser(t, env, models.RoleStreamer, "agency_a_streamer1")
	streamerA2, _ := createDashboardUser(t, env, models.RoleStreamer, "agency_a_streamer2")
	streamerB1, _ := createDashboardUser(t, env, models.RoleStreamer, "agency_b_streamer1")

	seededA1 := seedStreamerRow(t, env, streamerA1.ID, &agencyA.ID, "a1_login")
	seededA2 := seedStreamerRow(t, env, streamerA2.ID, &agencyA.ID, "a2_login")
	seededB1 := seedStreamerRow(t, env, streamerB1.ID, &agencyB.ID, "b1_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers", agencyToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != 2 {
		t.Fatalf("agency_a: want 2 streamers, got %d", len(streamers))
	}
	expected := map[string]struct {
		channelID    string
		agencyUserID string
	}{
		seededA1.ID.String(): {channelID: "a1_login", agencyUserID: agencyA.ID.String()},
		seededA2.ID.String(): {channelID: "a2_login", agencyUserID: agencyA.ID.String()},
	}
	seen := make(map[string]bool)
	for _, s := range streamers {
		row := s.(map[string]interface{})
		id := row["id"].(string)
		want, ok := expected[id]
		if !ok {
			t.Fatalf("agency_a: unexpected streamer id=%s (streamer_b1=%s should be excluded)", id, seededB1.ID)
		}
		if seen[id] {
			t.Fatalf("agency_a: duplicate streamer id=%s", id)
		}
		seen[id] = true
		if row["agency_user_id"] != want.agencyUserID {
			t.Fatalf("agency_a: streamer id=%s got agency_user_id=%v, want %s", id, row["agency_user_id"], want.agencyUserID)
		}
		if row["channel_id"] != want.channelID {
			t.Fatalf("agency_a: streamer id=%s got channel_id=%v, want %s", id, row["channel_id"], want.channelID)
		}
	}
	if seen[seededB1.ID.String()] {
		t.Fatalf("agency_a: streamer_b1=%s must be excluded", seededB1.ID)
	}
}

func TestList_AdminSeesAll(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	_, adminToken := createDashboardUser(t, env, models.RoleAdmin, "list_admin")
	agencyA, _ := createDashboardUser(t, env, models.RoleAgency, "list_agency_a")
	agencyB, _ := createDashboardUser(t, env, models.RoleAgency, "list_agency_b")
	streamerA, _ := createDashboardUser(t, env, models.RoleStreamer, "list_streamer_a")
	streamerB, _ := createDashboardUser(t, env, models.RoleStreamer, "list_streamer_b")

	seededA := seedStreamerRow(t, env, streamerA.ID, &agencyA.ID, "admin_a_login")
	seededB := seedStreamerRow(t, env, streamerB.ID, &agencyB.ID, "admin_b_login")

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers", adminToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != 2 {
		t.Fatalf("admin: want 2 streamers, got %d", len(streamers))
	}
	expected := map[string]struct {
		channelID    string
		agencyUserID string
	}{
		seededA.ID.String(): {channelID: "admin_a_login", agencyUserID: agencyA.ID.String()},
		seededB.ID.String(): {channelID: "admin_b_login", agencyUserID: agencyB.ID.String()},
	}
	seen := make(map[string]bool)
	for _, s := range streamers {
		row := s.(map[string]interface{})
		id := row["id"].(string)
		want, ok := expected[id]
		if !ok {
			t.Fatalf("admin: unexpected streamer id=%s", id)
		}
		if seen[id] {
			t.Fatalf("admin: duplicate streamer id=%s", id)
		}
		seen[id] = true
		if row["agency_user_id"] != want.agencyUserID {
			t.Fatalf("admin: streamer id=%s got agency_user_id=%v, want %s", id, row["agency_user_id"], want.agencyUserID)
		}
		if row["channel_id"] != want.channelID {
			t.Fatalf("admin: streamer id=%s got channel_id=%v, want %s", id, row["channel_id"], want.channelID)
		}
	}
	for id := range expected {
		if !seen[id] {
			t.Fatalf("admin: missing streamer id=%s", id)
		}
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

	// Non-admin callers receive 404 for unauthorized streamer_ids to prevent existence enumeration.
	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", streamerToken, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
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

	// Non-admin callers receive 404 for unauthorized streamer_ids to prevent existence enumeration.
	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer.ID.String()+"/stats", agencyToken, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
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

// TestGetStats_MultiChannelSameUser verifies that GetStats returns stats for
// the requested streamer_id even when the same user owns multiple channels.
func TestGetStats_MultiChannelSameUser(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	streamerUser, streamerToken := createDashboardUser(t, env, models.RoleStreamer, "multi_ch_streamer")
	viewer1, _ := createDashboardUser(t, env, models.RoleViewer, "multi_ch_viewer1")
	viewer2, _ := createDashboardUser(t, env, models.RoleViewer, "multi_ch_viewer2")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "ch_multi_1")
	seedTwitchProviderForUser(t, env, streamerUser.ID, "ch_multi_2")

	streamer1 := seedStreamerRow(t, env, streamerUser.ID, nil, "ch_multi_1")
	streamer2 := seedStreamerRow(t, env, streamerUser.ID, nil, "ch_multi_2")

	now := time.Now()
	if err := env.db.Exec(`
		INSERT INTO broadcast_time_logs (id, streamer_id, channel_id, seconds, recorded_at)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)
	`,
		uuid.New(), streamerUser.ID, "ch_multi_1", 11, now,
		uuid.New(), streamerUser.ID, "ch_multi_2", 29, now,
	).Error; err != nil {
		t.Fatalf("seed broadcast logs: %v", err)
	}
	if err := env.db.Exec(`
		INSERT INTO points_ledgers (id, user_id, channel_id, cumulative_total, spendable_balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`,
		uuid.New(), viewer1.ID, "ch_multi_1", 101, 51,
		uuid.New(), viewer2.ID, "ch_multi_2", 202, 102,
	).Error; err != nil {
		t.Fatalf("seed points ledgers: %v", err)
	}
	if err := env.db.Exec(`
		INSERT INTO watch_sessions (
			id, user_id, channel_id, accumulated_seconds, rewarded_seconds,
			last_heartbeat_at, click_cooldown_until, is_active, created_at, updated_at
		) VALUES
			(?, ?, ?, ?, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(?, ?, ?, ?, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`,
		uuid.New(), viewer1.ID, "ch_multi_1", 5,
		uuid.New(), viewer2.ID, "ch_multi_2", 17,
	).Error; err != nil {
		t.Fatalf("seed watch sessions: %v", err)
	}

	w1 := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer1.ID.String()+"/stats", streamerToken, "")
	if w1.Code != http.StatusOK {
		t.Fatalf("streamer1: want 200, got %d: %s", w1.Code, w1.Body.String())
	}
	resp1 := parseBody(t, w1.Body.Bytes())
	data1 := resp1["data"].(map[string]interface{})
	if data1["channel_id"] != "ch_multi_1" {
		t.Fatalf("streamer1: want channel_id=ch_multi_1, got %v", data1["channel_id"])
	}
	stats1 := data1["stats"].(map[string]interface{})
	if stats1["current_session_seconds"] != float64(5) {
		t.Fatalf("streamer1: want current_session_seconds=5, got %v", stats1["current_session_seconds"])
	}
	if stats1["daily_seconds"] != float64(11) {
		t.Fatalf("streamer1: want daily_seconds=11, got %v", stats1["daily_seconds"])
	}
	if stats1["monthly_seconds"] != float64(11) {
		t.Fatalf("streamer1: want monthly_seconds=11, got %v", stats1["monthly_seconds"])
	}
	if stats1["yearly_seconds"] != float64(11) {
		t.Fatalf("streamer1: want yearly_seconds=11, got %v", stats1["yearly_seconds"])
	}
	if stats1["avg_session_seconds"] != float64(5) {
		t.Fatalf("streamer1: want avg_session_seconds=5, got %v", stats1["avg_session_seconds"])
	}
	if stats1["total_token_minted"] != float64(101) {
		t.Fatalf("streamer1: want total_token_minted=101, got %v", stats1["total_token_minted"])
	}
	if stats1["spendable_in_circulation"] != float64(51) {
		t.Fatalf("streamer1: want spendable_in_circulation=51, got %v", stats1["spendable_in_circulation"])
	}
	if stats1["unique_miners"] != float64(1) {
		t.Fatalf("streamer1: want unique_miners=1, got %v", stats1["unique_miners"])
	}

	w2 := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/streamers/"+streamer2.ID.String()+"/stats", streamerToken, "")
	if w2.Code != http.StatusOK {
		t.Fatalf("streamer2: want 200, got %d: %s", w2.Code, w2.Body.String())
	}
	resp2 := parseBody(t, w2.Body.Bytes())
	data2 := resp2["data"].(map[string]interface{})
	if data2["channel_id"] != "ch_multi_2" {
		t.Fatalf("streamer2: want channel_id=ch_multi_2, got %v", data2["channel_id"])
	}
	stats2 := data2["stats"].(map[string]interface{})
	if stats2["current_session_seconds"] != float64(17) {
		t.Fatalf("streamer2: want current_session_seconds=17, got %v", stats2["current_session_seconds"])
	}
	if stats2["daily_seconds"] != float64(29) {
		t.Fatalf("streamer2: want daily_seconds=29, got %v", stats2["daily_seconds"])
	}
	if stats2["monthly_seconds"] != float64(29) {
		t.Fatalf("streamer2: want monthly_seconds=29, got %v", stats2["monthly_seconds"])
	}
	if stats2["yearly_seconds"] != float64(29) {
		t.Fatalf("streamer2: want yearly_seconds=29, got %v", stats2["yearly_seconds"])
	}
	if stats2["avg_session_seconds"] != float64(17) {
		t.Fatalf("streamer2: want avg_session_seconds=17, got %v", stats2["avg_session_seconds"])
	}
	if stats2["total_token_minted"] != float64(202) {
		t.Fatalf("streamer2: want total_token_minted=202, got %v", stats2["total_token_minted"])
	}
	if stats2["spendable_in_circulation"] != float64(102) {
		t.Fatalf("streamer2: want spendable_in_circulation=102, got %v", stats2["spendable_in_circulation"])
	}
	if stats2["unique_miners"] != float64(1) {
		t.Fatalf("streamer2: want unique_miners=1, got %v", stats2["unique_miners"])
	}
}

// TestLegacyChannelStats_AgencyForbidden verifies that the legacy
// GET /dashboard/channels/:channel_id/stats is not accessible to agency role.
func TestLegacyChannelStats_AgencyForbidden(t *testing.T) {
	env := newStreamerDashboardEnv(t)
	agencyUser, agencyToken := createDashboardUser(t, env, models.RoleAgency, "legacy_stats_agency")
	seedTwitchProviderForUser(t, env, agencyUser.ID, "agency_owned_channel")
	seedStreamerRow(t, env, agencyUser.ID, nil, "agency_owned_channel")
	if err := env.db.Create(&models.BroadcastTimeLog{
		StreamerID: agencyUser.ID,
		ChannelID:  "agency_owned_channel",
		Seconds:    15,
		RecordedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	w := dashboardRequest(t, env.router, http.MethodGet, "/api/v1/dashboard/channels/agency_owned_channel/stats", agencyToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("agency on legacy stats: want 403, got %d: %s", w.Code, w.Body.String())
	}
}
