package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMeHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "meuser", "me@example.com", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	user, _ := data["user"].(map[string]interface{})
	if user["email"] != "me@example.com" {
		t.Errorf("email: want me@example.com, got %v", user["email"])
	}
}

func TestUpdateMeHandler_Username(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "oldname", "update@example.com", "password123")

	body := `{"username":"brandnewname"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	user, _ := data["user"].(map[string]interface{})
	if user["username"] != "brandnewname" {
		t.Errorf("username: want brandnewname, got %v", user["username"])
	}
}

func TestUpdateMeHandler_DuplicateUsername(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "taken", "taken@example.com", "password123")
	accessToken, _ := env.registerUser(t, "myuser", "my@example.com", "password123")

	body := `{"username":"taken"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", w.Code)
	}
}

func TestUpdateMeHandler_AvatarURL(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "avataruser", "avatar@example.com", "password123")

	body := `{"avatar_url":"https://example.com/pic.png"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListProvidersHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "provuser", "prov@example.com", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/providers", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	providers, _ := data["providers"].([]interface{})
	// Register creates an email provider record
	if len(providers) == 0 {
		t.Error("expected at least one provider after registration")
	}
}
