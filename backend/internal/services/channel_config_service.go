package services

import (
	"github.com/tachigo/tachigo/internal/models"
	"gorm.io/gorm"
)

const DefaultSecondsPerPoint int64 = 60

type ChannelConfigService struct {
	db *gorm.DB
}

func NewChannelConfigService(db *gorm.DB) *ChannelConfigService {
	return &ChannelConfigService{db: db}
}

func (s *ChannelConfigService) UpdateChannelConfig(channelID string, secondsPerPoint int64) (*models.ChannelConfig, error) {
	cfg := &models.ChannelConfig{
		ChannelID:       channelID,
		SecondsPerPoint: secondsPerPoint,
	}

	if err := s.db.
		Where("channel_id = ?", channelID).
		Assign(models.ChannelConfig{SecondsPerPoint: secondsPerPoint}).
		FirstOrCreate(cfg).Error; err != nil {
		return nil, err
	}

	return cfg, nil
}
