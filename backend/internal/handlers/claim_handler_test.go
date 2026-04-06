package handlers_test

import (
	"bytes"
	"encoding/json"
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

// resolveUserID resolves a userID from email, failing the test if not found or unparseable.
func resolveUserID(t *testing.T, env *testEnv, email string) uuid.UUID {
	t.Helper()
	var userIDStr string
	if err := env.db.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&userIDStr).Error; err != nil {
		t.Fatalf("resolveUserID: scan error: %v", err)
	}
	if userIDStr == "" {
		t.Fatalf("resolveUserID: no user found for email %s", email)
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		t.Fatalf("resolveUserID: parse error: %v", err)
	}
	return userID
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

	userID := resolveUserID(t, env, "user2@example.com")
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

	userID := resolveUserID(t, env, "user3@example.com")
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

	userID := resolveUserID(t, env, "user4@example.com")
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

func TestClaimHandler_MalformedJSON(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user6", "user6@example.com", "password123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClaimHandler_GetBalanceAfterClaim(t *testing.T) {
	env, r := newClaimTestEnv(t)
	token, _ := env.registerUser(t, "user5", "user5@example.com", "password123")

	userID := resolveUserID(t, env, "user5@example.com")
	seedLedgerForHandler(t, env, userID, "ch1", 300)

	// Claim 100 and assert success first
	b, _ := json.Marshal(map[string]int{"amount": 100})
	wClaim := httptest.NewRecorder()
	reqClaim, _ := http.NewRequest("POST", "/api/v1/users/me/points/claim", bytes.NewBuffer(b))
	reqClaim.Header.Set("Authorization", "Bearer "+token)
	reqClaim.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wClaim, reqClaim)

	if wClaim.Code != http.StatusOK {
		t.Fatalf("claim expected 200, got %d: %s", wClaim.Code, wClaim.Body.String())
	}

	// Get balance and verify
	wBal := httptest.NewRecorder()
	reqBal, _ := http.NewRequest("GET", "/api/v1/users/me/tachi/balance", nil)
	reqBal.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(wBal, reqBal)

	if wBal.Code != http.StatusOK {
		t.Fatalf("balance expected 200, got %d: %s", wBal.Code, wBal.Body.String())
	}
	resp := parseBody(t, wBal.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	if data["tachi_balance"].(float64) != 100 {
		t.Fatalf("expected tachi_balance=100, got %v", data["tachi_balance"])
	}
}
