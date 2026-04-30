package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProviderType string

const (
	ProviderTwitch ProviderType = "twitch"
	ProviderGoogle ProviderType = "google"
	ProviderWeb3   ProviderType = "web3"
	ProviderEmail  ProviderType = "email"
)

// AuthProvider stores a linked external login method for a user.
// For Web3, ProviderID is the wallet address (checksummed).
type AuthProvider struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index"                       json:"user_id"`
	Provider       ProviderType   `gorm:"type:varchar(20);not null"                      json:"provider"`
	ProviderID     string         `gorm:"type:varchar(255);not null"                     json:"provider_id"`
	AccessToken    *string        `gorm:"type:text"                                      json:"-"`
	RefreshToken   *string        `gorm:"type:text"                                      json:"-"`
	TokenExpiresAt *time.Time     `                                                      json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt      time.Time      `                                                      json:"created_at"`
	UpdatedAt      time.Time      `                                                      json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index"                                          json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (a *AuthProvider) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
