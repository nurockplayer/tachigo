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

func (s *ChannelConfigService) Get(channelID string) (*models.ChannelConfig, error) {
	var cfg models.ChannelConfig
	if err := s.db.Where("channel_id = ?", channelID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (s *ChannelConfigService) EffectiveMultiplier(channelID string) (int64, error) {
	cfg, err := s.Get(channelID)
	if err != nil {
		return 0, err
	}
	if cfg == nil || cfg.Multiplier <= 0 {
		return 1, nil
	}
	return cfg.Multiplier, nil
}

func (s *ChannelConfigService) UpdateChannelConfig(channelID string, secondsPerPoint, multiplier int64) (*models.ChannelConfig, error) {
	cfg, err := s.Get(channelID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &models.ChannelConfig{
			ChannelID:       channelID,
			SecondsPerPoint: DefaultSecondsPerPoint,
			Multiplier:      1,
		}
	}
	if secondsPerPoint > 0 {
		cfg.SecondsPerPoint = secondsPerPoint
	}
	if multiplier > 0 {
		cfg.Multiplier = multiplier
	}

	if err := s.db.
		Where("channel_id = ?", channelID).
		Assign(models.ChannelConfig{
			SecondsPerPoint: cfg.SecondsPerPoint,
			Multiplier:      cfg.Multiplier,
		}).
		FirstOrCreate(cfg).Error; err != nil {
		return nil, err
	}

	return cfg, nil
}
