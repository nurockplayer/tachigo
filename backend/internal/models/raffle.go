package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RaffleStatus string
type RaffleSource string

const (
	RaffleStatusDraft     RaffleStatus = "draft"
	RaffleStatusActive    RaffleStatus = "active"
	RaffleStatusCompleted RaffleStatus = "completed"

	RaffleSourceCSV       RaffleSource = "csv"
	RaffleSourceTwitchAPI RaffleSource = "twitch_api"
)

// Raffle represents a single raffle event owned by a streamer.
type Raffle struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID    `gorm:"type:uuid;not null;index"                       json:"user_id"`
	Title       string       `gorm:"type:varchar(255);not null"                     json:"title"`
	Status      RaffleStatus `gorm:"type:varchar(50);not null;default:'draft'"      json:"status"`
	Source      RaffleSource `gorm:"type:varchar(50);not null;default:'csv'"        json:"source"`
	ScheduledAt *time.Time   `                                                      json:"scheduled_at"`
	CreatedAt   time.Time    `                                                      json:"created_at"`
	UpdatedAt   time.Time    `                                                      json:"updated_at"`
}

func (r *Raffle) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		r.ID = id
	}
	return nil
}

// RaffleEntry is one participant row in a raffle.
// UserID is set by the service layer for users with a tachigo account; the
// pointer allows nil in direct-insert test fixtures without a linked account.
type RaffleEntry struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid();uniqueIndex:idx_entry_id_raffle,priority:1"                          json:"id"`
	RaffleID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_raffle_entry_twitch;uniqueIndex:idx_entry_id_raffle,priority:2"                json:"raffle_id"`
	UserID      *uuid.UUID `gorm:"type:uuid;index"                                         json:"user_id"`
	TwitchLogin string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_raffle_entry_twitch" json:"twitch_login"`
	DisplayName string     `gorm:"type:varchar(255)"                                       json:"display_name"`
	CreatedAt   time.Time  `                                                               json:"created_at"`
	Raffle      Raffle     `gorm:"foreignKey:RaffleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	User        *User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"  json:"-"`
}

func (e *RaffleEntry) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		e.ID = id
	}
	return nil
}

// RaffleDraw records one drawn winner.
// ClaimToken is a one-time token sent to the winner for submitting shipping info.
type RaffleDraw struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"        json:"id"`
	RaffleID       uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex:idx_raffle_draw_entry"  json:"raffle_id"`
	EntryID        uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex:idx_raffle_draw_entry"  json:"entry_id"`
	ClaimToken     string      `gorm:"type:varchar(255);not null;uniqueIndex"                json:"claim_token"`
	ClaimExpiresAt time.Time   `                                                             json:"claim_expires_at"`
	DrawnAt        time.Time   `                                                             json:"drawn_at"`
	Raffle         Raffle      `gorm:"foreignKey:RaffleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Entry          RaffleEntry `gorm:"foreignKey:EntryID,RaffleID;references:ID,RaffleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"entry,omitempty"`
}

func (d *RaffleDraw) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		d.ID = id
	}
	return nil
}

// RaffleClaim holds the shipping info submitted by the winner.
type RaffleClaim struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DrawID        uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"                 json:"draw_id"`
	RecipientName string     `gorm:"type:varchar(255);not null"                     json:"recipient_name"`
	Phone         string     `gorm:"type:varchar(50)"                               json:"phone"`
	AddressLine1  string     `gorm:"type:varchar(255);not null"                     json:"address_line1"`
	AddressLine2  string     `gorm:"type:varchar(255)"                              json:"address_line2"`
	City          string     `gorm:"type:varchar(100);not null"                     json:"city"`
	PostalCode    string     `gorm:"type:varchar(20)"                               json:"postal_code"`
	Country       string     `gorm:"type:varchar(10);not null;default:'TW'"         json:"country"`
	SubmittedAt   time.Time  `                                                      json:"submitted_at"`
	Draw          RaffleDraw `gorm:"foreignKey:DrawID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (c *RaffleClaim) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		c.ID = id
	}
	return nil
}
