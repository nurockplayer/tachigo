package services

import (
	"errors"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var ErrInvalidWalletAddress = errors.New("invalid wallet address")

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type UpdateProfileInput struct {
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

type LinkWalletInput struct {
	Address   string `json:"address" binding:"required"`
	Nonce     string `json:"nonce" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

func (s *UserService) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (s *UserService) UpdateProfile(id uuid.UUID, input UpdateProfileInput) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if input.Username != nil {
		// Uniqueness check
		var count int64
		s.db.Model(&models.User{}).
			Where("username = ? AND id != ?", *input.Username, id).
			Count(&count)
		if count > 0 {
			return nil, ErrUsernameExists
		}
		user.Username = input.Username
	}

	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) ListProviders(userID uuid.UUID) ([]models.AuthProvider, error) {
	var providers []models.AuthProvider
	if err := s.db.Where("user_id = ?", userID).Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *UserService) LinkWallet(userID uuid.UUID, input LinkWalletInput) (string, error) {
	if !common.IsHexAddress(input.Address) {
		return "", ErrInvalidWalletAddress
	}

	checksumAddr := common.HexToAddress(input.Address).Hex()
	lookupAddr := strings.ToLower(checksumAddr)

	var nonceRecord models.Web3Nonce
	if err := s.db.Where("nonce = ? AND address = ?", input.Nonce, lookupAddr).
		First(&nonceRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrInvalidNonce
		}
		return "", err
	}
	if nonceRecord.IsExpired() {
		return "", ErrInvalidNonce
	}

	issuedAt := nonceRecord.CreatedAt.UTC().Format(time.RFC3339)
	msg := siweMessage(checksumAddr, input.Nonce, issuedAt)
	if !verifyEthSignature(msg, input.Signature, checksumAddr) {
		return "", ErrInvalidSignature
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("nonce = ? AND address = ?", input.Nonce, lookupAddr).
			Delete(&models.Web3Nonce{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidNonce
		}

		var count int64
		if err := tx.Model(&models.AuthProvider{}).
			Where("provider = ? AND provider_id = ? AND deleted_at IS NULL AND user_id != ?",
				models.ProviderWeb3, checksumAddr, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrProviderLinked
		}

		now := time.Now()
		if err := tx.Model(&models.AuthProvider{}).
			Where("user_id = ? AND provider = ? AND deleted_at IS NULL", userID, models.ProviderWeb3).
			Update("deleted_at", now).Error; err != nil {
			return err
		}

		var ap models.AuthProvider
		findErr := tx.Unscoped().
			Where("user_id = ? AND provider = ? AND provider_id = ?", userID, models.ProviderWeb3, checksumAddr).
			First(&ap).Error

		if findErr == nil {
			if err := tx.Unscoped().Model(&ap).
				UpdateColumn("deleted_at", gorm.Expr("NULL")).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return ErrProviderLinked
				}
				return err
			}
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		newAP := models.AuthProvider{
			UserID:     userID,
			Provider:   models.ProviderWeb3,
			ProviderID: checksumAddr,
		}
		if err := tx.Create(&newAP).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrProviderLinked
			}
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return checksumAddr, nil
}
