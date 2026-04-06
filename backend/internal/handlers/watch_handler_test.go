package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

// watchEnv wraps testEnv and adds watch/points services + routes.
type watchEnv struct {
	*testEnv
	watchSvc  *services.WatchService
	pointsSvc *services.PointsService
}

func newWatchTestEnv(t *testing.T) *watchEnv {
	t.Helper()
	base := newTestEnv(t)

	watchSvc := services.NewWatchService(base.db)
	pointsSvc := services.NewPointsService(base.db, watchSvc)
	watchH := handlers.NewWatchHandler(watchSvc, pointsSvc)

	watch := base.router.Group("/api/v1/extension/watch")
	watch.Use(middleware.JWTAuth(base.authSvc))
	{
		watch.POST("/start", watchH.StartSession)
		watch.POST("/heartbeat", watchH.Heartbeat)
		watch.POST("/end", watchH.EndSession)
		watch.GET("/balance", watchH.GetBalance)
	}

	return &watchEnv{testEnv: base, watchSvc: watchSvc, pointsSvc: pointsSvc}
}

type failingPointsService struct {
	addHeartbeatTimeErr error
}

func (s *failingPointsService) AddHeartbeatTime(uuid.UUID, string, int64) error {
	return s.addHeartbeatTimeErr
}

