package services

import (
	"errors"
	"testing"
	"time"

	"github.com/tachigo/tachigo/internal/models"
)

func TestRefresh_ExpiredTokenDeletesStoredToken(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, testConfig())
	_, tokens, err := svc.Register(RegisterInput{
		Username: "expired_cleanup_user",
		Email:    "expired-cleanup@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	hash := hashToken(tokens.RefreshToken)
	if err := db.Model(&models.RefreshToken{}).
		Where("token_hash = ?", hash).
		Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatalf("expire refresh token: %v", err)
	}

	_, err = svc.Refresh(tokens.RefreshToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}

	var count int64
	if err := db.Model(&models.RefreshToken{}).Where("token_hash = ?", hash).Count(&count).Error; err != nil {
		t.Fatalf("count refresh token: %v", err)
	}
	if count != 0 {
		t.Fatalf("expired refresh token should be deleted, got %d rows", count)
	}
}
