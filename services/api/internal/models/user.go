package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleViewer   UserRole = "viewer"
	RoleStreamer UserRole = "streamer"
	RoleAgency   UserRole = "agency"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username            *string        `gorm:"type:varchar(50);uniqueIndex"                   json:"username"`
	Email               *string        `gorm:"type:varchar(255);uniqueIndex"                  json:"email"`
	PasswordHash        *string        `gorm:"type:varchar(255)"                              json:"-"`
	AvatarURL           *string        `gorm:"type:text"                                      json:"avatar_url"`
	Role                UserRole       `gorm:"type:user_role;default:'viewer'"                json:"role"`
	IsActive            bool           `gorm:"default:true"                                   json:"is_active"`
	EmailVerified       bool           `gorm:"default:false"                                  json:"email_verified"`
	ActiveCharacter     CharacterKind  `gorm:"type:varchar(50);not null;default:'crab';check:chk_users_active_character,active_character IN ('crab','dolphin','turtle','whale','capybara')" json:"active_character"`
	SwitchCooldownUntil *time.Time     `gorm:"type:timestamptz"                              json:"switch_cooldown_until,omitempty"`
	CreatedAt           time.Time      `                                                      json:"created_at"`
	UpdatedAt           time.Time      `                                                      json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index"                                          json:"-"`

	AuthProviders []AuthProvider    `gorm:"foreignKey:UserID" json:"auth_providers,omitempty"`
	Addresses     []ShippingAddress `gorm:"foreignKey:UserID" json:"addresses,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
