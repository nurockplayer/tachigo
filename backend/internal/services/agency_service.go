package services

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var ErrAgencyEmailTaken = errors.New("agency email already taken")

type AgencyService struct {
	db *gorm.DB
}

func NewAgencyService(db *gorm.DB) *AgencyService {
	return &AgencyService{db: db}
}

func (s *AgencyService) Create(name, email string) (*models.User, error) {
	user := &models.User{
		Username:     &name,
		Email:        &email,
		Role:         models.RoleAgency,
		PasswordHash: nil,
	}

	if err := s.db.Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "users.email") {
			return nil, ErrAgencyEmailTaken
		}
		return nil, err
	}

	return user, nil
}

func (s *AgencyService) OwnsChannel(agencyUserID uuid.UUID, channelID string) (bool, error) {
	var count int64
	err := s.db.Model(&models.AgencyStreamer{}).
		Where("agency_id = ? AND channel_id = ?", agencyUserID, channelID).
		Count(&count).Error
	return count > 0, err
}
