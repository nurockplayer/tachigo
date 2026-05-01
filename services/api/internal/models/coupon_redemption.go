package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CouponRedemptionStatus string

const (
	CouponRedemptionPending            CouponRedemptionStatus = "pending"
	CouponRedemptionRedeemed           CouponRedemptionStatus = "redeemed"
	CouponRedemptionCompensationNeeded CouponRedemptionStatus = "compensation-needed"
)

type CouponRedemption struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID              `gorm:"type:uuid;not null;index"                       json:"user_id"`
	CouponID     string                 `gorm:"type:varchar(255);not null"                     json:"coupon_id"`
	Amount       int64                  `gorm:"not null"                                       json:"amount"`
	TxHash       string                 `gorm:"type:varchar(255);not null"                     json:"tx_hash"`
	Status       CouponRedemptionStatus `gorm:"type:varchar(50);not null"                      json:"status"`
	VoucherCode  *string                `gorm:"type:varchar(255)"                              json:"voucher_code,omitempty"`
	ErrorMessage *string                `gorm:"type:text"                                      json:"error_message,omitempty"`
	CreatedAt    time.Time              `                                                      json:"created_at"`
	UpdatedAt    time.Time              `                                                      json:"updated_at"`
}

func (c *CouponRedemption) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		c.ID = id
	}
	return nil
}
