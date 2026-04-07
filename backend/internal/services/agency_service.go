package services

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var ErrAgencyEmailTaken = errors.New("email already registered")
var ErrAgencyNameTaken = errors.New("name already taken")

type AgencyService struct {
	db *gorm.DB
}

func NewAgencyService(db *gorm.DB) *AgencyService {
	return &AgencyService{db: db}
}

func (s *AgencyService) Create(name, email string) (*models.User, error) {
	if len(name) > 50 {
		return nil, ErrAgencyNameTaken
	}

	var count int64
	if err := s.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAgencyEmailTaken
	}

	count = 0
	if err := s.db.Model(&models.User{}).Where("username = ?", name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAgencyNameTaken
	}

	user := &models.User{
		Username:     &name,
		Email:        &email,
		Role:         models.RoleAgency,
		PasswordHash: nil,
	}

	if err := s.db.Create(user).Error; err != nil {
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
