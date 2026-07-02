package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CharacterKind string

const (
	CharacterCrab     CharacterKind = "crab"
	CharacterDolphin  CharacterKind = "dolphin"
	CharacterTurtle   CharacterKind = "turtle"
	CharacterWhale    CharacterKind = "whale"
	CharacterCapybara CharacterKind = "capybara"
)

// UserCharacter stores a user's unlocked ocean character progression.
type UserCharacter struct {
	ID         uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:idx_user_characters_user_character,priority:1" json:"user_id"`
	Character  CharacterKind `gorm:"type:varchar(50);not null;uniqueIndex:idx_user_characters_user_character,priority:2;check:chk_user_characters_character,character IN ('crab','dolphin','turtle','whale','capybara')" json:"character"`
	Unlocked   bool          `gorm:"not null;default:false" json:"unlocked"`
	Stage      int           `gorm:"not null;default:1;check:chk_user_characters_stage,stage >= 1 AND stage <= 3" json:"stage"`
	XP         int64         `gorm:"not null;default:0;check:chk_user_characters_xp,xp >= 0" json:"xp"`
	UnlockedAt *time.Time    `gorm:"type:timestamptz" json:"unlocked_at,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`

	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (UserCharacter) TableName() string { return "user_characters" }

func (c *UserCharacter) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		id, err := uuidV7Func()
		if err != nil {
			id = uuid.New()
		}
		c.ID = id
	}
	return nil
}