// registerViewer registers a new user and returns their UUID + access token.
func (e *watchEnv) registerViewer(t *testing.T, suffix string) (uuid.UUID, string) {
	t.Helper()
	user, tokens, err := e.authSvc.Register(services.RegisterInput{
		Username: "viewer_" + suffix,
		Email:    fmt.Sprintf("viewer_%s@example.com", suffix),
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("registerViewer: %v", err)
	}
	return user.ID, tokens.AccessToken
}

// seedActiveSession inserts an active watch session with last_heartbeat_at
// set to `agoSeconds` seconds in the past, allowing the test to control
// whether the heartbeat delta will be accepted (>= 20 s) or ignored (< 20 s).
func (e *watchEnv) seedActiveSession(t *testing.T, userID uuid.UUID, channelID string, agoSeconds int) {
	t.Helper()
	now := time.Now()
	lastHB := now.Add(-time.Duration(agoSeconds) * time.Second)
	id := uuid.New()
	err := e.db.Exec(`
		INSERT INTO watch_sessions
			(id, user_id, channel_id, accumulated_seconds, rewarded_seconds, last_heartbeat_at, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 0, 0, ?, 1, ?, ?)`,
		id, userID, channelID, lastHB, now, now,
	).Error
	if err != nil {
		t.Fatalf("seedActiveSession: %v", err)
	}
}

// watchTimeSeconds reads the accumulated watch seconds for a user/channel from watch_time_stats.
// Returns 0 if no row exists.
func (e *watchEnv) watchTimeSeconds(t *testing.T, userID uuid.UUID, channelID string) int64 {
	t.Helper()
	var total int64
	if err := e.db.Raw(
		`SELECT COALESCE(total_watch_seconds, 0) FROM watch_time_stats WHERE user_id = ? AND channel_id = ?`,
		userID, channelID,
	).Scan(&total).Error; err != nil {
		t.Fatalf("watchTimeSeconds query failed: %v", err)
	}
	return total
}

// heartbeatRequest sends POST /api/v1/extension/watch/heartbeat with the given token and channel.
func heartbeatRequest(t *testing.T, router http.Handler, token, channelID string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"channel_id": channelID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extension/watch/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TestWatchHandler_Heartbeat_AccumulatesWatchTime verifies that a successful
// heartbeat (delta >= 20 s) causes PointsService.AddWatchTime to write into
// watch_time_stats.
func TestWatchHandler_Heartbeat_AccumulatesWatchTime(t *testing.T) {
	e := newWatchTestEnv(t)
	channelID := "ch_test_001"

	userID, token := e.registerViewer(t, "acc")
	e.seedActiveSession(t, userID, channelID, 30) // 30 s ago → delta accepted

	w := heartbeatRequest(t, e.router, token, channelID)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	secs := e.watchTimeSeconds(t, userID, channelID)
	if secs <= 0 {
		t.Fatalf("expected watch_time_stats to be > 0, got %d", secs)
	}
}

// TestWatchHandler_Heartbeat_DeltaTooSmall_NoWatchTime verifies that when the
// heartbeat delta is below the 20 s minimum, PointsService.AddWatchTime is NOT
// called (DeltaSeconds == 0 guard in the handler).
func TestWatchHandler_Heartbeat_DeltaTooSmall_NoWatchTime(t *testing.T) {
	e := newWatchTestEnv(t)
	channelID := "ch_test_002"

	userID, token := e.registerViewer(t, "small")
	e.seedActiveSession(t, userID, channelID, 5) // 5 s ago → delta ignored

	w := heartbeatRequest(t, e.router, token, channelID)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	secs := e.watchTimeSeconds(t, userID, channelID)
	if secs != 0 {
		t.Fatalf("expected watch_time_stats to remain 0, got %d", secs)
	}
}

// TestWatchHandler_Heartbeat_NoSession_Returns400 verifies that a heartbeat
// sent without an active session returns 400.
func TestWatchHandler_Heartbeat_NoSession_Returns400(t *testing.T) {
	e := newWatchTestEnv(t)

	_, token := e.registerViewer(t, "nosess")

	w := heartbeatRequest(t, e.router, token, "ch_no_session")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWatchHandler_Heartbeat_PointsServiceFailure_Returns500(t *testing.T) {
	base := newTestEnv(t)
	watchSvc := services.NewWatchService(base.db)
	watchH := handlers.NewWatchHandler(watchSvc, &failingPointsService{
		addHeartbeatTimeErr: fmt.Errorf("heartbeat aggregation failed"),
	})

	watch := base.router.Group("/api/v1/extension/watch")
	watch.Use(middleware.JWTAuth(base.authSvc))
	watch.POST("/heartbeat", watchH.Heartbeat)

	e := &watchEnv{testEnv: base, watchSvc: watchSvc}
	channelID := "ch_test_500"
	userID, token := e.registerViewer(t, "pointsfail")
	e.seedActiveSession(t, userID, channelID, 30)

	w := heartbeatRequest(t, e.router, token, channelID)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	var resp handlers.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != "internal server error" {
		t.Fatalf("expected internal server error, got %q", resp.Error)
	}
}

func TestWatchHandler_Heartbeat_Overflow_Returns500(t *testing.T) {
	e := newWatchTestEnv(t)
	channelID := "ch_handler_overflow"
	userID, token := e.registerViewer(t, "overflow")
	e.seedActiveSession(t, userID, channelID, 25)
	if err := e.db.Exec(
		`UPDATE watch_sessions
		 SET accumulated_seconds = ?, rewarded_seconds = ?
		 WHERE user_id = ? AND channel_id = ? AND is_active = 1`,
		math.MaxInt64-10,
		math.MaxInt64-10,
		userID,
		channelID,
	).Error; err != nil {
		t.Fatalf("seed near-overflow session counters: %v", err)
	}

	w := heartbeatRequest(t, e.router, token, channelID)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	var resp handlers.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false, got true")
	}
	if resp.Error != "internal server error" {
		t.Fatalf("expected internal server error, got %q", resp.Error)
	}
}

func TestWatchHandler_WatchTimeSeconds_QueryErrorFailsTest(t *testing.T) {
	if os.Getenv("WATCH_TIME_SECONDS_FATAL") == "1" {
		e := newWatchTestEnv(t)
		if err := e.db.Exec(`DROP TABLE watch_time_stats`).Error; err != nil {
			t.Fatalf("drop watch_time_stats: %v", err)
		}
		_ = e.watchTimeSeconds(t, uuid.New(), "ch_broken")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestWatchHandler_WatchTimeSeconds_QueryErrorFailsTest")
	cmd.Env = append(os.Environ(), "WATCH_TIME_SECONDS_FATAL=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected watchTimeSeconds to fail when the query errors")
	}
}
