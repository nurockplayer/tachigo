package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

func newClaimTestEnv(t *testing.T) (*testEnv, *gin.Engine) {
	t.Helper()
	env := newTestEnv(t)
	claimSvc := services.NewClaimService(env.db)
	claimH := handlers.NewClaimHandler(claimSvc)

	r := env.router
	v1 := r.Group("/api/v1")
	protected := v1.Group("/")
	protected.Use(middleware.JWTAuth(env.authSvc))
	protected.POST("users/me/points/claim", claimH.Claim)
	protected.GET("users/me/tachi/balance", claimH.GetTachiBalance)

	return env, r
}

func seedLedgerForHandler(t *testing.T, env *testEnv, userID uuid.UUID, channelID string, spendable int64) {
	t.Helper()
	id := uuid.New()
	if err := env.db.Exec(`
		INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, userID, channelID, spendable, spendable).Error; err != nil {
		t.Fatalf("seedLedger: %v", err)
	}
}

func TestClaimHandler_GetTachiBalance_Empty(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user1", "user1@example.com", "password123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/users/me/tachi/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w.Body.Bytes())
	data := body["data"].(map[string]interface{})
	if data["tachi_balance"].(float64) != 0 {
		t.Fatalf("expected tachi_balance=0, got %v", data["tachi_balance"])
	}
}

func TestClaimHandler_ClaimAll(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user2", "user2@example.com", "password123")

	// Resolve userID from DB
	var userIDStr string
	env.db.Raw("SELECT id FROM users WHERE email = 'user2@example.com'").Scan(&userIDStr)
	userID, _ := uuid.Parse(userIDStr)
	seedLedgerForHandler(t, env, userID, "ch1", 200)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w.Body.Bytes())
	data := body["data"].(map[string]interface{})
	if data["tachi_balance"].(float64) != 200 {
		t.Fatalf("expected tachi_balance=200, got %v", data["tachi_balance"])
	}
}

func TestClaimHandler_ClaimPartial(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user3", "user3@example.com", "password123")

	var userIDStr string
	env.db.Raw("SELECT id FROM users WHERE email = 'user3@example.com'").Scan(&userIDStr)
	userID, _ := uuid.Parse(userIDStr)
	seedLedgerForHandler(t, env, userID, "ch1", 100)

	body, _ := json.Marshal(map[string]int{"amount": 60})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	if data["tachi_balance"].(float64) != 60 {
		t.Fatalf("expected tachi_balance=60, got %v", data["tachi_balance"])
	}
}

func TestClaimHandler_InsufficientBalance(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user4", "user4@example.com", "password123")

	var userIDStr string
	env.db.Raw("SELECT id FROM users WHERE email = 'user4@example.com'").Scan(&userIDStr)
	userID, _ := uuid.Parse(userIDStr)
	seedLedgerForHandler(t, env, userID, "ch1", 10)

	body, _ := json.Marshal(map[string]int{"amount": 999})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 422 {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClaimHandler_Unauthorized(t *testing.T) {
	_, r := newClaimTestEnv(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestClaimHandler_GetBalanceAfterClaim(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user5", "user5@example.com", "password123")

	var userIDStr string
	env.db.Raw("SELECT id FROM users WHERE email = 'user5@example.com'").Scan(&userIDStr)
	userID, _ := uuid.Parse(userIDStr)
	seedLedgerForHandler(t, env, userID, "ch1", 300)

	// Claim 100
	b, _ := json.Marshal(map[string]int{"amount": 100})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Get balance
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/users/me/tachi/balance", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	resp := parseBody(t, w2.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	if data["tachi_balance"].(float64) != 100 {
		t.Fatalf("expected tachi_balance=100, got %v", data["tachi_balance"])
	}
}

// Ensure the unused fmt import doesn't cause build failure in test helpers
var _ = fmt.Sprintf
