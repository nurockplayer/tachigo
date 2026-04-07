package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

type AgencyService struct {
	db *gorm.DB
}

func NewAgencyService(db *gorm.DB) *AgencyService {
	return &AgencyService{db: db}
}

func (s *AgencyService) OwnsChannel(agencyUserID uuid.UUID, channelID string) (bool, error) {
	var count int64
	err := s.db.Model(&models.AgencyStreamer{}).
		Where("agency_id = ? AND channel_id = ?", agencyUserID, channelID).
		Count(&count).Error
	return count > 0, err
}
