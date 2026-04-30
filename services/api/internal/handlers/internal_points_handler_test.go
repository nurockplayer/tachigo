package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/models"
)

func TestInternalPointsHandler_NormalizesEmailQueryAndReturnsCanonicalEmail(t *testing.T) {
	env := newTestEnv(t)

	canonicalEmail := "viewer@example.com"
	user := &models.User{
		Username: stringPtr("viewer"),
		Email:    &canonicalEmail,
	}
	if err := env.db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := env.db.Create(&models.PointsLedger{
		UserID:           user.ID,
		ChannelID:        "ch_one",
		SpendableBalance: 40,
		CumulativeTotal:  50,
	}).Error; err != nil {
		t.Fatalf("create ledger 1: %v", err)
	}
	if err := env.db.Create(&models.PointsLedger{
		UserID:           user.ID,
		ChannelID:        "ch_two",
		SpendableBalance: 60,
		CumulativeTotal:  70,
	}).Error; err != nil {
		t.Fatalf("create ledger 2: %v", err)
	}

	r := gin.New()
	r.GET("/api/v1/internal/tachiya/users/points/balance", handlers.NewInternalPointsHandler(env.db).GetUserPointsBalance)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/tachiya/users/points/balance?email=%20Viewer@Example.com%20", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	resp := parseBody(t, rec.Body.Bytes())
	data := resp["data"].(map[string]interface{})

	if data["email"] != canonicalEmail {
		t.Fatalf("want canonical email %q, got %v", canonicalEmail, data["email"])
	}
	if data["spendable_balance"] != float64(100) {
		t.Fatalf("want spendable_balance 100, got %v", data["spendable_balance"])
	}
	if data["cumulative_total"] != float64(120) {
		t.Fatalf("want cumulative_total 120, got %v", data["cumulative_total"])
	}
}

func stringPtr(s string) *string {
	return &s
}
