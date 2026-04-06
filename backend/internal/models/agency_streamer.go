package models

import (
	"time"

	"github.com/google/uuid"
)

type AgencyStreamer struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AgencyID  uuid.UUID `gorm:"type:uuid;not null"`
	ChannelID string    `gorm:"not null"`
	CreatedAt time.Time
}
