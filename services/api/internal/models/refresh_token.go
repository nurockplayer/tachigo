package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"                       json:"user_id"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"         json:"-"`
	ExpiresAt time.Time `gorm:"not null"                                       json:"expires_at"`
	CreatedAt time.Time `                                                      json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// Web3Nonce stores one-time nonces for wallet signature verification.
type Web3Nonce struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nonce     string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	Address   string    `gorm:"type:varchar(42);not null;index"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (w *Web3Nonce) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

func (w *Web3Nonce) IsExpired() bool {
	return time.Now().After(w.ExpiresAt)
}
