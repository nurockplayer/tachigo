package models

import (
	"time"

	"github.com/google/uuid"
)

// TachiBalance stores the on-chain $TACHI token balance for a user.
// Populated when a user claims spendable_balance from points_ledgers.
type TachiBalance struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"                 json:"user_id"`
	Balance   float64   `gorm:"type:numeric(20,6);not null;default:0"          json:"balance"`
	UpdatedAt time.Time `                                                      json:"updated_at"`
}
