package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrAlreadyVerified   = errors.New("email already verified")
	ErrInvalidVerifyToken = errors.New("invalid or expired verification token")
	ErrInvalidResetToken  = errors.New("invalid or expired password reset token")
)

const (
	verifyTokenTTL = 24 * time.Hour
	resetTokenTTL  = 1 * time.Hour
)

type EmailAuthService struct {
	db     *gorm.DB
	cfg    *config.Config
	mailer Mailer
}

func NewEmailAuthService(db *gorm.DB, cfg *config.Config, mailer Mailer) *EmailAuthService {
	return &EmailAuthService{db: db, cfg: cfg, mailer: mailer}
}

// ─── Email Verification ───────────────────────────────────────────────────────

// SendVerificationEmail generates a token and emails a verification link to the user.
func (s *EmailAuthService) SendVerificationEmail(userID uuid.UUID) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return ErrUserNotFound
	}
	if user.EmailVerified {
		return ErrAlreadyVerified
	}
	if user.Email == nil {
		return errors.New("user has no email address")
	}

	rawToken, err := generateNonce()
	if err != nil {
		return err
	}

	// Replace any existing verification token for this user
	s.db.Where("user_id = ?", userID).Delete(&models.EmailVerification{})

	if err := s.db.Create(&models.EmailVerification{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(verifyTokenTTL),
	}).Error; err != nil {
		return err
	}

	link := fmt.Sprintf("%s/verify-email?token=%s", s.cfg.App.FrontendURL, rawToken)
	body := verificationEmailBody(link)
	return s.mailer.Send(*user.Email, "Verify your Tachigo email", body)
}

// VerifyEmail marks the user's email as verified using a raw token.
func (s *EmailAuthService) VerifyEmail(rawToken string) error {
	hash := hashToken(rawToken)

	var record models.EmailVerification
	if err := s.db.Where("token_hash = ?", hash).First(&record).Error; err != nil {
		return ErrInvalidVerifyToken
	}
	if record.IsExpired() {
		s.db.Delete(&record)
		return ErrInvalidVerifyToken
	}

	if err := s.db.Model(&models.User{}).Where("id = ?", record.UserID).
		Update("email_verified", true).Error; err != nil {
		return err
	}

	s.db.Delete(&record)
	return nil
}

// ─── Password Reset ───────────────────────────────────────────────────────────

// ForgotPassword sends a password reset link to the given email.
// Returns nil even when the email is not found to avoid user enumeration.
func (s *EmailAuthService) ForgotPassword(email string) error {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		// Do not reveal whether the email exists
		return nil
	}

	rawToken, err := generateNonce()
	if err != nil {
		return err
	}

	// Replace any existing reset token for this email
	s.db.Where("email = ?", email).Delete(&models.PasswordReset{})

	if err := s.db.Create(&models.PasswordReset{
		Email:     email,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}).Error; err != nil {
		return err
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.App.FrontendURL, rawToken)
	body := passwordResetEmailBody(link)
	return s.mailer.Send(email, "Reset your Tachigo password", body)
}

// ResetPassword validates the token and updates the user's password.
func (s *EmailAuthService) ResetPassword(rawToken, newPassword string) error {
	hash := hashToken(rawToken)

	var record models.PasswordReset
	if err := s.db.Where("token_hash = ?", hash).First(&record).Error; err != nil {
		return ErrInvalidResetToken
	}
	if record.IsExpired() {
		s.db.Delete(&record)
		return ErrInvalidResetToken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.db.Model(&models.User{}).Where("email = ?", record.Email).
		Update("password_hash", string(hashed)).Error; err != nil {
		return err
	}

	s.db.Delete(&record)
	return nil
}

// ─── Email templates ──────────────────────────────────────────────────────────

func verificationEmailBody(link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;color:#333;max-width:480px;margin:auto;padding:24px">
  <h2>Verify your email</h2>
  <p>Thanks for signing up for Tachigo. Click the button below to verify your email address.</p>
  <p>The link expires in <strong>24 hours</strong>.</p>
  <p style="margin:32px 0">
    <a href="%s" style="background:#6441a5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold">
      Verify Email
    </a>
  </p>
  <p style="font-size:12px;color:#999">If you didn't create an account, you can ignore this email.</p>
</body>
</html>`, link)
}

func passwordResetEmailBody(link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;color:#333;max-width:480px;margin:auto;padding:24px">
  <h2>Reset your password</h2>
  <p>We received a request to reset your Tachigo password. Click the button below to choose a new one.</p>
  <p>The link expires in <strong>1 hour</strong>.</p>
  <p style="margin:32px 0">
    <a href="%s" style="background:#6441a5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold">
      Reset Password
    </a>
  </p>
  <p style="font-size:12px;color:#999">If you didn't request a password reset, you can ignore this email. Your password will not be changed.</p>
</body>
</html>`, link)
}
