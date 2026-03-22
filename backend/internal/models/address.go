package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShippingAddress struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index"                       json:"user_id"`
	RecipientName string         `gorm:"type:varchar(100);not null"                     json:"recipient_name"`
	Phone         *string        `gorm:"type:varchar(20)"                               json:"phone"`
	AddressLine1  string         `gorm:"type:varchar(255);not null"                     json:"address_line1"`
	AddressLine2  *string        `gorm:"type:varchar(255)"                              json:"address_line2"`
	City          string         `gorm:"type:varchar(100);not null"                     json:"city"`
	District      *string        `gorm:"type:varchar(100)"                              json:"district"`
	PostalCode    *string        `gorm:"type:varchar(20)"                               json:"postal_code"`
	Country       string         `gorm:"type:varchar(50);default:'TW'"                  json:"country"`
	IsDefault     bool           `gorm:"default:false"                                  json:"is_default"`
	CreatedAt     time.Time      `                                                      json:"created_at"`
	UpdatedAt     time.Time      `                                                      json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index"                                          json:"-"`
}

func (a *ShippingAddress) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
