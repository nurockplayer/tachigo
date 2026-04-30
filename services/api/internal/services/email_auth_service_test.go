package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
)

// mockMailer records sent emails for assertion in tests.
type mockMailer struct {
	sent []struct{ to, subject, body string }
}

func (m *mockMailer) Send(to, subject, body string) error {
	m.sent = append(m.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

func (m *mockMailer) lastTo() string {
	if len(m.sent) == 0 {
		return ""
	}
	return m.sent[len(m.sent)-1].to
}

func newEmailAuthSvc(t *testing.T) (*EmailAuthService, *mockMailer) {
	t.Helper()
	db := newTestDB(t)
	cfg := testConfig()
	cfg.App.FrontendURL = "http://localhost:3000"
	mailer := &mockMailer{}
	return NewEmailAuthService(db, cfg, mailer), mailer
}

// seedVerifiedUser creates a user with a verified email.
func seedEmailUser(t *testing.T, svc *EmailAuthService, email string, verified bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	svc.db.Create(&models.User{ID: id, Email: &email, Role: models.RoleViewer, EmailVerified: verified})
	return id
}

// ─── SendVerificationEmail ────────────────────────────────────────────────────

func TestSendVerificationEmail_Success(t *testing.T) {
	svc, mailer := newEmailAuthSvc(t)
	email := "user@example.com"
	userID := seedEmailUser(t, svc, email, false)

	if err := svc.SendVerificationEmail(userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mailer.lastTo() != email {
		t.Errorf("email sent to %q, want %q", mailer.lastTo(), email)
	}
}

func TestSendVerificationEmail_AlreadyVerified(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	userID := seedEmailUser(t, svc, "verified@example.com", true)

	err := svc.SendVerificationEmail(userID)
	if err != ErrAlreadyVerified {
		t.Errorf("want ErrAlreadyVerified, got %v", err)
	}
}

func TestSendVerificationEmail_UserNotFound(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)

	err := svc.SendVerificationEmail(uuid.New())
	if err != ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

func TestSendVerificationEmail_ReplacesExistingToken(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	userID := seedEmailUser(t, svc, "resend@example.com", false)

	svc.SendVerificationEmail(userID)
	svc.SendVerificationEmail(userID)

	var count int64
	svc.db.Model(&models.EmailVerification{}).Where("user_id = ?", userID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 verification token, got %d", count)
	}
}

// ─── VerifyEmail ──────────────────────────────────────────────────────────────

func TestVerifyEmail_Success(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	userID := seedEmailUser(t, svc, "toverify@example.com", false)

	// Generate and store a real token
	rawToken, _ := generateNonce()
	svc.db.Create(&models.EmailVerification{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	if err := svc.VerifyEmail(rawToken); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var user models.User
	svc.db.First(&user, "id = ?", userID)
	if !user.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}

	// Token should be consumed
	var count int64
	svc.db.Model(&models.EmailVerification{}).Where("user_id = ?", userID).Count(&count)
	if count != 0 {
		t.Errorf("expected token to be deleted, got %d records", count)
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)

	err := svc.VerifyEmail("no-such-token")
	if err != ErrInvalidVerifyToken {
		t.Errorf("want ErrInvalidVerifyToken, got %v", err)
	}
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	userID := seedEmailUser(t, svc, "expired@example.com", false)

	rawToken, _ := generateNonce()
	svc.db.Create(&models.EmailVerification{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(-time.Minute), // already expired
	})

	err := svc.VerifyEmail(rawToken)
	if err != ErrInvalidVerifyToken {
		t.Errorf("want ErrInvalidVerifyToken, got %v", err)
	}
}

func TestVerifyEmail_RoundTrip(t *testing.T) {
	svc, mailer := newEmailAuthSvc(t)
	email := "roundtrip@example.com"
	userID := seedEmailUser(t, svc, email, false)

	// Send → extract token from DB (simulate clicking link) → confirm
	if err := svc.SendVerificationEmail(userID); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(mailer.sent) == 0 {
		t.Fatal("no email was sent")
	}

	var record models.EmailVerification
	svc.db.Where("user_id = ?", userID).First(&record)

	// We can't get the raw token back from the hash, so use the service flow
	// end-to-end: generate known token, store, verify
	rawToken, _ := generateNonce()
	svc.db.Where("user_id = ?", userID).Delete(&models.EmailVerification{})
	svc.db.Create(&models.EmailVerification{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	if err := svc.VerifyEmail(rawToken); err != nil {
		t.Fatalf("verify: %v", err)
	}

	var user models.User
	svc.db.First(&user, "id = ?", userID)
	if !user.EmailVerified {
		t.Error("expected user to be verified")
	}
}

// ─── ForgotPassword ───────────────────────────────────────────────────────────

func TestForgotPassword_KnownEmail_SendsEmail(t *testing.T) {
	svc, mailer := newEmailAuthSvc(t)
	email := "forgot@example.com"
	seedEmailUser(t, svc, email, true)

	if err := svc.ForgotPassword(email); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mailer.lastTo() != email {
		t.Errorf("email sent to %q, want %q", mailer.lastTo(), email)
	}
}

func TestForgotPassword_UnknownEmail_NoError(t *testing.T) {
	svc, mailer := newEmailAuthSvc(t)

	// Should return nil even for unknown email (no enumeration)
	if err := svc.ForgotPassword("nobody@example.com"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mailer.sent) != 0 {
		t.Error("no email should be sent for unknown address")
	}
}

func TestForgotPassword_ReplacesExistingToken(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	email := "resend_reset@example.com"
	seedEmailUser(t, svc, email, true)

	svc.ForgotPassword(email)
	svc.ForgotPassword(email)

	var count int64
	svc.db.Model(&models.PasswordReset{}).Where("email = ?", email).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 reset token, got %d", count)
	}
}

// ─── ResetPassword ────────────────────────────────────────────────────────────

func TestResetPassword_Success(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	email := "reset@example.com"
	seedEmailUser(t, svc, email, true)

	rawToken, _ := generateNonce()
	svc.db.Create(&models.PasswordReset{
		Email:     email,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	if err := svc.ResetPassword(rawToken, "newpassword123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token should be consumed
	var count int64
	svc.db.Model(&models.PasswordReset{}).Where("email = ?", email).Count(&count)
	if count != 0 {
		t.Error("reset token should be deleted after use")
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)

	err := svc.ResetPassword("bad-token", "newpassword123")
	if err != ErrInvalidResetToken {
		t.Errorf("want ErrInvalidResetToken, got %v", err)
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	svc, _ := newEmailAuthSvc(t)
	email := "expreset@example.com"
	seedEmailUser(t, svc, email, true)

	rawToken, _ := generateNonce()
	svc.db.Create(&models.PasswordReset{
		Email:     email,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(-time.Minute),
	})

	err := svc.ResetPassword(rawToken, "newpassword123")
	if err != ErrInvalidResetToken {
		t.Errorf("want ErrInvalidResetToken, got %v", err)
	}
}

func TestResetPassword_AllowsLoginWithNewPassword(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	cfg.App.FrontendURL = "http://localhost:3000"
	mailer := &mockMailer{}
	emailSvc := NewEmailAuthService(db, cfg, mailer)
	authSvc := NewAuthService(db, cfg)

	// Register user
	user, _, err := authSvc.Register(RegisterInput{
		Username: "resetme",
		Email:    "resetme@example.com",
		Password: "oldpassword",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Issue reset token
	rawToken, _ := generateNonce()
	db.Create(&models.PasswordReset{
		Email:     *user.Email,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(time.Hour),
	})

	// Reset password
	if err := emailSvc.ResetPassword(rawToken, "newpassword123"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	// Old password should fail
	if _, _, err := authSvc.Login(LoginInput{Email: *user.Email, Password: "oldpassword"}); err != ErrInvalidCredentials {
		t.Errorf("old password should fail, got %v", err)
	}

	// New password should work
	if _, _, err := authSvc.Login(LoginInput{Email: *user.Email, Password: "newpassword123"}); err != nil {
		t.Errorf("new password should work, got %v", err)
	}
}
