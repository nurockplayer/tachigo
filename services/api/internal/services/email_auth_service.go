package services

import (
	"context"
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
	ErrAlreadyVerified        = errors.New("email already verified")
	ErrInvalidVerifyToken     = errors.New("invalid or expired verification token")
	ErrInvalidResetToken      = errors.New("invalid or expired password reset token")
	ErrPasswordResetEmailSend = errors.New("failed to send password reset email")
)

const (
	verifyTokenTTL      = 24 * time.Hour
	resetTokenTTL       = 1 * time.Hour
	resetCleanupTimeout = 5 * time.Second
	// passwordResetCooldown throttles how often a reset email may be sent to the
	// same address, to prevent inbox bombing / email-cost abuse via the public
	// /auth/forgot-password endpoint.
	passwordResetCooldown = 5 * time.Minute
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
func (s *EmailAuthService) SendVerificationEmail(ctx context.Context, userID uuid.UUID) error {
	db := s.db.WithContext(ctx)

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
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
	if err := db.Where("user_id = ?", userID).Delete(&models.EmailVerification{}).Error; err != nil {
		return err
	}

	if err := db.Create(&models.EmailVerification{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(verifyTokenTTL),
	}).Error; err != nil {
		return err
	}

	link := fmt.Sprintf("%s/verify-email?token=%s", s.cfg.App.FrontendURL, rawToken)
	body := verificationEmailBody(link)
	return s.mailer.Send(ctx, *user.Email, "Verify your Tachigo email", body)
}

// VerifyEmail marks the user's email as verified using a raw token.
func (s *EmailAuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	db := s.db.WithContext(ctx)
	hash := hashToken(rawToken)

	var record models.EmailVerification
	if err := db.Where("token_hash = ?", hash).First(&record).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return ErrInvalidVerifyToken
	}
	if record.IsExpired() {
		if err := db.Delete(&record).Error; err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return err
		}
		return ErrInvalidVerifyToken
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", record.UserID).
			Update("email_verified", true).Error; err != nil {
			return err
		}
		if err := tx.Delete(&record).Error; err != nil {
			return err
		}
		return nil
	})
}

// ─── Password Reset ───────────────────────────────────────────────────────────

// ForgotPassword sends a password reset link to the given email.
// Returns nil even when the email is not found to avoid user enumeration.
func (s *EmailAuthService) ForgotPassword(ctx context.Context, email string) error {
	db := s.db.WithContext(ctx)

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Do not reveal whether the email exists (anti-enumeration)
			return nil
		}
		return err
	}

	// Per-email throttle: if a reset email was already issued within the cooldown
	// window, silently skip re-sending. Returning nil — indistinguishable from the
	// success and not-found paths — keeps the endpoint from leaking whether the
	// address exists or has a pending reset, while preventing inbox bombing.
	var lastReset models.PasswordReset
	switch err := db.Where("email = ?", email).Order("created_at DESC").First(&lastReset).Error; {
	case err == nil:
		if time.Since(lastReset.CreatedAt) < passwordResetCooldown {
			return nil
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		// No prior reset token for this address; proceed to issue one.
	default:
		return err
	}

	rawToken, err := generateNonce()
	if err != nil {
		return err
	}

	reset := &models.PasswordReset{
		Email:     email,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}
	if err := db.Create(reset).Error; err != nil {
		return err
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.App.FrontendURL, rawToken)
	body := passwordResetEmailBody(link)
	if err := s.mailer.Send(ctx, email, "Reset your Tachigo password", body); err != nil {
		// The cooldown is based on persisted reset rows. If delivery fails after the
		// row is created, clear that unsent attempt so an immediate retry is not
		// silently throttled into a temporary lockout window.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resetCleanupTimeout)
		defer cancel()
		cleanupDB := s.db.WithContext(cleanupCtx)
		if deleteErr := cleanupDB.Delete(reset).Error; deleteErr != nil {
			return fmt.Errorf("%w: %w; failed to clear unsent reset token: %v", ErrPasswordResetEmailSend, err, deleteErr)
		}
		return fmt.Errorf("%w: %w", ErrPasswordResetEmailSend, err)
	}

	if err := db.Where("email = ? AND id <> ?", email, reset.ID).Delete(&models.PasswordReset{}).Error; err != nil {
		return err
	}
	return nil
}

// ResetPassword validates the token and updates the user's password.
func (s *EmailAuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	db := s.db.WithContext(ctx)
	hash := hashToken(rawToken)

	var record models.PasswordReset
	if err := db.Where("token_hash = ?", hash).First(&record).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return ErrInvalidResetToken
	}
	if record.IsExpired() {
		if err := db.Delete(&record).Error; err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return err
		}
		return ErrInvalidResetToken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("email = ?", record.Email).
			Update("password_hash", string(hashed)).Error; err != nil {
			return err
		}
		if err := tx.Delete(&record).Error; err != nil {
			return err
		}
		return nil
	})
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
