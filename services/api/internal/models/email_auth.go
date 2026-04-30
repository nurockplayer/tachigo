package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailVerification stores one-time tokens for verifying a user's email address.
type EmailVerification struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (e *EmailVerification) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (e *EmailVerification) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// PasswordReset stores one-time tokens for resetting a user's password.
type PasswordReset struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email     string    `gorm:"type:varchar(255);not null;index"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (p *PasswordReset) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (p *PasswordReset) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
