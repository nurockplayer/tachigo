package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WatchTimeStat accumulates total watch seconds per viewer per channel.
// Each row is unique on (user_id, channel_id); seconds are upserted atomically.
type WatchTimeStat struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"              json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_watch_time_user_channel"  json:"user_id"`
	ChannelID         string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_watch_time_user_channel" json:"channel_id"`
	TotalWatchSeconds int64     `gorm:"not null;default:0"                                          json:"total_watch_seconds"`
	CreatedAt         time.Time `                                                                   json:"created_at"`
	UpdatedAt         time.Time `                                                                   json:"updated_at"`
}

func (w *WatchTimeStat) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		w.ID = id
	}
	return nil
}

// BroadcastTimeStat accumulates total broadcast seconds per streamer per channel.
// Each row is unique on (streamer_id, channel_id); seconds are upserted atomically.
type BroadcastTimeStat struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"                    json:"id"`
	StreamerID             uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_broadcast_time_streamer_channel" json:"streamer_id"`
	ChannelID              string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_broadcast_time_streamer_channel" json:"channel_id"`
	TotalBroadcastSeconds  int64     `gorm:"not null;default:0"                                                json:"total_broadcast_seconds"`
	CreatedAt              time.Time `                                                                          json:"created_at"`
	UpdatedAt              time.Time `                                                                          json:"updated_at"`
}

func (b *BroadcastTimeStat) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		b.ID = id
	}
	return nil
}

// BroadcastTimeLog records each heartbeat's broadcast-second increment.
// Used for time-windowed queries (daily / monthly / yearly).
// RecordedAt is indexed to support efficient range scans.
type BroadcastTimeLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StreamerID uuid.UUID `gorm:"type:uuid;not null;index"                       json:"streamer_id"`
	ChannelID  string    `gorm:"type:varchar(255);not null;index"               json:"channel_id"`
	Seconds    int64     `gorm:"not null"                                       json:"seconds"`
	RecordedAt time.Time `gorm:"not null;index"                                 json:"recorded_at"`
}

func (b *BroadcastTimeLog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		id, _ := uuid.NewV7()
		b.ID = id
	}
	return nil
}
