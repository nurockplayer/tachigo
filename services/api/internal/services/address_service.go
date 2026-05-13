package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var ErrAddressNotFound = errors.New("address not found")

type AddressService struct {
	db *gorm.DB
}

func NewAddressService(db *gorm.DB) *AddressService {
	return &AddressService{db: db}
}

func (s *AddressService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.db.WithContext(ctx)
}

func addressContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

type AddressInput struct {
	RecipientName string  `json:"recipient_name" binding:"required"`
	Phone         *string `json:"phone"`
	AddressLine1  string  `json:"address_line1"  binding:"required"`
	AddressLine2  *string `json:"address_line2"`
	City          string  `json:"city"           binding:"required"`
	District      *string `json:"district"`
	PostalCode    *string `json:"postal_code"`
	Country       string  `json:"country"`
	IsDefault     bool    `json:"is_default"`
}

func (s *AddressService) List(userID uuid.UUID) ([]models.ShippingAddress, error) {
	return s.ListContext(context.Background(), userID)
}

func (s *AddressService) ListContext(ctx context.Context, userID uuid.UUID) ([]models.ShippingAddress, error) {
	var addrs []models.ShippingAddress
	err := s.dbWithContext(ctx).Where("user_id = ?", userID).Order("is_default DESC, created_at ASC").Find(&addrs).Error
	return addrs, err
}

func (s *AddressService) Create(userID uuid.UUID, input AddressInput) (*models.ShippingAddress, error) {
	return s.CreateContext(context.Background(), userID, input)
}

func (s *AddressService) CreateContext(ctx context.Context, userID uuid.UUID, input AddressInput) (*models.ShippingAddress, error) {
	country := input.Country
	if country == "" {
		country = "TW"
	}

	addr := &models.ShippingAddress{
		UserID:        userID,
		RecipientName: input.RecipientName,
		Phone:         input.Phone,
		AddressLine1:  input.AddressLine1,
		AddressLine2:  input.AddressLine2,
		City:          input.City,
		District:      input.District,
		PostalCode:    input.PostalCode,
		Country:       country,
		IsDefault:     input.IsDefault,
	}

	if err := s.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.IsDefault {
			if err := tx.Model(&models.ShippingAddress{}).
				Where("user_id = ?", userID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		return tx.Create(addr).Error
	}); err != nil {
		return nil, err
	}
	return addr, nil
}

func (s *AddressService) Update(userID, addrID uuid.UUID, input AddressInput) (*models.ShippingAddress, error) {
	return s.UpdateContext(context.Background(), userID, addrID, input)
}

func (s *AddressService) UpdateContext(ctx context.Context, userID, addrID uuid.UUID, input AddressInput) (*models.ShippingAddress, error) {
	db := s.dbWithContext(ctx)
	var addr models.ShippingAddress
	if err := db.Where("id = ? AND user_id = ?", addrID, userID).First(&addr).Error; err != nil {
		if ctxErr := addressContextError(ctx, err); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, ErrAddressNotFound
	}
	wasDefault := addr.IsDefault

	addr.RecipientName = input.RecipientName
	addr.Phone = input.Phone
	addr.AddressLine1 = input.AddressLine1
	addr.AddressLine2 = input.AddressLine2
	addr.City = input.City
	addr.District = input.District
	addr.PostalCode = input.PostalCode
	if input.Country != "" {
		addr.Country = input.Country
	}
	addr.IsDefault = input.IsDefault

	if err := db.Transaction(func(tx *gorm.DB) error {
		if input.IsDefault && !wasDefault {
			if err := tx.Model(&models.ShippingAddress{}).
				Where("user_id = ? AND id != ?", userID, addrID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		return tx.Save(&addr).Error
	}); err != nil {
		return nil, err
	}
	return &addr, nil
}

func (s *AddressService) Delete(userID, addrID uuid.UUID) error {
	return s.DeleteContext(context.Background(), userID, addrID)
}

func (s *AddressService) DeleteContext(ctx context.Context, userID, addrID uuid.UUID) error {
	result := s.dbWithContext(ctx).Where("id = ? AND user_id = ?", addrID, userID).Delete(&models.ShippingAddress{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAddressNotFound
	}
	return nil
}

func (s *AddressService) SetDefault(userID, addrID uuid.UUID) (*models.ShippingAddress, error) {
	return s.SetDefaultContext(context.Background(), userID, addrID)
}

func (s *AddressService) SetDefaultContext(ctx context.Context, userID, addrID uuid.UUID) (*models.ShippingAddress, error) {
	db := s.dbWithContext(ctx)
	var addr models.ShippingAddress
	if err := db.Where("id = ? AND user_id = ?", addrID, userID).First(&addr).Error; err != nil {
		if ctxErr := addressContextError(ctx, err); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, ErrAddressNotFound
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ShippingAddress{}).
			Where("user_id = ? AND id != ?", userID, addrID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		addr.IsDefault = true
		return tx.Save(&addr).Error
	}); err != nil {
		return nil, err
	}
	return &addr, nil
}
