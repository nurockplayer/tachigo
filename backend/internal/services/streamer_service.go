package services

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var ErrStreamerNotFound = errors.New("streamer not found")

type StreamerService struct {
	db        *gorm.DB
	pointsSvc *PointsService
}

func NewStreamerService(db *gorm.DB, pointsSvc *PointsService) *StreamerService {
	return &StreamerService{db: db, pointsSvc: pointsSvc}
}

func (s *StreamerService) Register(userID uuid.UUID, channelID, displayName string) (*models.Streamer, error) {
	streamer := &models.Streamer{
		UserID:      userID,
		ChannelID:   channelID,
		DisplayName: displayName,
	}

	if err := s.db.
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		Assign(models.Streamer{DisplayName: displayName}).
		FirstOrCreate(streamer).Error; err != nil {
		return nil, err
	}

	return streamer, nil
}

func (s *StreamerService) ListChannels(userID uuid.UUID) ([]models.Streamer, error) {
	var streamers []models.Streamer
	if err := s.db.
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&streamers).Error; err != nil {
		return nil, err
	}
	if streamers == nil {
		return []models.Streamer{}, nil
	}
	return streamers, nil
}

func (s *StreamerService) GetChannelStats(channelID string) (*BroadcastStats, error) {
	var provider models.AuthProvider
	if err := s.db.
		Where("provider = ? AND provider_id = ?", models.ProviderTwitch, channelID).
		First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStreamerNotFound
		}
		return nil, err
	}

	return s.pointsSvc.GetBroadcastStats(provider.UserID, channelID)
}
