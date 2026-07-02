package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StreamerFamiliarity stores per-user watch familiarity for one streamer channel.
type StreamerFamiliarity struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID                 uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_streamer_familiarities_user_channel,priority:1" json:"user_id"`
	ChannelID              string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_streamer_familiarities_user_channel,priority:2" json:"channel_id"`
	CumulativeWatchSeconds int64      `gorm:"not null;default:0;check:chk_streamer_familiarities_watch_seconds,cumulative_watch_seconds >= 0" json:"cumulative_watch_seconds"`
	LastWatchedAt          *time.Time `gorm:"type:timestamptz" json:"last_watched_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (StreamerFamiliarity) TableName() string { return "streamer_familiarities" }

func (f *StreamerFamiliarity) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		id, err := uuidV7Func()
		if err != nil {
			id = uuid.New()
		}
		f.ID = id
	}
	return nil
}
