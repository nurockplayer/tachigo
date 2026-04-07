package services

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var ErrAgencyEmailTaken = errors.New("email already registered")
var ErrAgencyNameTaken = errors.New("name already taken")
var ErrAgencyNameTooLong = errors.New("agency name exceeds 50 characters")

type AgencyService struct {
	db *gorm.DB
}

func NewAgencyService(db *gorm.DB) *AgencyService {
	return &AgencyService{db: db}
}

func (s *AgencyService) Create(name, email string) (*models.User, error) {
	if utf8.RuneCountInString(name) > 50 {
		return nil, ErrAgencyNameTooLong
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

	// Wrap user + email auth_provider in one transaction so we never end up
	// with a users row but no auth_providers row (or vice-versa).
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return tx.Create(&models.AuthProvider{
			UserID:     user.ID,
			Provider:   models.ProviderEmail,
			ProviderID: email,
		}).Error
	}); err != nil {
		// Handle race condition: unique violation reached DB despite pre-checks.
		var pgErr *pgconn.PgError
		isUniq := errors.Is(err, gorm.ErrDuplicatedKey) ||
			(errors.As(err, &pgErr) && pgErr.Code == "23505")
		if isUniq {
			info := err.Error()
			if pgErr != nil {
				info = pgErr.ConstraintName
			}
			if strings.Contains(info, "email") {
				return nil, ErrAgencyEmailTaken
			}
			return nil, ErrAgencyNameTaken
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

func (s *AgencyService) ListStreamers(agencyID uuid.UUID) ([]models.AgencyStreamer, error) {
	var streamers []models.AgencyStreamer
	if err := s.db.
		Where("agency_id = ?", agencyID).
		Order("created_at ASC").
		Find(&streamers).Error; err != nil {
		return nil, err
	}
	if streamers == nil {
		return []models.AgencyStreamer{}, nil
	}
	return streamers, nil
}

func (s *AgencyService) ListStreamerUserIDs(channelIDs []string) (map[string]uuid.UUID, error) {
	type streamerUserRow struct {
		ChannelID string
		UserID    uuid.UUID
	}

	if len(channelIDs) == 0 {
		return map[string]uuid.UUID{}, nil
	}

	var rows []streamerUserRow
	if err := s.db.Model(&models.Streamer{}).
		Select("channel_id, user_id").
		Where("channel_id IN ?", channelIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	userIDs := make(map[string]uuid.UUID, len(rows))
	for _, row := range rows {
		userIDs[row.ChannelID] = row.UserID
	}

	return userIDs, nil
}
