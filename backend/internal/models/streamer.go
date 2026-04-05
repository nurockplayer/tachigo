package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Streamer struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_streamers_user_channel" json:"user_id"`
	ChannelID   string    `gorm:"type:varchar(255);not null;index;uniqueIndex:idx_streamers_user_channel" json:"channel_id"`
	DisplayName string    `gorm:"type:varchar(255)"                              json:"display_name"`
	CreatedAt   time.Time `                                                      json:"created_at"`
	UpdatedAt   time.Time `                                                      json:"updated_at"`
}

func (s *Streamer) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		s.ID = id
	}
	return nil
}
