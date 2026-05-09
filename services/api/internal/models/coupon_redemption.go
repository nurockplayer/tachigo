package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CouponRedemptionStatus string

const (
	// CouponRedemptionPending means burn succeeded and tx_hash is recorded, but
	// Tachiya redeem or local voucher persistence is still unresolved.
	CouponRedemptionPending CouponRedemptionStatus = "pending"
	// CouponRedemptionRedeemed means Tachiya returned a voucher and it was persisted locally.
	CouponRedemptionRedeemed CouponRedemptionStatus = "redeemed"
	// CouponRedemptionCompensationNeeded means burn succeeded and Tachiya redeem failed.
	CouponRedemptionCompensationNeeded CouponRedemptionStatus = "compensation-needed"
)

type CouponRedemption struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID              `gorm:"type:uuid;not null;index"                       json:"user_id"`
	User         User                   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"  json:"-"`
	CouponID     string                 `gorm:"type:varchar(255);not null"                     json:"coupon_id"`
	Amount       int64                  `gorm:"not null;check:chk_coupon_redemptions_amount_gt_0,amount > 0" json:"amount"`
	TxHash       string                 `gorm:"type:varchar(255);not null"                     json:"tx_hash"`
	Status       CouponRedemptionStatus `gorm:"type:varchar(50);not null;check:chk_coupon_redemptions_status,status IN ('pending','redeemed','compensation-needed')" json:"status"`
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
