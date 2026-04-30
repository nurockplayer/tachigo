package models

import (
	"time"

	"github.com/google/uuid"
)

type AgencyStreamer struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AgencyID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_agency_streamers_agency_channel"`
	ChannelID string    `gorm:"not null;uniqueIndex:idx_agency_streamers_agency_channel"`
	CreatedAt time.Time

	Agency User `gorm:"foreignKey:AgencyID;constraint:OnDelete:CASCADE"`
}
