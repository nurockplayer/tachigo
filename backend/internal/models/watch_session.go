package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WatchSession records a viewer's active or completed watch session in a channel.
// Only one active session (is_active=true) is allowed per (user_id, channel_id) pair.
// The partial unique index is created manually in main.go — GORM does not support
// partial indexes via struct tags.
//
// Session lifecycle:
//
//	active  : is_active = true,  ended_at = NULL
//	finished: is_active = false, ended_at = <timestamp>
type WatchSession struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID             uuid.UUID  `gorm:"type:uuid;not null;index"                       json:"user_id"`
	ChannelID          string     `gorm:"type:varchar(255);not null;index"               json:"channel_id"`
	AccumulatedSeconds int64      `gorm:"not null;default:0"                             json:"accumulated_seconds"`
	RewardedSeconds    int64      `gorm:"not null;default:0"                             json:"rewarded_seconds"`
	LastHeartbeatAt    time.Time  `gorm:"not null;default:now()"                         json:"last_heartbeat_at"`
	IsActive           bool       `gorm:"not null;default:true;index"                    json:"is_active"`
	EndedAt            *time.Time `                                                      json:"ended_at"`
	CreatedAt          time.Time  `                                                      json:"created_at"`
	UpdatedAt          time.Time  `                                                      json:"updated_at"`
}

func (w *WatchSession) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
