package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var uuidV7Func = uuid.NewV7

type ClaimStatus string

const (
	ClaimStatusPending   ClaimStatus = "pending"
	ClaimStatusBroadcast ClaimStatus = "broadcast"
	ClaimStatusConfirmed ClaimStatus = "confirmed"
	ClaimStatusFailed    ClaimStatus = "failed"
)

// Claim tracks a single on-chain mint request lifecycle.
type Claim struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"                                                          json:"id"`
	UserID       uuid.UUID   `gorm:"type:uuid;not null;index:idx_claims_user_created_at,priority:1"                                         json:"user_id"`
	WalletAddr   string      `gorm:"type:varchar(42);not null"                                                                               json:"wallet_addr"`
	Amount       int64       `gorm:"not null;check:chk_claim_amount_gt_0,amount > 0"                                                        json:"amount"`
	Status       ClaimStatus `gorm:"type:varchar(20);not null;index:idx_claims_status_created_at,priority:1;check:chk_claim_status,status IN ('pending','broadcast','confirmed','failed')" json:"status"`
	TxHash       *string     `gorm:"type:varchar(66)"                                                                                         json:"tx_hash"`
	ErrorMessage *string     `gorm:"type:text"                                                                                                json:"error_message"`
	BroadcastAt  *time.Time  `                                                                                                                json:"broadcast_at"`
	ConfirmedAt  *time.Time  `                                                                                                                json:"confirmed_at"`
	FailedAt     *time.Time  `                                                                                                                json:"failed_at"`
	CreatedAt    time.Time   `gorm:"index:idx_claims_user_created_at,priority:2;index:idx_claims_status_created_at,priority:2"             json:"created_at"`
	UpdatedAt    time.Time   `                                                                                                                json:"updated_at"`

	User  User        `gorm:"foreignKey:UserID" json:"-"`
	Items []ClaimItem `gorm:"foreignKey:ClaimID" json:"items,omitempty"`
}

func (c *Claim) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		id, err := uuidV7Func()
		if err != nil {
			id = uuid.New()
		}
		c.ID = id
	}
	return nil
}

// ClaimItem stores per-ledger deduction rows that belong to a claim request.
type ClaimItem struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"                 json:"id"`
	ClaimID             uuid.UUID `gorm:"type:uuid;not null;index"                                        json:"claim_id"`
	LedgerID            uuid.UUID `gorm:"type:uuid;not null;index"                                        json:"ledger_id"`
	PointsTransactionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"                                  json:"points_transaction_id"`
	Amount              int64     `gorm:"not null;check:chk_claim_item_amount_gt_0,amount > 0"           json:"amount"`
	CreatedAt           time.Time `                                                                        json:"created_at"`

	Claim             Claim             `gorm:"foreignKey:ClaimID"             json:"-"`
	Ledger            PointsLedger      `gorm:"foreignKey:LedgerID"            json:"-"`
	PointsTransaction PointsTransaction `gorm:"foreignKey:PointsTransactionID" json:"-"`
}

func (c *ClaimItem) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		id, err := uuidV7Func()
		if err != nil {
			id = uuid.New()
		}
		c.ID = id
	}
	return nil
}
