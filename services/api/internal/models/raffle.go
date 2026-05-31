package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RaffleStatus string
type RaffleMode string
type RaffleSource string

const (
	RaffleStatusDraft     RaffleStatus = "draft"
	RaffleStatusActive    RaffleStatus = "active"
	RaffleStatusCompleted RaffleStatus = "completed"

	RaffleModePublic      RaffleMode = "public"
	RaffleModeSubscribers RaffleMode = "subscribers_only"

	RaffleSourceCSV             RaffleSource = "csv"
	RaffleSourceTwitchAPI       RaffleSource = "twitch_api"
	RaffleSourceExtensionButton RaffleSource = "extension_button"
)

// Raffle represents a single raffle event owned by a streamer.
type Raffle struct {
	ID                uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID    `gorm:"type:uuid;not null;index"                       json:"user_id"`
	Title             string       `gorm:"type:varchar(255);not null"                     json:"title"`
	Status            RaffleStatus `gorm:"type:varchar(50);not null;default:'draft'"      json:"status"`
	Source            RaffleSource `gorm:"type:varchar(50);not null;default:'csv'"        json:"source"`
	Mode              RaffleMode   `gorm:"type:varchar(50);not null;default:'public'"     json:"mode"`
	EntryOpen         bool         `gorm:"not null;default:false"                         json:"entry_open"`
	WinnerCount       int          `gorm:"not null;default:1"                             json:"winner_count"`
	ScheduledAt       *time.Time   `                          json:"scheduled_at"`
	DiscordWebhookURL *string      `gorm:"type:varchar(512)"  json:"-"`
	CreatedAt         time.Time    `                          json:"created_at"`
	UpdatedAt         time.Time    `                          json:"updated_at"`
}

// DiscordWebhookConfigured reports whether a webhook URL is set without
// exposing the secret token embedded in the URL.
func (r *Raffle) DiscordWebhookConfigured() bool { return r.DiscordWebhookURL != nil }

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
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid();uniqueIndex:idx_entry_id_raffle,priority:1"                          json:"id"`
	RaffleID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_raffle_entry_twitch;uniqueIndex:idx_entry_id_raffle,priority:2"                json:"raffle_id"`
	UserID           *uuid.UUID `gorm:"type:uuid;index"                                         json:"user_id"`
	TwitchLogin      string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_raffle_entry_twitch" json:"twitch_login"`
	DisplayName      string     `gorm:"type:varchar(255)"                                       json:"display_name"`
	Source           string     `gorm:"type:varchar(50);not null;default:'csv'"                 json:"source"`
	Eligible         bool       `gorm:"not null;default:true"                                   json:"eligible"`
	IneligibleReason string     `gorm:"type:varchar(100);not null;default:''"                  json:"ineligible_reason"`
	CreatedAt        time.Time  `                                                               json:"created_at"`
	Raffle           Raffle     `gorm:"foreignKey:RaffleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	User             *User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"  json:"-"`
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
// ClaimToken stores the SHA-256 hex hash of the raw token.
// ClaimTokenRaw is a transient field populated only when a draw is first created;
// it is never persisted and carries the raw token for the HTTP response and email.
type RaffleDraw struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"        json:"id"`
	RaffleID       uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_raffle_draw_entry"  json:"raffle_id"`
	EntryID        uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_raffle_draw_entry"  json:"entry_id"`
	ClaimToken     string           `gorm:"type:varchar(255);not null;uniqueIndex"                json:"-"`
	ClaimTokenRaw  string           `gorm:"-"                                                     json:"claim_token,omitempty"`
	ClaimExpiresAt time.Time        `                                                             json:"claim_expires_at"`
	DrawnAt        time.Time        `                                                             json:"drawn_at"`
	PrizeTierID    *uuid.UUID       `gorm:"type:uuid"              json:"prize_tier_id,omitempty"`
	PrizeTier      *RafflePrizeTier `gorm:"foreignKey:PrizeTierID" json:"prize_tier,omitempty"`
	Raffle         Raffle           `gorm:"foreignKey:RaffleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Entry          RaffleEntry      `gorm:"foreignKey:EntryID,RaffleID;references:ID,RaffleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"entry,omitempty"`
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

// RafflePrizeTier represents one prize layer within a Raffle.
type RafflePrizeTier struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RaffleID         uuid.UUID `gorm:"type:uuid;not null;index"                       json:"raffle_id"`
	Name             string    `gorm:"type:varchar(255);not null"                     json:"name"`
	PrizeDescription string    `gorm:"type:text;not null;default:''"                  json:"prize_description"`
	WinnerCount      int       `gorm:"not null"                                       json:"winner_count"`
	DrawnCount       int       `gorm:"not null;default:0"                             json:"drawn_count"`
	Position         int       `gorm:"not null;default:0"                             json:"position"`
	CreatedAt        time.Time `                                                       json:"created_at"`
	UpdatedAt        time.Time `                                                       json:"updated_at"`
}

func (RafflePrizeTier) TableName() string { return "raffle_prize_tiers" }

func (t *RafflePrizeTier) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		t.ID = id
	}
	return nil
}
