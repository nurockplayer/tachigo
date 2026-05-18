package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrInvalidExtJWT        = errors.New("invalid extension JWT")
	ErrInvalidReceipt       = errors.New("invalid transaction receipt")
	ErrExtSecretMissing     = errors.New("TWITCH_EXTENSION_SECRET not configured")
	ErrDuplicateTransaction = errors.New("transaction already processed")
	ErrInvalidReceiptAmount = errors.New("receipt amount must be greater than zero")
	ErrInvalidReceiptType   = errors.New("receipt type must be bits")
	// ErrUserNotFound is defined in auth_service.go (same package).
)

// ExtensionClaims are the claims embedded in a Twitch Extension JWT.
type ExtensionClaims struct {
	ExtensionScopedUserID string `json:"opaque_user_id"`
	UserID                string `json:"user_id"`
	ChannelID             string `json:"channel_id"`
	Role                  string `json:"role"`
	jwt.RegisteredClaims
}

// ReceiptClaims are the claims embedded in a Bits transaction receipt JWT.
type ReceiptClaims struct {
	Data struct {
		TransactionID string `json:"transactionId"`
		UserID        string `json:"userId"`
		SKU           string `json:"sku"`
		Amount        int    `json:"amount"`
		Type          string `json:"type"` // "bits" — Twitch SDK contract, do not rename
	} `json:"data"`
	jwt.RegisteredClaims
}

type ExtensionService struct {
	db        *gorm.DB
	cfg       *config.Config
	authSvc   *AuthService
	pointsSvc *PointsService
}

