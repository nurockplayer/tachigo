package handlers_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/models"
)

// ─── Register ────────────────────────────────────────────────────────────────

func TestRegisterHandler_Success(t *testing.T) {
	env := newTestEnv(t)

	body := `{"username":"testuser","email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	if resp["success"] != true {
		t.Error("want success: true")
	}
	assertRefreshCookieSet(t, w, "", http.SameSiteLaxMode, false)
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "existing", "dup@example.com", "password123")

	body := `{"username":"newuser","email":"dup@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", w.Code)
	}
}

func TestRegisterHandler_DuplicateUsername(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "sameuser", "first@example.com", "password123")

	body := `{"username":"sameuser","email":"second@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", w.Code)
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestRegisterHandler_ShortPassword(t *testing.T) {
	env := newTestEnv(t)

	body := `{"username":"user","email":"u@example.com","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestLoginHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "loginuser", "login@example.com", "mypassword")

	body := `{"email":"login@example.com","password":"mypassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, "", http.SameSiteLaxMode, false)
}

func TestLoginHandler_SetsSecureRefreshCookieInProduction(t *testing.T) {
	env := newTestEnvWithServerEnv(t, "production")
	env.registerUser(t, "prodlogin", "prodlogin@example.com", "mypassword")

	body := `{"email":"prodlogin@example.com","password":"mypassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, "", http.SameSiteLaxMode, true)
}

func TestLoginHandler_SetsSameSiteNoneRefreshCookieForCrossSiteFrontendInProduction(t *testing.T) {
	env := newTestEnvWithConfig(t, "production", "https://dashboard.example.org")
	env.registerUser(t, "crosssite", "crosssite@example.com", "mypassword")

	body := `{"email":"crosssite@example.com","password":"mypassword"}`
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "api.example.com"
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, "", http.SameSiteNoneMode, true)
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "user", "user@example.com", "correctpass")

	body := `{"email":"user@example.com","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestLoginHandler_UnknownEmail(t *testing.T) {
	env := newTestEnv(t)

	body := `{"email":"nobody@example.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestLoginHandler_MissingFields(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestRefreshHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	_, refreshToken := env.registerUser(t, "ruser", "r@example.com", "password123")

	body := fmt.Sprintf(`{"refresh_token":"%s"}`, refreshToken)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, refreshToken, http.SameSiteLaxMode, false)
}

