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

	return env, r
}

func makeAccessToken(t *testing.T, role models.UserRole) string {
	t.Helper()

	subject := uuid.NewString()
	claims := services.Claims{
		UserID: uuid.NewString(),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   subject,
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(agencyTestAccessSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
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
