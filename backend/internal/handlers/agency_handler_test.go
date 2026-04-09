package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type failingMailer struct{}

func (m *failingMailer) Send(to, subject, body string) error {
	return errors.New("smtp: connection refused")
}

const agencyTestAccessSecret = "test-access-secret-at-least-32-chars!"

func newAgencyTestEnv(t *testing.T) (*testEnv, http.Handler) {
	t.Helper()

	env := newTestEnv(t)
	agencySvc := services.NewAgencyService(env.db)
	agencyH := handlers.NewAgencyHandler(agencySvc, env.emailAuthSvc)

	r := env.router
	v1 := r.Group("/api/v1")
	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(env.authSvc))
	agencies.POST("", middleware.RequireRole(models.RoleAdmin), agencyH.Create)
	agencies.PUT("/:id/settings",
		middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
		agencyH.UpdateSettings,
	)
	agencies.GET("/:id/streamers",
		middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
		agencyH.ListStreamers,
	)

	return env, r
}

func makeAccessToken(t *testing.T, role models.UserRole) string {
	t.Helper()

	claims := services.Claims{
		UserID: uuid.NewString(),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   uuid.NewString(),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(agencyTestAccessSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
}

func makeAccessTokenForUser(t *testing.T, userID uuid.UUID, role models.UserRole) string {
	t.Helper()

	claims := services.Claims{
		UserID: userID.String(),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(agencyTestAccessSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
}

func seedAgencyStreamerListData(t *testing.T, db *gorm.DB, agencyID uuid.UUID) []map[string]string {
	t.Helper()

	rows := []struct {
		userID    uuid.UUID
		channelID string
		email     string
		username  string
	}{
		{userID: uuid.New(), channelID: "ch_alpha", email: "streamer-alpha@example.com", username: "streamer-alpha"},
		{userID: uuid.New(), channelID: "ch_beta", email: "streamer-beta@example.com", username: "streamer-beta"},
	}

	for _, row := range rows {
		if err := db.Exec(
			`INSERT INTO users (id, username, email, role, is_active, email_verified, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			row.userID, row.username, row.email, models.RoleStreamer,
		).Error; err != nil {
			t.Fatalf("seed streamer user: %v", err)
		}

		if err := db.Exec(
			`INSERT INTO streamers (id, user_id, channel_id, display_name, created_at, updated_at)
			 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			uuid.New(), row.userID, row.channelID, row.username,
		).Error; err != nil {
			t.Fatalf("seed streamer row: %v", err)
		}

		if err := db.Exec(
			`INSERT INTO agency_streamers (id, agency_id, channel_id, created_at)
			 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			uuid.New(), agencyID, row.channelID,
		).Error; err != nil {
			t.Fatalf("seed agency streamer row: %v", err)
		}
	}

	return []map[string]string{
		{"channel_id": rows[0].channelID, "user_id": rows[0].userID.String()},
		{"channel_id": rows[1].channelID, "user_id": rows[1].userID.String()},
	}
}

func TestAgencyHandler_Create_Success(t *testing.T) {
	env, r := newAgencyTestEnv(t)

	body, _ := json.Marshal(map[string]string{
		"name":  "agency-one",
		"email": "agency-one@example.com",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	if data["id"] == nil || data["id"] == "" {
		t.Fatalf("expected non-empty id, got %v", data["id"])
	}
	if data["name"] != "agency-one" {
		t.Fatalf("expected name agency-one, got %v", data["name"])
	}
	if data["email"] != "agency-one@example.com" {
		t.Fatalf("expected email agency-one@example.com, got %v", data["email"])
	}

	// email_verified must be true for admin-created accounts.
	var emailVerified bool
	if err := env.db.Table("users").
		Where("email = ?", "agency-one@example.com").
		Select("email_verified").
		Scan(&emailVerified).Error; err != nil {
		t.Fatalf("query email_verified: %v", err)
	}
	if !emailVerified {
		t.Fatal("expected email_verified = true for admin-created agency")
	}
}

func TestAgencyHandler_Create_TriggersPasswordReset(t *testing.T) {
	env, r := newAgencyTestEnv(t)

	body, _ := json.Marshal(map[string]string{
		"name":  "agency-onboard",
		"email": "agency-onboard@example.com",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	if err := env.db.Table("password_resets").
		Where("email = ?", "agency-onboard@example.com").
		Count(&count).Error; err != nil {
		t.Fatalf("query password_resets: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 password_resets row, got %d", count)
	}
}

func TestAgencyHandler_Create_MailerFailureStillReturns201(t *testing.T) {
	env := newTestEnv(t)

	// Wire up an EmailAuthService backed by a mailer that always fails.
	testCfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
	}
	failSvc := services.NewEmailAuthService(env.db, testCfg, &failingMailer{})
	agencySvc := services.NewAgencyService(env.db)
	agencyH := handlers.NewAgencyHandler(agencySvc, failSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(env.authSvc))
	agencies.POST("", middleware.RequireRole(models.RoleAdmin), agencyH.Create)

	body, _ := json.Marshal(map[string]string{
		"name":  "agency-mailfail",
		"email": "agency-mailfail@example.com",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 even when mailer fails, got %d: %s", w.Code, w.Body.String())
	}

	// Agency user must exist.
	var userCount int64
	if err := env.db.Table("users").
		Where("email = ? AND role = ?", "agency-mailfail@example.com", models.RoleAgency).
		Count(&userCount).Error; err != nil {
		t.Fatalf("query users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected 1 agency user, got %d", userCount)
	}

	// password_resets token must still be written (token is persisted before mailer.Send).
	var resetCount int64
	if err := env.db.Table("password_resets").
		Where("email = ?", "agency-mailfail@example.com").
		Count(&resetCount).Error; err != nil {
		t.Fatalf("query password_resets: %v", err)
	}
	if resetCount != 1 {
		t.Fatalf("expected 1 password_resets row even on mailer failure, got %d", resetCount)
	}
}

// TestAgencyHandler_Create_PartialSuccess_TokenWriteFailure verifies that
// POST /agencies returns 201 and the agency user is created even when the
// password_resets write fails (partial success: agency committed, setup incomplete).
// Admin can re-trigger password setup via POST /auth/forgot-password.
func TestAgencyHandler_Create_PartialSuccess_TokenWriteFailure(t *testing.T) {
	env := newTestEnv(t)

	agencySvc := services.NewAgencyService(env.db)
	agencyH := handlers.NewAgencyHandler(agencySvc, env.emailAuthSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(env.authSvc))
	agencies.POST("", middleware.RequireRole(models.RoleAdmin), agencyH.Create)

	// Drop password_resets table so ForgotPassword fails at DB write (before mailer.Send).
	if err := env.db.Exec("DROP TABLE IF EXISTS password_resets").Error; err != nil {
		t.Fatalf("drop password_resets: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"name":  "agency-dbfail",
		"email": "agency-dbfail@example.com",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Partial success: agency is created, password setup failed, but caller gets 201
	// so they don't retry and hit duplicate email/name errors.
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (partial success), got %d: %s", w.Code, w.Body.String())
	}

	// Agency user must exist — the core operation succeeded.
	var userCount int64
	if err := env.db.Table("users").
		Where("email = ? AND role = ?", "agency-dbfail@example.com", models.RoleAgency).
		Count(&userCount).Error; err != nil {
		t.Fatalf("query users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected agency user to exist after partial success, got %d", userCount)
	}
}

func TestAgencyHandler_Create_DuplicateEmail(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	firstBody, _ := json.Marshal(map[string]string{
		"name":  "agency-dup",
		"email": "agency-dup@example.com",
	})
	secondBody, _ := json.Marshal(map[string]string{
		"name":  "agency-dup-2",
		"email": "agency-dup@example.com",
	})

	first := httptest.NewRecorder()
	firstReq, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(firstBody))
	firstReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	firstReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(first, firstReq)

	if first.Code != http.StatusCreated {
		t.Fatalf("first create expected 201, got %d: %s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	secondReq, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(secondBody))
	secondReq.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	secondReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(second, secondReq)

	if second.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", second.Code, second.Body.String())
	}
}

func TestAgencyHandler_Create_InvalidBody(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	body, _ := json.Marshal(map[string]string{
		"name": "agency-invalid",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_Create_RequiresAdmin(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	body, _ := json.Marshal(map[string]string{
		"name":  "agency-viewer",
		"email": "agency-viewer@example.com",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/agencies", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleViewer))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func seedAgencyUser(t *testing.T, db *gorm.DB, name, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(
		`INSERT INTO users (id, username, email, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id, name, email, models.RoleAgency,
	).Error; err != nil {
		t.Fatalf("seed agency user: %v", err)
	}
	return id
}

func TestAgencyHandler_UpdateSettings_AdminSuccess(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := seedAgencyUser(t, env.db, "agency-orig", "agency-orig@example.com")

	body, _ := json.Marshal(map[string]string{"name": "agency-renamed"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/"+agencyID.String()+"/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	if resp["data"].(map[string]interface{})["name"] != "agency-renamed" {
		t.Fatalf("expected name agency-renamed in response, got %v", resp["data"])
	}
}

func TestAgencyHandler_UpdateSettings_AgencyCanUpdateOwn(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := seedAgencyUser(t, env.db, "agency-self", "agency-self@example.com")

	body, _ := json.Marshal(map[string]string{"name": "agency-self-new"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/"+agencyID.String()+"/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, agencyID, models.RoleAgency))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_UpdateSettings_AgencyCannotUpdateOthers(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := seedAgencyUser(t, env.db, "agency-other", "agency-other@example.com")

	body, _ := json.Marshal(map[string]string{"name": "hacked"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/"+agencyID.String()+"/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, uuid.New(), models.RoleAgency))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_UpdateSettings_NotFound(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	body, _ := json.Marshal(map[string]string{"name": "ghost"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/"+uuid.NewString()+"/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_UpdateSettings_NameTaken(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	seedAgencyUser(t, env.db, "taken-name", "taken@example.com")
	agencyID := seedAgencyUser(t, env.db, "agency-b", "agency-b@example.com")

	body, _ := json.Marshal(map[string]string{"name": "taken-name"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/"+agencyID.String()+"/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_UpdateSettings_NameTooLong(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := seedAgencyUser(t, env.db, "agency-toolong", "agency-toolong@example.com")

	body, _ := json.Marshal(map[string]string{"name": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}) // 51 chars
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/"+agencyID.String()+"/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_UpdateSettings_InvalidID(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	body, _ := json.Marshal(map[string]string{"name": "x"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agencies/not-a-uuid/settings", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_ListStreamers_AdminCanQuery(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := uuid.New()
	expected := seedAgencyStreamerListData(t, env.db, agencyID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != len(expected) {
		t.Fatalf("expected %d streamers, got %d", len(expected), len(streamers))
	}

	for i, item := range streamers {
		streamer := item.(map[string]interface{})
		if streamer["channel_id"] != expected[i]["channel_id"] {
			t.Fatalf("streamer %d channel_id: expected %s, got %v", i, expected[i]["channel_id"], streamer["channel_id"])
		}
		if streamer["user_id"] != expected[i]["user_id"] {
			t.Fatalf("streamer %d user_id: expected %s, got %v", i, expected[i]["user_id"], streamer["user_id"])
		}
	}
}

func TestAgencyHandler_ListStreamers_AgencyCanQueryOwn(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := uuid.New()
	expected := seedAgencyStreamerListData(t, env.db, agencyID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, agencyID, models.RoleAgency))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != len(expected) {
		t.Fatalf("expected %d streamers, got %d", len(expected), len(streamers))
	}
}

func TestAgencyHandler_ListStreamers_AgencyCannotQueryOthers(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+uuid.NewString()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessTokenForUser(t, uuid.New(), models.RoleAgency))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_ListStreamers_InvalidID(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/not-a-uuid/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_ListStreamers_OrphanChannelReturns500(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := uuid.New()

	// Insert agency_streamers row for a channel that has NO matching streamers row.
	if err := env.db.Exec(
		`INSERT INTO agency_streamers (id, agency_id, channel_id, created_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		uuid.New(), agencyID, "ch_orphan",
	).Error; err != nil {
		t.Fatalf("seed orphan agency streamer: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for orphan channel, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_ListStreamers_EmptyAgency(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := uuid.New()

	// Seed a real agency user so the agency exists, but with no streamers.
	if err := env.db.Exec(
		`INSERT INTO users (id, username, email, role, is_active, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		agencyID, "agency-empty", "agency-empty@example.com", models.RoleAgency,
	).Error; err != nil {
		t.Fatalf("seed agency user: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	streamers := data["streamers"].([]interface{})
	if len(streamers) != 0 {
		t.Fatalf("expected empty streamers, got %d", len(streamers))
	}
}

func TestAgencyHandler_ListStreamers_NotFound(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+uuid.NewString()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_ListStreamers_DuplicateChannelReturns500(t *testing.T) {
	env, r := newAgencyTestEnv(t)
	agencyID := uuid.New()

	// Insert two different streamer users that both claim the same channel_id.
	// streamers has UNIQUE(user_id, channel_id) but NOT UNIQUE(channel_id),
	// so this is valid at the DB level and must be caught at the service layer.
	for i, email := range []string{"dup-a@example.com", "dup-b@example.com"} {
		userID := uuid.New()
		username := email
		if err := env.db.Exec(
			`INSERT INTO users (id, username, email, role, is_active, email_verified, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			userID, username, email, models.RoleStreamer,
		).Error; err != nil {
			t.Fatalf("seed streamer user %d: %v", i, err)
		}
		if err := env.db.Exec(
			`INSERT INTO streamers (id, user_id, channel_id, display_name, created_at, updated_at)
			 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			uuid.New(), userID, "ch_dup", username,
		).Error; err != nil {
			t.Fatalf("seed streamer row %d: %v", i, err)
		}
	}
	if err := env.db.Exec(
		`INSERT INTO agency_streamers (id, agency_id, channel_id, created_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		uuid.New(), agencyID, "ch_dup",
	).Error; err != nil {
		t.Fatalf("seed agency streamer: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+agencyID.String()+"/streamers", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, models.RoleAdmin))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for duplicate channel_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgencyHandler_ListStreamers_RequiresAuth(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+uuid.NewString()+"/streamers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
