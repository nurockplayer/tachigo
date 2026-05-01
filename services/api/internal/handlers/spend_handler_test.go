package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type mockSpendBurnCaller struct {
	txHash string
	err    error
}

func (m *mockSpendBurnCaller) BurnOnChain(_ context.Context, _ string, _ int64) (string, error) {
	return m.txHash, m.err
}

type mockSpendTachiyaClient struct {
	err error
}

func (m *mockSpendTachiyaClient) RedeemCoupon(_ context.Context, _ string, _ int64) (string, error) {
	return "", m.err
}

func newSpendTestEnv(t *testing.T, tachiyaClient services.TachiyaClient) (*testEnv, *gin.Engine) {
	t.Helper()
	env := newTestEnv(t)
	spendSvc := services.NewSpendService(env.db, config.ContractConfig{}, nil, tachiyaClient)
	spendSvc.SetBurnCallerForTest(&mockSpendBurnCaller{txHash: "0xburn123"})
	spendH := handlers.NewSpendHandler(spendSvc)

	protected := env.router.Group("/api/v1")
	protected.Use(middleware.JWTAuth(env.authSvc))
	protected.POST("/spend/redeem", spendH.Redeem)

	return env, env.router
}

func seedTachiBalanceForHandler(t *testing.T, env *testEnv, userID uuid.UUID, balance int64) {
	t.Helper()
	if err := env.db.Exec(`
		INSERT INTO tachi_balances (id, user_id, balance, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, uuid.New().String(), userID.String(), balance).Error; err != nil {
		t.Fatalf("seedTachiBalanceForHandler: %v", err)
	}
}

func TestSpendHandler_RedeemTachiyaFailureReturnsServiceUnavailable(t *testing.T) {
	env, r := newSpendTestEnv(t, &mockSpendTachiyaClient{err: errors.New("tachiya unavailable")})
	token, _ := env.registerUser(t, "spend-tachiya-fail", "spend-tachiya-fail@example.com", "password123")
	userID := resolveUserID(t, env, "spend-tachiya-fail@example.com")
	seedWeb3ProviderForHandler(t, env, userID)
	seedTachiBalanceForHandler(t, env, userID, 300)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/spend/redeem", bytes.NewBufferString(`{"coupon_id":"coupon-123","amount":100}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}
