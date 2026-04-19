package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

// raffleTestEnv wires only the raffle-related handlers.
type raffleTestEnv struct {
	db        *gorm.DB
	authSvc   *services.AuthService
	raffleSvc *services.RaffleService
	router    *gin.Engine
}

func newRaffleTestEnv(t *testing.T) *raffleTestEnv {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable fk: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := testConfig()
	authSvc := services.NewAuthService(db, cfg)
	raffleSvc := services.NewRaffleService(db)
	raffleH := handlers.NewRaffleHandler(raffleSvc)

	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")

	// Public claim routes
	claim := v1.Group("/claim")
	claim.GET("/:token", raffleH.GetClaim)
	claim.POST("/:token", raffleH.SubmitClaim)

	// Extension result
	v1.GET("/extension/raffles/:id/result", raffleH.GetResult)

	// Dashboard raffle routes (streamer role)
	dash := v1.Group("/dashboard")
	dash.Use(middleware.JWTAuth(authSvc))
	dash.Use(middleware.RequireRole(models.RoleStreamer))
	dash.POST("/raffles", raffleH.Create)
	dash.GET("/raffles", raffleH.List)
	dash.GET("/raffles/:id", raffleH.Get)
	dash.POST("/raffles/:id/entries/import-csv", raffleH.ImportCSV)
	dash.POST("/raffles/:id/draws", raffleH.DrawNext)
	dash.GET("/raffles/:id/draws", raffleH.ListDraws)
	dash.POST("/raffles/:id/complete", raffleH.Complete)

	return &raffleTestEnv{db: db, authSvc: authSvc, raffleSvc: raffleSvc, router: r}
}

