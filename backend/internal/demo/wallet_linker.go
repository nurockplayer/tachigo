package demo

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrDemoUserSelectorRequired = errors.New("user_id or email is required")
	ErrDemoUserNotFound         = errors.New("demo user not found")
	ErrInvalidWalletAddress     = errors.New("invalid wallet address")
)

type LinkDemoWalletInput struct {
	UserID        string
	Email         string
	WalletAddress string
}

type LinkedDemoWallet struct {
	UserID        uuid.UUID
	WalletAddress string
}

func LinkDemoWallet(ctx context.Context, db *gorm.DB, input LinkDemoWalletInput) (LinkedDemoWallet, error) {
	if !common.IsHexAddress(input.WalletAddress) {
		return LinkedDemoWallet{}, ErrInvalidWalletAddress
	}

	walletAddress := common.HexToAddress(input.WalletAddress).Hex()

	var user models.User
	if err := findDemoUser(ctx, db, input, &user); err != nil {
		return LinkedDemoWallet{}, err
	}

	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var provider models.AuthProvider
		err := tx.Where("user_id = ? AND provider = ?", user.ID, models.ProviderWeb3).
			Order("created_at ASC").
			First(&provider).Error
		if err == nil {
			return tx.Model(&provider).Update("provider_id", walletAddress).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return tx.Create(&models.AuthProvider{
			UserID:     user.ID,
			Provider:   models.ProviderWeb3,
			ProviderID: walletAddress,
		}).Error
	}); err != nil {
		return LinkedDemoWallet{}, err
	}

	return LinkedDemoWallet{UserID: user.ID, WalletAddress: walletAddress}, nil
}

func findDemoUser(ctx context.Context, db *gorm.DB, input LinkDemoWalletInput, user *models.User) error {
	query := db.WithContext(ctx)
	switch {
	case input.UserID != "":
		userID, err := uuid.Parse(input.UserID)
		if err != nil {
			return fmt.Errorf("parse user_id: %w", err)
		}
		query = query.Where("id = ?", userID)
	case input.Email != "":
		query = query.Where("email = ?", input.Email)
	default:
		return ErrDemoUserSelectorRequired
	}

	err := query.First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrDemoUserNotFound
	}
	return err
}
