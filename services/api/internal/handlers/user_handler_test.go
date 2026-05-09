package handlers_test

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tachigo/tachigo/internal/models"
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

func newHandlerTestWallet(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	addr := common.HexToAddress(crypto.PubkeyToAddress(key.PublicKey).Hex()).Hex()
	return key, addr
}

func handlerSIWEMessage(address, nonce, issuedAt string) string {
	return fmt.Sprintf(
		"tachigo.io wants you to sign in with your Ethereum account:\n%s\n\nSign in to Tachigo\n\nNonce: %s\nIssued At: %s",
		address, nonce, issuedAt,
	)
}

func handlerSignSIWE(t *testing.T, message string, key *ecdsa.PrivateKey) string {
	t.Helper()
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed))
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sig[64] += 27
	return "0x" + hex.EncodeToString(sig)
}

func seedHandlerWalletNonce(t *testing.T, env *testEnv, address, nonce string) *models.Web3Nonce {
	t.Helper()
	record := &models.Web3Nonce{
		Nonce:     nonce,
		Address:   strings.ToLower(address),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := env.db.Create(record).Error; err != nil {
		t.Fatalf("seed nonce: %v", err)
	}
	return record
}

func TestLinkWalletHandler_NoAuth(t *testing.T) {
	env := newTestEnv(t)

	body := `{"address":"0x1234","nonce":"abc","signature":"0xdeadbeef"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestLinkWalletHandler_InvalidAddress(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser1", "w1@example.com", "password123")

	body := `{"address":"not-an-address","nonce":"abc","signature":"0xdeadbeef"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	if resp["error"] != "invalid wallet address" {
		t.Errorf("error: want invalid wallet address, got %v", resp["error"])
	}
}

func TestLinkWalletHandler_InvalidNonce(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser2", "w2@example.com", "password123")
	_, addr := newHandlerTestWallet(t)

	body := fmt.Sprintf(`{"address":%q,"nonce":"unknown","signature":"0x%s"}`,
		addr, strings.Repeat("ab", 65))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	if resp["error"] != "invalid or expired nonce" {
		t.Errorf("error: want invalid or expired nonce, got %v", resp["error"])
	}
}

func TestLinkWalletHandler_InvalidSignature(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser4", "w4@example.com", "password123")

	_, addr := newHandlerTestWallet(t)
	nonce := "handler-invalid-signature"
	seedHandlerWalletNonce(t, env, addr, nonce)
	invalidSig := "0x" + strings.Repeat("11", 65)

	body := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce, invalidSig)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	if resp["error"] != "invalid wallet signature" {
		t.Errorf("error: want invalid wallet signature, got %v", resp["error"])
	}
}

func TestLinkWalletHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	accessToken, _ := env.registerUser(t, "walletuser3", "w3@example.com", "password123")

	key, addr := newHandlerTestWallet(t)
	nonce := "handler-success-nonce"
	nr := seedHandlerWalletNonce(t, env, addr, nonce)
	msg := handlerSIWEMessage(addr, nonce, nr.CreatedAt.UTC().Format(time.RFC3339))
	sig := handlerSignSIWE(t, msg, key)

	body := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonce, sig)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w.Body.Bytes())
	data, _ := resp["data"].(map[string]interface{})
	if data["address"] != addr {
		t.Errorf("address: want %s, got %v", addr, data["address"])
	}
}

func TestLinkWalletHandler_WalletAlreadyLinked(t *testing.T) {
	env := newTestEnv(t)
	key, addr := newHandlerTestWallet(t)

	accessA, _ := env.registerUser(t, "walletA", "wa@example.com", "password123")
	nonceA := "handler-conflict-a"
	nrA := seedHandlerWalletNonce(t, env, addr, nonceA)
	msgA := handlerSIWEMessage(addr, nonceA, nrA.CreatedAt.UTC().Format(time.RFC3339))
	bodyA := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonceA, handlerSignSIWE(t, msgA, key))
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(bodyA))
	reqA.Header.Set("Authorization", "Bearer "+accessA)
	reqA.Header.Set("Content-Type", "application/json")
	wA := httptest.NewRecorder()
	env.router.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusOK {
		t.Fatalf("first link want 200, got %d: %s", wA.Code, wA.Body.String())
	}

	accessB, _ := env.registerUser(t, "walletB", "wb@example.com", "password123")
	nonceB := "handler-conflict-b"
	nrB := seedHandlerWalletNonce(t, env, addr, nonceB)
	msgB := handlerSIWEMessage(addr, nonceB, nrB.CreatedAt.UTC().Format(time.RFC3339))
	bodyB := fmt.Sprintf(`{"address":%q,"nonce":%q,"signature":%q}`, addr, nonceB, handlerSignSIWE(t, msgB, key))
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/wallet", bytes.NewBufferString(bodyB))
	reqB.Header.Set("Authorization", "Bearer "+accessB)
	reqB.Header.Set("Content-Type", "application/json")
	wB := httptest.NewRecorder()
	env.router.ServeHTTP(wB, reqB)

	if wB.Code != http.StatusConflict {
		t.Errorf("want 409, got %d: %s", wB.Code, wB.Body.String())
	}
	resp := parseBody(t, wB.Body.Bytes())
	if resp["error"] != "wallet already linked to another account" {
		t.Errorf("error: want wallet already linked to another account, got %v", resp["error"])
	}
}
