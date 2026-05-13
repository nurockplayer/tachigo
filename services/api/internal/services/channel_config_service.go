package services

import (
	"context"

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

func (s *ChannelConfigService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.db.WithContext(ctx)
}

func (s *ChannelConfigService) Get(channelID string) (*models.ChannelConfig, error) {
	return s.GetContext(context.Background(), channelID)
}

func (s *ChannelConfigService) GetContext(ctx context.Context, channelID string) (*models.ChannelConfig, error) {
	var cfg models.ChannelConfig
	if err := s.dbWithContext(ctx).Where("channel_id = ?", channelID).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (s *ChannelConfigService) EffectiveMultiplier(channelID string) (int64, error) {
	return s.EffectiveMultiplierContext(context.Background(), channelID)
}

func (s *ChannelConfigService) EffectiveMultiplierContext(ctx context.Context, channelID string) (int64, error) {
	cfg, err := s.GetContext(ctx, channelID)
	if err != nil {
		return 0, err
	}
	if cfg == nil || cfg.Multiplier <= 0 {
		return 1, nil
	}
	return cfg.Multiplier, nil
}

func (s *ChannelConfigService) UpdateChannelConfig(channelID string, secondsPerPoint, multiplier int64) (*models.ChannelConfig, error) {
	return s.UpdateChannelConfigContext(context.Background(), channelID, secondsPerPoint, multiplier)
}

func (s *ChannelConfigService) UpdateChannelConfigContext(ctx context.Context, channelID string, secondsPerPoint, multiplier int64) (*models.ChannelConfig, error) {
	cfg, err := s.GetContext(ctx, channelID)
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

	if err := s.dbWithContext(ctx).
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
