package models

import "time"

// ChannelConfig stores per-channel watch-time earning settings.
// A missing row means the channel uses system defaults.
type ChannelConfig struct {
	ChannelID       string    `gorm:"type:varchar(255);primaryKey" json:"channel_id"`
	SecondsPerPoint int64     `gorm:"not null;default:60"         json:"seconds_per_point"`
	CreatedAt       time.Time `                                   json:"created_at"`
	UpdatedAt       time.Time `                                   json:"updated_at"`
}