// registerStreamer creates a streamer-role user and returns an access token.
func (e *raffleTestEnv) registerStreamer(t *testing.T, username, email, password string) string {
	t.Helper()
	user, _, err := e.authSvc.Register(services.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := e.db.Model(user).Update("role", models.RoleStreamer).Error; err != nil {
		t.Fatalf("set role: %v", err)
	}
	_, tokens, err := e.authSvc.Login(services.LoginInput{Email: email, Password: password})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return tokens.AccessToken
}

// createTwitchLinkedUser registers a tachigo user and links a Twitch login so
// ImportCSV can match the entry.
func (e *raffleTestEnv) createTwitchLinkedUser(t *testing.T, twitchLogin string) {
	t.Helper()
	// username matches Twitch login name; provider_id stores the numeric Twitch ID.
	user, _, err := e.authSvc.Register(services.RegisterInput{
		Username: twitchLogin,
		Email:    twitchLogin + "@test.com",
		Password: "pass1234",
	})
	if err != nil {
		t.Fatalf("register %s: %v", twitchLogin, err)
	}
	providerID := "twitch_id_" + twitchLogin // simulate numeric Twitch ID
	id, _ := uuid.NewV7()
	if err := e.db.Exec(
		`INSERT INTO auth_providers (id, user_id, provider, provider_id, created_at, updated_at) VALUES (?, ?, 'twitch', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		id.String(), user.ID.String(), providerID,
	).Error; err != nil {
		t.Fatalf("link twitch auth_provider for %s: %v", twitchLogin, err)
	}
}

func bearer(token string) string { return "Bearer " + token }

// ── Tests ──────────────────────────────────────────────────────────────────────

func TestRaffle_Create(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "streamer1", "s1@test.com", "pass1234")

	body, _ := json.Marshal(map[string]string{"title": "月底大抽獎"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	raffle := data["raffle"].(map[string]interface{})
	if raffle["title"] != "月底大抽獎" {
		t.Errorf("unexpected title: %v", raffle["title"])
	}
	if raffle["status"] != "draft" {
		t.Errorf("expected draft status, got %v", raffle["status"])
	}
}

func TestRaffle_Create_Unauthorized(t *testing.T) {
	env := newRaffleTestEnv(t)
	body, _ := json.Marshal(map[string]string{"title": "test"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestRaffle_List(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "streamer2", "s2@test.com", "pass1234")

	// Create two raffles
	for _, title := range []string{"Raffle A", "Raffle B"} {
		body, _ := json.Marshal(map[string]string{"title": title})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer(token))
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create raffle: %d", w.Code)
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/raffles", nil)
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	resp := parseBody(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	raffles := data["raffles"].([]interface{})
	if len(raffles) != 2 {
		t.Errorf("want 2 raffles, got %d", len(raffles))
	}
}

func TestRaffle_Get_Forbidden(t *testing.T) {
	env := newRaffleTestEnv(t)
	ownerToken := env.registerStreamer(t, "owner", "owner@test.com", "pass1234")
	otherToken := env.registerStreamer(t, "other", "other@test.com", "pass1234")

	// Create raffle as owner
	body, _ := json.Marshal(map[string]string{"title": "private"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(ownerToken))
	env.router.ServeHTTP(w, req)
	resp := parseBody(t, w.Body.Bytes())
	raffleID := resp["data"].(map[string]interface{})["raffle"].(map[string]interface{})["id"].(string)

	// Access as other streamer → 403
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/raffles/"+raffleID, nil)
	req2.Header.Set("Authorization", bearer(otherToken))
	env.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w2.Code)
	}
}

func TestRaffle_ImportCSV_And_DrawNext(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "host", "host@test.com", "pass1234")

	// Create raffle
	body, _ := json.Marshal(map[string]string{"title": "CSV draw test"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)
	resp := parseBody(t, w.Body.Bytes())
	raffleID := resp["data"].(map[string]interface{})["raffle"].(map[string]interface{})["id"].(string)

	// Create tachigo users linked to Twitch logins before importing.
	for _, login := range []string{"userA", "userB", "userC"} {
		env.createTwitchLinkedUser(t, login)
	}

	// Upload CSV with 3 entries
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "entries.csv")
	fmt.Fprintln(fw, "twitch_login,display_name")
	fmt.Fprintln(fw, "userA,User A")
	fmt.Fprintln(fw, "userB,User B")
	fmt.Fprintln(fw, "userC,User C")
	mw.Close()

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/entries/import-csv",
		&buf)
	req2.Header.Set("Content-Type", mw.FormDataContentType())
	req2.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("import-csv want 200, got %d: %s", w2.Code, w2.Body.String())
	}
	importResp := parseBody(t, w2.Body.Bytes())
	importData := importResp["data"].(map[string]interface{})
	if int(importData["imported"].(float64)) != 3 {
		t.Errorf("want imported=3, got %v", importData["imported"])
	}

	// DrawNext — draw all 3
	drawnTokens := map[string]bool{}
	for i := 0; i < 3; i++ {
		w3 := httptest.NewRecorder()
		req3, _ := http.NewRequest(http.MethodPost,
			"/api/v1/dashboard/raffles/"+raffleID+"/draws", nil)
		req3.Header.Set("Authorization", bearer(token))
		env.router.ServeHTTP(w3, req3)
		if w3.Code != http.StatusCreated {
			t.Fatalf("draw %d: want 201, got %d: %s", i+1, w3.Code, w3.Body.String())
		}
		drawResp := parseBody(t, w3.Body.Bytes())
		draw := drawResp["data"].(map[string]interface{})["draw"].(map[string]interface{})
		claimToken := draw["claim_token"].(string)
		if drawnTokens[claimToken] {
			t.Errorf("duplicate claim_token drawn: %s", claimToken)
		}
		drawnTokens[claimToken] = true
	}

	// 4th draw → 409 exhausted
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/draws", nil)
	req4.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w4, req4)
	if w4.Code != http.StatusConflict {
		t.Fatalf("exhausted draw: want 409, got %d", w4.Code)
	}
}

func TestRaffle_Complete(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "completer", "comp@test.com", "pass1234")

	body, _ := json.Marshal(map[string]string{"title": "to complete"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)
	resp := parseBody(t, w.Body.Bytes())
	raffleID := resp["data"].(map[string]interface{})["raffle"].(map[string]interface{})["id"].(string)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/complete", nil)
	req2.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("complete: want 200, got %d: %s", w2.Code, w2.Body.String())
	}
	completeResp := parseBody(t, w2.Body.Bytes())
	raffle := completeResp["data"].(map[string]interface{})["raffle"].(map[string]interface{})
	if raffle["status"] != "completed" {
		t.Errorf("expected completed status, got %v", raffle["status"])
	}
}

func TestRaffle_ClaimFlow(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "claimhost", "ch@test.com", "pass1234")

	// Create raffle + import one entry
	body, _ := json.Marshal(map[string]string{"title": "claim flow"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)
	resp := parseBody(t, w.Body.Bytes())
	raffleID := resp["data"].(map[string]interface{})["raffle"].(map[string]interface{})["id"].(string)

	env.createTwitchLinkedUser(t, "winner1")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "entries.csv")
	fmt.Fprintln(fw, "winner1")
	mw.Close()
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/entries/import-csv", &buf)
	req2.Header.Set("Content-Type", mw.FormDataContentType())
	req2.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w2, req2)

	// Draw
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/draws", nil)
	req3.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w3, req3)
	drawResp := parseBody(t, w3.Body.Bytes())
	claimToken := drawResp["data"].(map[string]interface{})["draw"].(map[string]interface{})["claim_token"].(string)

	// GET /claim/:token
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodGet, "/api/v1/claim/"+claimToken, nil)
	env.router.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("get claim: want 200, got %d: %s", w4.Code, w4.Body.String())
	}

	// POST /claim/:token — submit shipping info
	claimBody, _ := json.Marshal(map[string]string{
		"recipient_name": "王大明",
		"phone":          "0912345678",
		"address_line1":  "台北市信義區信義路五段7號",
		"city":           "台北市",
		"country":        "TW",
	})
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodPost, "/api/v1/claim/"+claimToken,
		bytes.NewReader(claimBody))
	req5.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("submit claim: want 200, got %d: %s", w5.Code, w5.Body.String())
	}

	// POST again → 409
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodPost, "/api/v1/claim/"+claimToken,
		bytes.NewReader(claimBody))
	req6.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(w6, req6)
	if w6.Code != http.StatusConflict {
		t.Fatalf("duplicate claim: want 409, got %d", w6.Code)
	}
}

func TestRaffle_ClaimExpired(t *testing.T) {
	env := newRaffleTestEnv(t)

	// Insert an expired draw directly into DB
	raffleID := "00000000-0000-0000-0000-000000000001"
	entryID := "00000000-0000-0000-0000-000000000002"
	drawID := "00000000-0000-0000-0000-000000000003"
	expiredToken := "expired-token-test"

	// Need a user for raffle.user_id
	env.db.Exec(`INSERT INTO users (id, role) VALUES ('00000000-0000-0000-0000-000000000099', 'streamer')`)
	env.db.Exec(`INSERT INTO raffles (id, user_id, title, status) VALUES (?, '00000000-0000-0000-0000-000000000099', 'x', 'active')`, raffleID)
	env.db.Exec(`INSERT INTO raffle_entries (id, raffle_id, twitch_login) VALUES (?, ?, 'testuser')`, entryID, raffleID)
	env.db.Exec(`INSERT INTO raffle_draws (id, raffle_id, entry_id, claim_token, claim_expires_at, drawn_at) VALUES (?, ?, ?, ?, ?, ?)`,
		drawID, raffleID, entryID, expiredToken,
		time.Now().Add(-24*time.Hour).Format(time.RFC3339),
		time.Now().Add(-25*time.Hour).Format(time.RFC3339),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/claim/"+expiredToken, nil)
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusGone {
		t.Fatalf("expired claim: want 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRaffle_GetResult_Extension(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "exthost", "ext@test.com", "pass1234")

	// Create raffle + entry + draw
	body, _ := json.Marshal(map[string]string{"title": "ext result"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)
	resp := parseBody(t, w.Body.Bytes())
	raffleID := resp["data"].(map[string]interface{})["raffle"].(map[string]interface{})["id"].(string)

	env.createTwitchLinkedUser(t, "viewer1")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "e.csv")
	fmt.Fprintln(fw, "viewer1")
	mw.Close()
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/entries/import-csv", &buf)
	req2.Header.Set("Content-Type", mw.FormDataContentType())
	req2.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w2, req2)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost,
		"/api/v1/dashboard/raffles/"+raffleID+"/draws", nil)
	req3.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w3, req3)

	// Extension result (no auth)
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodGet,
		"/api/v1/extension/raffles/"+raffleID+"/result", nil)
	env.router.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("ext result: want 200, got %d: %s", w4.Code, w4.Body.String())
	}
	extResp := parseBody(t, w4.Body.Bytes())
	draws := extResp["data"].(map[string]interface{})["draws"].([]interface{})
	if len(draws) != 1 {
		t.Errorf("want 1 draw, got %d", len(draws))
	}
}

func TestRaffle_CSVDuplicateSkipped(t *testing.T) {
	env := newRaffleTestEnv(t)
	token := env.registerStreamer(t, "dedup", "dup@test.com", "pass1234")

	body, _ := json.Marshal(map[string]string{"title": "dedup"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/dashboard/raffles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer(token))
	env.router.ServeHTTP(w, req)
	resp := parseBody(t, w.Body.Bytes())
	raffleID := resp["data"].(map[string]interface{})["raffle"].(map[string]interface{})["id"].(string)

	uploadCSV := func(csv string) map[string]interface{} {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", "e.csv")
		fw.Write([]byte(csv))
		mw.Close()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost,
			"/api/v1/dashboard/raffles/"+raffleID+"/entries/import-csv", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", bearer(token))
		env.router.ServeHTTP(w, req)
		return parseBody(t, w.Body.Bytes())["data"].(map[string]interface{})
	}

	// Create tachigo users for userX, userY, userZ before importing.
	for _, login := range []string{"userX", "userY", "userZ"} {
		env.createTwitchLinkedUser(t, login)
	}

	r1 := uploadCSV("userX\nuserY\n")
	if int(r1["imported"].(float64)) != 2 {
		t.Errorf("first import: want 2, got %v", r1["imported"])
	}

	r2 := uploadCSV("userX\nuserZ\n")
	if int(r2["imported"].(float64)) != 1 {
		t.Errorf("second import: want 1 new, got %v", r2["imported"])
	}
	if int(r2["skipped"].(float64)) != 1 {
		t.Errorf("second import: want 1 skipped, got %v", r2["skipped"])
	}
}

// testConfig returns minimal config for tests.
func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
	}
}