func TestRefreshHandler_SuccessWithCookie(t *testing.T) {
	env := newTestEnv(t)
	_, refreshToken := env.registerUser(t, "cookieuser", "cookie@example.com", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/api/v1/auth"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, refreshToken, http.SameSiteLaxMode, false)
}

func TestRefreshHandler_PrefersCookieOverBodyToken(t *testing.T) {
	env := newTestEnv(t)
	_, refreshToken := env.registerUser(t, "prefcookie", "prefcookie@example.com", "password123")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		bytes.NewBufferString(`{"refresh_token":"badtoken"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/api/v1/auth"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, refreshToken, http.SameSiteLaxMode, false)
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{"refresh_token":"badtoken"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestRefreshHandler_MissingToken(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestLogoutHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	_, refreshToken := env.registerUser(t, "luser", "l@example.com", "password123")

	body := fmt.Sprintf(`{"refresh_token":"%s"}`, refreshToken)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	assertRefreshCookieCleared(t, w, http.SameSiteLaxMode, false)
}

func TestLogoutHandler_SuccessWithCookie(t *testing.T) {
	env := newTestEnv(t)
	_, refreshToken := env.registerUser(t, "cookielogout", "logout@example.com", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/api/v1/auth"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	assertRefreshCookieCleared(t, w, http.SameSiteLaxMode, false)
}

func TestLogoutHandler_PrefersCookieOverBodyToken(t *testing.T) {
	env := newTestEnv(t)
	_, refreshToken := env.registerUser(t, "logoutpref", "logoutpref@example.com", "password123")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		bytes.NewBufferString(`{"refresh_token":"badtoken"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/api/v1/auth"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	assertRefreshCookieCleared(t, w, http.SameSiteLaxMode, false)
}

func TestLogoutHandler_ClearsCookieWithSameSiteNoneForCrossSiteFrontendInProduction(t *testing.T) {
	env := newTestEnvWithConfig(t, "production", "https://dashboard.example.org")
	_, refreshToken := env.registerUser(t, "crosssitelogout", "crosssitelogout@example.com", "password123")

	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/api/v1/auth/logout", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "api.example.com"
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/api/v1/auth"})
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	assertRefreshCookieCleared(t, w, http.SameSiteNoneMode, true)
}

func TestLogoutHandler_MissingToken(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func assertRefreshCookieSet(
	t *testing.T,
	w *httptest.ResponseRecorder,
	previousValue string,
	expectedSameSite http.SameSite,
	expectedSecure bool,
) {
	t.Helper()

	cookie := responseCookie(t, w, "refresh_token")
	if cookie.Value == "" {
		t.Fatal("expected refresh token cookie to be set")
	}
	if cookie.MaxAge <= 0 {
		t.Fatalf("expected refresh token cookie MaxAge > 0, got %d", cookie.MaxAge)
	}
	if previousValue != "" && cookie.Value == previousValue {
		t.Fatal("expected refresh token cookie to rotate")
	}
	if !cookie.HttpOnly {
		t.Fatal("expected refresh token cookie to be HttpOnly")
	}
	if cookie.Path != "/api/v1/auth" {
		t.Fatalf("expected refresh token cookie path /api/v1/auth, got %q", cookie.Path)
	}
	if cookie.SameSite != expectedSameSite {
		t.Fatalf("expected refresh token cookie SameSite %v, got %v", expectedSameSite, cookie.SameSite)
	}
	if cookie.Secure != expectedSecure {
		t.Fatalf("expected refresh token cookie Secure %t, got %t", expectedSecure, cookie.Secure)
	}
}

func assertRefreshCookieCleared(
	t *testing.T,
	w *httptest.ResponseRecorder,
	expectedSameSite http.SameSite,
	expectedSecure bool,
) {
	t.Helper()

	cookie := responseCookie(t, w, "refresh_token")
	if cookie.Value != "" {
		t.Fatalf("expected cleared refresh token cookie, got %q", cookie.Value)
	}
	if cookie.MaxAge >= 0 {
		t.Fatalf("expected cleared refresh token cookie MaxAge < 0, got %d", cookie.MaxAge)
	}
	if cookie.Path != "/api/v1/auth" {
		t.Fatalf("expected refresh token cookie path /api/v1/auth, got %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected cleared refresh token cookie to be HttpOnly")
	}
	if cookie.SameSite != expectedSameSite {
		t.Fatalf("expected cleared refresh token cookie SameSite %v, got %v", expectedSameSite, cookie.SameSite)
	}
	if cookie.Secure != expectedSecure {
		t.Fatalf("expected cleared refresh token cookie Secure %t, got %t", expectedSecure, cookie.Secure)
	}
}

func responseCookie(t *testing.T, w *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("expected cookie %q to be present", name)
	return nil
}

// ─── Web3 Nonce ───────────────────────────────────────────────────────────────

func TestWeb3NonceHandler_Success(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/web3/nonce",
		bytes.NewBufferString(`{"address":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	if data["nonce"] == "" || data["nonce"] == nil {
		t.Error("expected non-empty nonce in response")
	}
}

func TestWeb3NonceHandler_MissingAddress(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/web3/nonce", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestWeb3VerifyHandler_SuccessSetsRefreshCookieAndConsumesNonce(t *testing.T) {
	env := newTestEnv(t)
	key, addr := newHandlerTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "handler-web3-verify-success"
	nonceRecord := seedHandlerWalletNonce(t, env, addr, nonce)
	msg := handlerSIWEMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := handlerSignSIWE(t, msg, key)
	body := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce, sig)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/web3/verify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertRefreshCookieSet(t, w, "", http.SameSiteLaxMode, false)

	resp := parseBody(t, w.Body.Bytes())
	data, ok := resp["data"].(map[string]interface{})
	if !ok || data == nil {
		t.Fatalf("expected data map in response, resp=%#v", resp)
	}
	tokens, ok := data["tokens"].(map[string]interface{})
	if !ok || tokens == nil {
		t.Fatalf("expected tokens map in response, data=%#v", data)
	}
	accessToken, ok := tokens["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatalf("expected non-empty access_token in response: %#v", tokens)
	}
	refreshToken, ok := tokens["refresh_token"].(string)
	if !ok || refreshToken == "" {
		t.Fatalf("expected non-empty refresh_token in response: %#v", tokens)
	}

	var provider models.AuthProvider
	if err := env.db.Where("provider = ? AND provider_id = ?", models.ProviderWeb3, addr).First(&provider).Error; err != nil {
		t.Fatalf("web3 provider not found: %v", err)
	}

	var nonceCount int64
	env.db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 0 {
		t.Fatalf("nonce should be consumed, got %d rows", nonceCount)
	}
}

func TestWeb3VerifyHandler_InvalidSignatureReturns401AndKeepsNonce(t *testing.T) {
	env := newTestEnv(t)
	_, addr := newHandlerTestWallet(t)
	wrongKey, _ := newHandlerTestWallet(t)
	lookupAddr := strings.ToLower(addr)
	nonce := "handler-web3-verify-bad-signature"
	nonceRecord := seedHandlerWalletNonce(t, env, addr, nonce)
	msg := handlerSIWEMessage(lookupAddr, nonce, nonceRecord.CreatedAt.UTC().Format(time.RFC3339))
	sig := handlerSignSIWE(t, msg, wrongKey)
	body := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce, sig)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/web3/verify", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	if resp["error"] != "invalid wallet signature" {
		t.Fatalf("want invalid wallet signature error, got %#v", resp["error"])
	}

	var nonceCount int64
	env.db.Model(&models.Web3Nonce{}).Where("nonce = ?", nonce).Count(&nonceCount)
	if nonceCount != 1 {
		t.Fatalf("invalid signature should keep nonce for retry, got %d rows", nonceCount)
	}
}

// ─── Protected route without token ───────────────────────────────────────────

func TestProtected_NoToken_Unauthorized(t *testing.T) {
	env := newTestEnv(t)

	for _, path := range []string{
		"/api/v1/users/me",
		"/api/v1/users/me/providers",
		"/api/v1/users/me/addresses",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: want 401, got %d", path, w.Code)
		}
	}
}

func TestProtected_InvalidToken_Unauthorized(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer totally.invalid.token")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}
