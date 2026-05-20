package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/models"
)

type internalPointsContextKey struct{}

func installInternalPointsDBContextProbe(t *testing.T, db *gorm.DB, key, want any) func() (int, int) {
	t.Helper()

	var querySeen int
	var rawSeen int
	name := "test:internal_points_db_context:" + uuid.NewString()
	queryProbe := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Context != nil && tx.Statement.Context.Value(key) == want {
			querySeen++
		}
	}
	rawProbe := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Context != nil && tx.Statement.Context.Value(key) == want {
			rawSeen++
		}
	}

	if err := db.Callback().Query().Before("gorm:query").Register(name+":query", queryProbe); err != nil {
		t.Fatalf("register query context probe: %v", err)
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register(name+":raw", rawProbe); err != nil {
		t.Fatalf("register raw context probe: %v", err)
	}
	if err := db.Callback().Row().Before("gorm:row").Register(name+":row", rawProbe); err != nil {
		t.Fatalf("register row context probe: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(name + ":query")
		_ = db.Callback().Raw().Remove(name + ":raw")
		_ = db.Callback().Row().Remove(name + ":row")
	})

	return func() (int, int) {
		return querySeen, rawSeen
	}
}

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

func TestInternalPointsHandler_GetUserPointsBalance_UsesRequestContext(t *testing.T) {
	env := newTestEnv(t)
	key := internalPointsContextKey{}
	seen := installInternalPointsDBContextProbe(t, env.db, key, "internal-points")
	ctx := context.WithValue(context.Background(), key, "internal-points")

	canonicalEmail := "context@example.com"
	user := &models.User{
		Username: stringPtr("context_viewer"),
		Email:    &canonicalEmail,
	}
	if err := env.db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	r := gin.New()
	r.GET("/api/v1/internal/tachiya/users/points/balance", handlers.NewInternalPointsHandler(env.db).GetUserPointsBalance)

	req := httptest.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"/api/v1/internal/tachiya/users/points/balance?email=context@example.com",
		nil,
	)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	querySeen, rawSeen := seen()
	if querySeen == 0 || rawSeen == 0 {
		t.Fatalf("expected internal points handler query and raw DB operations to carry request context, got query=%d raw=%d", querySeen, rawSeen)
	}
}

func stringPtr(s string) *string {
	return &s
}
