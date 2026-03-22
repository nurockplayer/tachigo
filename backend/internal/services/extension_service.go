package services

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrInvalidExtJWT      = errors.New("invalid extension JWT")
	ErrInvalidReceipt     = errors.New("invalid transaction receipt")
	ErrExtSecretMissing   = errors.New("TWITCH_EXTENSION_SECRET not configured")
)

// ExtensionClaims are the claims embedded in a Twitch Extension JWT.
type ExtensionClaims struct {
	OpaqueUserID string `json:"opaque_user_id"`
	UserID       string `json:"user_id"`
	ChannelID    string `json:"channel_id"`
	Role         string `json:"role"`
	jwt.RegisteredClaims
}

// ReceiptClaims are the claims embedded in a Bits transaction receipt JWT.
type ReceiptClaims struct {
	Data struct {
		TransactionID string `json:"transactionId"`
		SKU           string `json:"sku"`
		Amount        int    `json:"amount"`
		Type          string `json:"type"` // "bits"
	} `json:"data"`
	jwt.RegisteredClaims
}

type ExtensionService struct {
	db      *gorm.DB
	cfg     *config.Config
	authSvc *AuthService
}

func NewExtensionService(db *gorm.DB, cfg *config.Config, authSvc *AuthService) *ExtensionService {
	return &ExtensionService{db: db, cfg: cfg, authSvc: authSvc}
}

// VerifyExtJWT verifies a Twitch Extension JWT and returns its claims.
func (s *ExtensionService) VerifyExtJWT(tokenStr string) (*ExtensionClaims, error) {
	secret := s.cfg.OAuth.Twitch.ExtensionSecret
	if secret == "" {
		return nil, ErrExtSecretMissing
	}

	rawKey, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("decode extension secret: %w", err)
	}

	claims := &ExtensionClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return rawKey, nil
	})
	if err != nil {
		return nil, ErrInvalidExtJWT
	}
	return claims, nil
}

// VerifyReceiptJWT verifies a Bits transaction receipt JWT.
func (s *ExtensionService) VerifyReceiptJWT(receiptStr string) (*ReceiptClaims, error) {
	secret := s.cfg.OAuth.Twitch.ExtensionSecret
	if secret == "" {
		return nil, ErrExtSecretMissing
	}

	rawKey, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return nil, fmt.Errorf("decode extension secret: %w", err)
	}

	claims := &ReceiptClaims{}
	_, err = jwt.ParseWithClaims(receiptStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return rawKey, nil
	})
	if err != nil {
		return nil, ErrInvalidReceipt
	}
	return claims, nil
}

// LoginWithExtension upserts a user based on their Twitch opaque/user ID and
// issues a tachigo token pair.
func (s *ExtensionService) LoginWithExtension(extJWT string) (*models.User, *TokenPair, error) {
	claims, err := s.VerifyExtJWT(extJWT)
	if err != nil {
		return nil, nil, err
	}

	twitchUserID := claims.UserID
	if twitchUserID == "" {
		twitchUserID = claims.OpaqueUserID
	}

	// Find or create user by Twitch provider link.
	var provider models.AuthProvider
	err = s.db.Where("provider = ? AND provider_id = ?", models.ProviderTwitch, twitchUserID).
		First(&provider).Error

	var user models.User
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// New user — create with a generated username.
		generatedUsername := "twitch_" + twitchUserID
		user = models.User{
			ID:       uuid.New(),
			Username: &generatedUsername,
			Role:     models.RoleViewer,
		}
		if err := s.db.Create(&user).Error; err != nil {
			return nil, nil, err
		}
		provider = models.AuthProvider{
			UserID:     user.ID,
			Provider:   models.ProviderTwitch,
			ProviderID: twitchUserID,
		}
		if err := s.db.Create(&provider).Error; err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	} else {
		if err := s.db.First(&user, provider.UserID).Error; err != nil {
			return nil, nil, err
		}
	}

	tokens, err := s.authSvc.issueTokenPair(&user)
	if err != nil {
		return nil, nil, err
	}
	return &user, tokens, nil
}

// CompleteBitsTransaction verifies the Extension JWT + receipt, then issues a
// tachigo token pair for the viewer.
func (s *ExtensionService) CompleteBitsTransaction(extJWT, receipt, sku string) (*models.User, *TokenPair, error) {
	extClaims, err := s.VerifyExtJWT(extJWT)
	if err != nil {
		return nil, nil, err
	}

	receiptClaims, err := s.VerifyReceiptJWT(receipt)
	if err != nil {
		return nil, nil, err
	}

	// Validate that the SKU in the receipt matches what was requested.
	if receiptClaims.Data.SKU != sku {
		return nil, nil, ErrInvalidReceipt
	}

	// Re-use the login flow to get/create the user, then issue tokens.
	user, tokens, err := s.LoginWithExtension(extJWT)
	if err != nil {
		return nil, nil, err
	}

	_ = extClaims // available for future logging / reward logic

	return user, tokens, nil
}
