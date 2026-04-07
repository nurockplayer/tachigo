package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

const agencyTestAccessSecret = "test-access-secret-at-least-32-chars!"

func newAgencyTestEnv(t *testing.T) (*testEnv, http.Handler) {
	t.Helper()

	env := newTestEnv(t)
	agencySvc := services.NewAgencyService(env.db)
	agencyH := handlers.NewAgencyHandler(agencySvc)

	r := env.router
	v1 := r.Group("/api/v1")
	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(env.authSvc))
	agencies.POST("", middleware.RequireRole(models.RoleAdmin), agencyH.Create)
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
	_, r := newAgencyTestEnv(t)

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

func TestAgencyHandler_ListStreamers_RequiresAuth(t *testing.T) {
	_, r := newAgencyTestEnv(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agencies/"+uuid.NewString()+"/streamers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