func NewExtensionService(db *gorm.DB, cfg *config.Config, authSvc *AuthService, pointsSvc *PointsService) *ExtensionService {
	return &ExtensionService{db: db, cfg: cfg, authSvc: authSvc, pointsSvc: pointsSvc}
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

// VerifyReceiptJWT verifies a Twitch transaction receipt JWT.
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

// lookupExtensionUser resolves a Twitch identity to an existing tachigo User.
// It does not issue tokens; call issueTokenPair separately.
// Returns ErrInvalidExtJWT if UserID is empty, ErrUserNotFound if no account is linked.
func (s *ExtensionService) lookupExtensionUser(claims *ExtensionClaims) (*models.User, error) {
	if claims.UserID == "" {
		return nil, ErrInvalidExtJWT
	}
	var provider models.AuthProvider
	err := s.db.Where("provider = ? AND provider_id = ?", models.ProviderTwitch, claims.UserID).
		First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := s.db.First(&user, provider.UserID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// findOrCreateExtensionUser finds or creates a tachigo account for the Twitch identity in claims.
// New users receive username "twitch_<twitchUserID>" and role viewer.
// On concurrent unique-constraint violation the winner's record is returned (conflict recovery).
func (s *ExtensionService) findOrCreateExtensionUser(claims *ExtensionClaims) (*models.User, error) {
	if claims.UserID == "" {
		return nil, ErrInvalidExtJWT
	}

	user, err := s.findOrCreateExtensionUserTx(claims)
	if err == nil {
		return user, nil
	}
	if !isDuplicatedKeyError(err) {
		return nil, err
	}
	// conflict recovery: concurrent goroutine won the INSERT race; look up the winning record
	return s.lookupExtensionUser(claims)
}

func (s *ExtensionService) findOrCreateExtensionUserTx(claims *ExtensionClaims) (*models.User, error) {
	var result *models.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var ap models.AuthProvider
		if err := tx.Where("provider = ? AND provider_id = ?", models.ProviderTwitch, claims.UserID).
			First(&ap).Error; err == nil {
			var u models.User
			if err := tx.First(&u, ap.UserID).Error; err != nil {
				return err
			}
			result = &u
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		username := "twitch_" + claims.UserID
		newUser := models.User{
			Username: &username,
			Role:     models.RoleViewer,
		}
		if err := tx.Create(&newUser).Error; err != nil {
			return err
		}

		newAP := models.AuthProvider{
			UserID:     newUser.ID,
			Provider:   models.ProviderTwitch,
			ProviderID: claims.UserID,
		}
		if err := tx.Create(&newAP).Error; err != nil {
			return err
		}

		result = &newUser
		return nil
	})
	return result, err
}

// LoginWithExtension verifies the Extension JWT and issues a tachigo token pair.
// If no tachigo account is linked to the Twitch identity yet, one is created automatically
// (username: "twitch_<twitchUserID>"). Concurrent first-logins are safe: at most one account
// is created and all callers receive a token pair for the same account.
//
// Returns ErrInvalidExtJWT if the JWT is invalid or the viewer has not shared their identity.
func (s *ExtensionService) LoginWithExtension(extJWT string) (*models.User, *TokenPair, error) {
	claims, err := s.VerifyExtJWT(extJWT)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.findOrCreateExtensionUser(claims)
	if err != nil {
		return nil, nil, err
	}

	tokens, err := s.authSvc.issueTokenPair(user)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

// CompleteTPointTransaction verifies the Extension JWT + receipt, then issues a
// tachigo token pair for the already-linked viewer.
func (s *ExtensionService) CompleteTPointTransaction(extJWT, receipt, sku string) (*models.User, *TokenPair, error) {
	extClaims, err := s.VerifyExtJWT(extJWT)
	if err != nil {
		return nil, nil, err
	}

	receiptClaims, err := s.VerifyReceiptJWT(receipt)
	if err != nil {
		return nil, nil, err
	}

	if receiptClaims.Data.SKU != sku {
		return nil, nil, ErrInvalidReceipt
	}
	if receiptClaims.Data.UserID == "" || receiptClaims.Data.UserID != extClaims.UserID {
		return nil, nil, ErrInvalidReceipt
	}
	if receiptClaims.Data.Type != "bits" {
		return nil, nil, ErrInvalidReceiptType
	}
	if receiptClaims.Data.Amount <= 0 {
		return nil, nil, ErrInvalidReceiptAmount
	}
	if receiptClaims.Data.TransactionID == "" {
		return nil, nil, ErrInvalidReceipt
	}
	if len([]rune(sku)) > 255 || len([]rune(receiptClaims.Data.TransactionID)) > 255 {
		return nil, nil, ErrInvalidReceipt
	}

	// Resolve user before touching any write path.
	user, err := s.lookupExtensionUser(extClaims)
	if err != nil {
		return nil, nil, err
	}

	// Write points first; tokens are issued only on success to avoid orphan
	// refresh token records when the points write fails.
	txID := receiptClaims.Data.TransactionID
	err = s.pointsSvc.AddPointsWithMeta(
		user.ID,
		extClaims.ChannelID,
		models.TxSourceTPoint,
		int64(receiptClaims.Data.Amount),
		PointsCreditMeta{
			SKU:                   &sku,
			ExternalTransactionID: &txID,
		},
	)
	if err != nil {
		if isDuplicateExternalTransactionError(err) {
			// ErrDuplicateTransaction also covers the retry case where points were
			// credited in a prior call but token issuance failed. The client should
			// call LoginWithExtension separately to obtain a token.
			return nil, nil, ErrDuplicateTransaction
		}
		return nil, nil, err
	}

	// Points are now committed. If issueTokenPair fails here, points remain credited
	// and the client will receive an error. On retry, AddPointsWithMeta returns
	// ErrDuplicateTransaction — the client should call LoginWithExtension to get tokens.
	tokens, err := s.authSvc.issueTokenPair(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func isDuplicateExternalTransactionError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" &&
			pgErr.ConstraintName == "idx_points_transactions_external_transaction_id"
	}

	errText := err.Error()
	return strings.Contains(errText, "UNIQUE constraint failed") &&
		strings.Contains(errText, "points_transactions.external_transaction_id")
}
