package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TxSource identifies the origin of a points transaction.
type TxSource string

const (
	TxSourceBits      TxSource = "bits"
	TxSourceWatchTime TxSource = "watch_time"
	TxSourceClick     TxSource = "click"
	TxSourceSpend     TxSource = "spend"
)

// PointsLedger is the per-channel points balance for a viewer.
// Each viewer has one ledger per channel; balances are not shared across channels.
type PointsLedger struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index"                       json:"user_id"`
	ChannelID        string    `gorm:"type:varchar(255);not null;index"               json:"channel_id"`
	CumulativeTotal  int64     `gorm:"not null;default:0"                             json:"cumulative_total"`
	SpendableBalance int64     `gorm:"not null;default:0"                             json:"spendable_balance"`
	CreatedAt        time.Time `                                                      json:"created_at"`
	UpdatedAt        time.Time `                                                      json:"updated_at"`
}

func (p *PointsLedger) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		p.ID = id
	}
	return nil
}

// PointsTransaction records every change to a viewer's points balance.
//
// WatchSessionID rules by source:
//   - "watch_time" → always non-nil; links to the session that triggered the reward
//   - "bits"       → always nil; no session context
//   - "spend"      → always nil; consumption has no session context
//
// No FK constraint on watch_session_id — sessions may be archived or purged
// independently without orphaning the transaction history.
type PointsTransaction struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	LedgerID       uuid.UUID  `gorm:"type:uuid;not null;index"                       json:"ledger_id"`
	WatchSessionID *uuid.UUID `gorm:"type:uuid;index"                                json:"watch_session_id"`
	Source         TxSource   `gorm:"type:varchar(50);not null"                      json:"source"`
	Delta          int64      `gorm:"not null"                                       json:"delta"`
	BalanceAfter   int64      `gorm:"not null"                                       json:"balance_after"`
	Note           *string    `gorm:"type:text"                                      json:"note"`
	CreatedAt      time.Time  `                                                      json:"created_at"`

	Ledger PointsLedger `gorm:"foreignKey:LedgerID" json:"-"`
}

func (p *PointsTransaction) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		p.ID = id
	}
	return nil
}
