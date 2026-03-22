package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

const minimalAddress = `{"recipient_name":"John Doe","address_line1":"123 Main St","city":"Taipei"}`

// createAddress is a helper that POSTs an address and returns the address ID.
func createAddress(t *testing.T, env *testEnv, accessToken, body string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createAddress: want 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	address, _ := data["address"].(map[string]interface{})
	return address["id"].(string)
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestCreateAddressHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "addruser", "addr@example.com", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewBufferString(minimalAddress))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	address, _ := data["address"].(map[string]interface{})
	if address["recipient_name"] != "John Doe" {
		t.Errorf("recipient_name: want John Doe, got %v", address["recipient_name"])
	}
	// Default country should be TW
	if address["country"] != "TW" {
		t.Errorf("country: want TW, got %v", address["country"])
	}
}

func TestCreateAddressHandler_MissingRequired(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "addr2user", "addr2@example.com", "password123")

	// Missing address_line1 and city
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses",
		bytes.NewBufferString(`{"recipient_name":"John"}`))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── List ────────────────────────────────────────────────────────────────────

func TestListAddressesHandler_Empty(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "listuser0", "list0@example.com", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/addresses", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	addresses, _ := data["addresses"].([]interface{})
	if len(addresses) != 0 {
		t.Errorf("want 0 addresses, got %d", len(addresses))
	}
}

func TestListAddressesHandler_ReturnsAll(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "listuser", "list@example.com", "password123")

	createAddress(t, env, accessToken, minimalAddress)
	createAddress(t, env, accessToken, minimalAddress)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/addresses", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	addresses, _ := data["addresses"].([]interface{})
	if len(addresses) != 2 {
		t.Errorf("want 2 addresses, got %d", len(addresses))
	}
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestUpdateAddressHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "updaddruser", "updaddr@example.com", "password123")
	addrID := createAddress(t, env, accessToken, minimalAddress)

	updateBody := `{"recipient_name":"Jane Smith","address_line1":"456 Other St","city":"Kaohsiung"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+addrID,
		bytes.NewBufferString(updateBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	address, _ := data["address"].(map[string]interface{})
	if address["recipient_name"] != "Jane Smith" {
		t.Errorf("recipient_name: want Jane Smith, got %v", address["recipient_name"])
	}
}

func TestUpdateAddressHandler_NotFound(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "updnfuser", "updnf@example.com", "password123")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/00000000-0000-0000-0000-000000000000",
		bytes.NewBufferString(minimalAddress))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestUpdateAddressHandler_InvalidID(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "invalididuser", "invalidid@example.com", "password123")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/not-a-uuid",
		bytes.NewBufferString(minimalAddress))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── Delete ──────────────────────────────────────────────────────────────────

func TestDeleteAddressHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "deluser", "del@example.com", "password123")
	addrID := createAddress(t, env, accessToken, minimalAddress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+addrID, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAddressHandler_NotFound(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "delnfuser", "delnf@example.com", "password123")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestDeleteAddressHandler_OtherUsersAddress(t *testing.T) {
	env := newTestEnv(t)
	user1Token, _ := env.registerUser(t, "owner", "owner@example.com", "password123")
	user2Token, _ := env.registerUser(t, "thief", "thief@example.com", "password123")

	addrID := createAddress(t, env, user1Token, minimalAddress)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+addrID, nil)
	req.Header.Set("Authorization", "Bearer "+user2Token)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when deleting another user's address, got %d", w.Code)
	}
}

// ─── SetDefault ──────────────────────────────────────────────────────────────

func TestSetDefaultAddressHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "defuser", "def@example.com", "password123")
	addrID := createAddress(t, env, accessToken, minimalAddress)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+addrID+"/default", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	address, _ := data["address"].(map[string]interface{})
	if address["is_default"] != true {
		t.Errorf("is_default: want true, got %v", address["is_default"])
	}
}

func TestSetDefaultAddressHandler_NotFound(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "defnfuser", "defnf@example.com", "password123")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/00000000-0000-0000-0000-000000000000/default", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}
