package models

import (
	"time"

	"github.com/google/uuid"
)

// TachiBalance stores the on-chain $TACHI token balance for a user.
// Populated when a user claims spendable_balance from points_ledgers.
type TachiBalance struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"                               json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"                                               json:"user_id"`
	User      *User     `gorm:"constraint:OnDelete:CASCADE"                                                  json:"-"`
	Balance   int64     `gorm:"type:numeric(20,6);not null;default:0;check:chk_tachi_balance_gte_0,balance >= 0" json:"balance"`
	UpdatedAt time.Time `gorm:"not null;default:now()"                                                       json:"updated_at"`
}
