package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type watchEnv struct {
	*testEnv
}

func newWatchTestEnv(t *testing.T) *watchEnv {
	t.Helper()

	base := newTestEnv(t)
	watchSvc := services.NewWatchService(base.db)
	watchH := handlers.NewWatchHandler(watchSvc)

	v1 := base.router.Group("/api/v1")
	ext := v1.Group("/extension")
	watch := ext.Group("/watch")
	watch.Use(middleware.JWTAuth(base.authSvc))
	{
		watch.POST("/start", watchH.StartSession)
		watch.POST("/heartbeat", watchH.Heartbeat)
		watch.POST("/end", watchH.EndSession)
		watch.GET("/balance", watchH.GetBalance)
	}

	return &watchEnv{testEnv: base}
}

func requestWithToken(method, path, token, body string) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func doWatchRequest(t *testing.T, router *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	router.ServeHTTP(w, requestWithToken(method, path, token, body))
	return w
}

func backdateActiveWatchSession(t *testing.T, env *watchEnv, channelID string, ago time.Duration) {
	t.Helper()

	ts := time.Now().Add(-ago)
	if err := env.db.Exec(
		`UPDATE watch_sessions SET last_heartbeat_at = ? WHERE channel_id = ? AND is_active = 1`,
		ts, channelID,
	).Error; err != nil {
		t.Fatalf("backdate active watch session: %v", err)
	}
}

func TestWatchAPI_StartSession_NoToken(t *testing.T) {
	env := newWatchTestEnv(t)

	w := doWatchRequest(t, env.router, http.MethodPost, "/api/v1/extension/watch/start", "", `{"channel_id":"ch_abc"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWatchAPI_Heartbeat_NoActiveSession(t *testing.T) {
	env := newWatchTestEnv(t)
	token, _ := env.registerUser(t, "watchuser", "watch@example.com", "password123")

	w := doWatchRequest(t, env.router, http.MethodPost, "/api/v1/extension/watch/heartbeat", token, `{"channel_id":"ch_abc"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWatchAPI_FullFlow_StartHeartbeatBalanceEnd(t *testing.T) {
	env := newWatchTestEnv(t)
	token, _ := env.registerUser(t, "flowuser", "flow@example.com", "password123")
	channelID := "ch_flow"

	startW := doWatchRequest(t, env.router, http.MethodPost, "/api/v1/extension/watch/start", token, `{"channel_id":"`+channelID+`"}`)
	if startW.Code != http.StatusOK {
		t.Fatalf("start: want 200, got %d: %s", startW.Code, startW.Body.String())
	}

	startResp := parseBody(t, startW.Body.Bytes())
	startData, _ := startResp["data"].(map[string]interface{})
	if startData["channel_id"] != channelID {
		t.Fatalf("start channel_id: want %s, got %v", channelID, startData["channel_id"])
	}

	for i := 0; i < 3; i++ {
		backdateActiveWatchSession(t, env, channelID, 25*time.Second)
		hbW := doWatchRequest(t, env.router, http.MethodPost, "/api/v1/extension/watch/heartbeat", token, `{"channel_id":"`+channelID+`"}`)
		if hbW.Code != http.StatusOK {
			t.Fatalf("heartbeat %d: want 200, got %d: %s", i+1, hbW.Code, hbW.Body.String())
		}
	}

	balanceW := doWatchRequest(t, env.router, http.MethodGet, "/api/v1/extension/watch/balance?channel_id="+channelID, token, "")
	if balanceW.Code != http.StatusOK {
		t.Fatalf("balance: want 200, got %d: %s", balanceW.Code, balanceW.Body.String())
	}

	balanceResp := parseBody(t, balanceW.Body.Bytes())
	balanceData, _ := balanceResp["data"].(map[string]interface{})
	if balanceData["spendable_balance"] != float64(1) {
		t.Fatalf("spendable_balance: want 1, got %v", balanceData["spendable_balance"])
	}
	if balanceData["cumulative_total"] != float64(1) {
		t.Fatalf("cumulative_total: want 1, got %v", balanceData["cumulative_total"])
	}

	endW := doWatchRequest(t, env.router, http.MethodPost, "/api/v1/extension/watch/end", token, `{"channel_id":"`+channelID+`"}`)
	if endW.Code != http.StatusOK {
		t.Fatalf("end: want 200, got %d: %s", endW.Code, endW.Body.String())
	}

	endResp := parseBody(t, endW.Body.Bytes())
	endData, _ := endResp["data"].(map[string]interface{})
	if endData["ended"] != true {
		t.Fatalf("ended: want true, got %v", endData["ended"])
	}

	var session models.WatchSession
	if err := env.db.Where("channel_id = ?", channelID).First(&session).Error; err != nil {
		t.Fatalf("load session: %v", err)
	}
	if session.IsActive {
		t.Fatal("expected session to be inactive after end")
	}
	if session.EndedAt == nil {
		t.Fatal("expected ended_at to be set after end")
	}
}
