package handlers_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/models"
)

// sha256hex hashes a string using SHA-256, matching the internal hashToken helper.
func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ─── verify-email/send (authenticated) ───────────────────────────────────────

func TestSendVerificationHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	access, _ := env.registerUser(t, "vuser", "vuser@example.com", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email/send", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendVerificationHandler_AlreadyVerified(t *testing.T) {
	env := newTestEnv(t)
	access, _ := env.registerUser(t, "alreadyv", "alreadyv@example.com", "password123")
	env.db.Model(&models.User{}).Where("email = ?", "alreadyv@example.com").Update("email_verified", true)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email/send", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendVerificationHandler_NoToken_Unauthorized(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email/send", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

// ─── verify-email/confirm (public) ───────────────────────────────────────────

func TestConfirmVerificationHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "confuser", "conf@example.com", "password123")

	var user models.User
	env.db.Where("email = ?", "conf@example.com").First(&user)

	rawToken := "known-test-token-000000000000000000000000000000000000"
	env.db.Create(&models.EmailVerification{
		UserID:    user.ID,
		TokenHash: sha256hex(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	body := fmt.Sprintf(`{"token":"%s"}`, rawToken)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email/confirm",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	env.db.First(&user, "id = ?", user.ID)
	if !user.EmailVerified {
		t.Error("expected email_verified = true after confirmation")
	}
}

func TestConfirmVerificationHandler_InvalidToken(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email/confirm",
		bytes.NewBufferString(`{"token":"bad-token"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestConfirmVerificationHandler_MissingToken(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email/confirm",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── forgot-password ──────────────────────────────────────────────────────────

func TestForgotPasswordHandler_KnownEmail_Returns200(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "forgotuser", "forgot@example.com", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"forgot@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestForgotPasswordHandler_UnknownEmail_StillReturns200(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"nobody@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	// Must not reveal whether email exists
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestForgotPasswordHandler_InvalidEmail(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ─── reset-password ───────────────────────────────────────────────────────────

func TestResetPasswordHandler_Success(t *testing.T) {
	env := newTestEnv(t)
	env.registerUser(t, "resetuser", "reset@example.com", "oldpassword")

	rawToken := "reset-test-token-00000000000000000000000000000000000"
	env.db.Create(&models.PasswordReset{
		Email:     "reset@example.com",
		TokenHash: sha256hex(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	body := fmt.Sprintf(`{"token":"%s","new_password":"newpassword123"}`, rawToken)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResetPasswordHandler_InvalidToken(t *testing.T) {
	env := newTestEnv(t)

	body := `{"token":"no-such-token","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestResetPasswordHandler_ShortPassword(t *testing.T) {
	env := newTestEnv(t)

	body := `{"token":"sometoken","new_password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestResetPasswordHandler_MissingFields(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}
