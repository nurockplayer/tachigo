package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type pointsEnv struct {
	*testEnv
	pointsSvc *services.PointsService
}

func newPointsEnv(t *testing.T) *pointsEnv {
	t.Helper()
	base := newTestEnv(t)
	watchSvc := services.NewWatchService(base.db)
	pointsSvc := services.NewPointsService(base.db, watchSvc)
	pointsH := handlers.NewPointsHandler(pointsSvc)

	protected := base.router.Group("/api/v1")
	protected.Use(middleware.JWTAuth(base.authSvc))
	protected.GET("/users/me/points", pointsH.GetBalance)
	protected.GET("/users/me/points/history", pointsH.GetHistory)

	return &pointsEnv{testEnv: base, pointsSvc: pointsSvc}
}

func (e *pointsEnv) registerViewer(t *testing.T, suffix string) (uuid.UUID, string) {
	t.Helper()
	user, tokens, err := e.authSvc.Register(services.RegisterInput{
		Username: "viewer_" + suffix,
		Email:    "viewer_" + suffix + "@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("registerViewer: %v", err)
	}
	return user.ID, tokens.AccessToken
}

func TestPointsHandler_GetBalance_ReturnsWrappedBalances(t *testing.T) {
	e := newPointsEnv(t)
	userID, token := e.registerViewer(t, "balance")

	if err := e.pointsSvc.AddPoints(userID, "ch_abc", models.TxSourceBits, 100); err != nil {
		t.Fatalf("seed points: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/points?channel_id=ch_abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseBody(t, rec.Body.Bytes())
	if resp["success"] != true {
		t.Fatalf("success: want true, got %v", resp["success"])
	}
	data := resp["data"].(map[string]interface{})
	if data["spendable_balance"] != float64(100) {
		t.Fatalf("spendable_balance: want 100, got %v", data["spendable_balance"])
	}
	if data["cumulative_total"] != float64(100) {
		t.Fatalf("cumulative_total: want 100, got %v", data["cumulative_total"])
	}
}

func TestPointsHandler_GetHistory_ReturnsMappedTransactions(t *testing.T) {
	e := newPointsEnv(t)
	userID, token := e.registerViewer(t, "history")

	if err := e.pointsSvc.AddPoints(userID, "ch_abc", models.TxSourceBits, 100); err != nil {
		t.Fatalf("seed earn: %v", err)
	}
	if err := e.pointsSvc.DeductPoints(userID, "ch_abc", 30, "avatar"); err != nil {
		t.Fatalf("seed spend: %v", err)
	}

	var ledger models.PointsLedger
	if err := e.db.Where("user_id = ? AND channel_id = ?", userID, "ch_abc").First(&ledger).Error; err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	base := time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC)
	if err := e.db.Model(&models.PointsTransaction{}).
		Where("ledger_id = ? AND source = ?", ledger.ID, models.TxSourceBits).
		Update("created_at", base).Error; err != nil {
		t.Fatalf("update earn timestamp: %v", err)
	}
	if err := e.db.Model(&models.PointsTransaction{}).
		Where("ledger_id = ? AND source = ?", ledger.ID, models.TxSourceSpend).
		Update("created_at", base.Add(time.Second)).Error; err != nil {
		t.Fatalf("update spend timestamp: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/points/history?channel_id=ch_abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseBody(t, rec.Body.Bytes())
	if resp["success"] != true {
		t.Fatalf("success: want true, got %v", resp["success"])
	}
	data := resp["data"].(map[string]interface{})
	transactions := data["transactions"].([]interface{})
	if len(transactions) != 2 {
		t.Fatalf("want 2 transactions, got %d", len(transactions))
	}

	first := transactions[0].(map[string]interface{})
	if first["type"] != "spend" {
		t.Fatalf("first.type: want spend, got %v", first["type"])
	}
	if first["amount"] != float64(30) {
		t.Fatalf("first.amount: want 30, got %v", first["amount"])
	}
	if first["note"] != "avatar" {
		t.Fatalf("first.note: want avatar, got %v", first["note"])
	}

	second := transactions[1].(map[string]interface{})
	if second["type"] != "earn" {
		t.Fatalf("second.type: want earn, got %v", second["type"])
	}
	if second["amount"] != float64(100) {
		t.Fatalf("second.amount: want 100, got %v", second["amount"])
	}
}

func TestPointsHandler_GetHistory_RequiresChannelID(t *testing.T) {
	e := newPointsEnv(t)
	_, token := e.registerViewer(t, "history-missing-channel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/points/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseBody(t, rec.Body.Bytes())
	if resp["success"] != false {
		t.Fatalf("success: want false, got %v", resp["success"])
	}
	if resp["error"] != "channel_id is required" {
		t.Fatalf("error: want channel_id is required, got %v", resp["error"])
	}
}

func TestPointsHandler_RequiresJWT(t *testing.T) {
	e := newPointsEnv(t)

	for _, path := range []string{
		"/api/v1/users/me/points?channel_id=ch_abc",
		"/api/v1/users/me/points/history?channel_id=ch_abc",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d: %s", path, rec.Code, rec.Body.String())
		}
		resp := parseBody(t, rec.Body.Bytes())
		if resp["success"] != false {
			t.Fatalf("%s: success want false, got %v", path, resp["success"])
		}
		if resp["error"] != "authorization header required" {
			t.Fatalf("%s: error want authorization header required, got %v", path, resp["error"])
		}
	}
}
