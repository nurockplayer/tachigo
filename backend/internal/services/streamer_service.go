package services

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrStreamerNotFound = errors.New("streamer not found")
	ErrChannelNotOwned  = errors.New("channel not owned by user")
)

type StreamerService struct {
	db        *gorm.DB
	pointsSvc *PointsService
}

type StreamerStats struct {
	CurrentSessionSeconds  int64   `json:"current_session_seconds"`
	DailySeconds           int64   `json:"daily_seconds"`
	MonthlySeconds         int64   `json:"monthly_seconds"`
	YearlySeconds          int64   `json:"yearly_seconds"`
	UniqueMiners           int64   `json:"unique_miners"`
	AvgSessionSeconds      float64 `json:"avg_session_seconds"`
	TotalTokenMinted       int64   `json:"total_token_minted"`
	SpendableInCirculation int64   `json:"spendable_in_circulation"`
}

func NewStreamerService(db *gorm.DB, pointsSvc *PointsService) *StreamerService {
	return &StreamerService{db: db, pointsSvc: pointsSvc}
}

func (s *StreamerService) Register(userID uuid.UUID, channelID, displayName string) (*models.Streamer, error) {
	// Verify the channelID matches the user's Twitch auth_provider entry.
	var count int64
	if err := s.db.Model(&models.AuthProvider{}).
		Where("user_id = ? AND provider = ? AND provider_id = ?", userID, models.ProviderTwitch, channelID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrChannelNotOwned
	}

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

func (s *StreamerService) Create(userID uuid.UUID, agencyUserID *uuid.UUID, channelID string) (*models.Streamer, error) {
	var count int64
	if err := s.db.Model(&models.AuthProvider{}).
		Where("user_id = ? AND provider = ? AND provider_id = ?", userID, models.ProviderTwitch, channelID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrChannelNotOwned
	}

	streamer := &models.Streamer{
		UserID:       userID,
		AgencyUserID: agencyUserID,
		ChannelID:    channelID,
	}

	if err := s.db.
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		Assign(models.Streamer{AgencyUserID: agencyUserID}).
		FirstOrCreate(streamer).Error; err != nil {
		return nil, err
	}

	return streamer, nil
}

func (s *StreamerService) OwnsChannel(userID uuid.UUID, channelID string) (bool, error) {
	var count int64
	if err := s.db.Model(&models.Streamer{}).
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *StreamerService) OwnsAgencyChannel(agencyUserID uuid.UUID, channelID string) (bool, error) {
	var count int64
	if err := s.db.Model(&models.Streamer{}).
		Where("agency_user_id = ? AND channel_id = ?", agencyUserID, channelID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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

func (s *StreamerService) GetByID(id uuid.UUID) (*models.Streamer, error) {
	var streamer models.Streamer
	if err := s.db.Where("id = ?", id).First(&streamer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStreamerNotFound
		}
		return nil, err
	}
	return &streamer, nil
}

func (s *StreamerService) ListAll() ([]models.Streamer, error) {
	var streamers []models.Streamer
	if err := s.db.Order("created_at ASC").Find(&streamers).Error; err != nil {
		return nil, err
	}
	return streamers, nil
}

func (s *StreamerService) ListByAgencyUserID(agencyUserID uuid.UUID) ([]models.Streamer, error) {
	var streamers []models.Streamer
	if err := s.db.
		Where("agency_user_id = ?", agencyUserID).
		Order("created_at ASC").
		Find(&streamers).Error; err != nil {
		return nil, err
	}
	return streamers, nil
}

func (s *StreamerService) OwnsStreamer(agencyUserID uuid.UUID, streamerID uuid.UUID) (bool, error) {
	var count int64
	if err := s.db.Model(&models.Streamer{}).
		Where("id = ? AND agency_user_id = ?", streamerID, agencyUserID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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

func (s *StreamerService) GetStats(streamerID uuid.UUID) (*StreamerStats, error) {
	var streamer models.Streamer
	if err := s.db.Where("id = ?", streamerID).First(&streamer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStreamerNotFound
		}
		return nil, err
	}
	channelID := streamer.ChannelID

	broadcast, err := s.pointsSvc.GetBroadcastStats(streamer.UserID, channelID)
	if err != nil {
		return nil, err
	}

	var traffic struct {
		UniqueMiners      int64   `gorm:"column:unique_miners"`
		AvgSessionSeconds float64 `gorm:"column:avg_session_seconds"`
	}
	if err := s.db.Raw(`
		SELECT
			COUNT(DISTINCT user_id) AS unique_miners,
			COALESCE(AVG(accumulated_seconds), 0) AS avg_session_seconds
		FROM watch_sessions
		WHERE channel_id = ?
	`, channelID).Scan(&traffic).Error; err != nil {
		return nil, err
	}

	var economy struct {
		TotalTokenMinted       int64 `gorm:"column:total_token_minted"`
		SpendableInCirculation int64 `gorm:"column:spendable_in_circulation"`
	}
	if err := s.db.Raw(`
		SELECT
			COALESCE(SUM(cumulative_total), 0) AS total_token_minted,
			COALESCE(SUM(spendable_balance), 0) AS spendable_in_circulation
		FROM points_ledgers
		WHERE channel_id = ?
	`, channelID).Scan(&economy).Error; err != nil {
		return nil, err
	}

	return &StreamerStats{
		CurrentSessionSeconds:  broadcast.CurrentSessionSeconds,
		DailySeconds:           broadcast.DailySeconds,
		MonthlySeconds:         broadcast.MonthlySeconds,
		YearlySeconds:          broadcast.YearlySeconds,
		UniqueMiners:           traffic.UniqueMiners,
		AvgSessionSeconds:      traffic.AvgSessionSeconds,
		TotalTokenMinted:       economy.TotalTokenMinted,
		SpendableInCirculation: economy.SpendableInCirculation,
	}, nil
}
