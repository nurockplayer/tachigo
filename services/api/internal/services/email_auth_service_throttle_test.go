package services

import (
	"context"
	"errors"
	"testing"

	"github.com/tachigo/tachigo/internal/models"
)

type failOnceResetMailer struct {
	failNext bool
	sent     []struct{ to, subject, body string }
}

func (m *failOnceResetMailer) Send(_ context.Context, to, subject, body string) error {
	if m.failNext {
		m.failNext = false
		return errors.New("forced password reset send failure")
	}
	m.sent = append(m.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

func TestForgotPassword_SendFailureDoesNotThrottleRetry(t *testing.T) {
	db := newTestDB(t)
	cfg := testConfig()
	cfg.App.FrontendURL = "http://localhost:3000"
	mailer := &failOnceResetMailer{failNext: true}
	svc := NewEmailAuthService(db, cfg, mailer)
	email := "retry-after-send-failure@example.com"
	seedEmailUser(t, svc, email, true)

	err := svc.ForgotPassword(context.Background(), email)
	if !errors.Is(err, ErrPasswordResetEmailSend) {
		t.Fatalf("want ErrPasswordResetEmailSend, got %v", err)
	}

	var count int64
	svc.db.Model(&models.PasswordReset{}).Where("email = ?", email).Count(&count)
	if count != 0 {
		t.Fatalf("failed send should not leave a throttling reset token, got %d", count)
	}

	if err := svc.ForgotPassword(context.Background(), email); err != nil {
		t.Fatalf("retry after failed send should not be throttled: %v", err)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("retry should send exactly one reset email, got %d", len(mailer.sent))
	}
	svc.db.Model(&models.PasswordReset{}).Where("email = ?", email).Count(&count)
	if count != 1 {
		t.Fatalf("successful retry should persist one reset token, got %d", count)
	}
}
