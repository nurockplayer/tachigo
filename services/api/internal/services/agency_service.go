package services

import (
	"context"
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
var ErrAgencyNotFound = errors.New("agency not found")
var ErrDuplicateChannelID = errors.New("channel_id maps to multiple streamer users")

type AgencyService struct {
	db *gorm.DB
}

func NewAgencyService(db *gorm.DB) *AgencyService {
	return &AgencyService{db: db}
}

func (s *AgencyService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.db.WithContext(ctx)
}

func (s *AgencyService) Create(name, email string) (*models.User, error) {
	return s.CreateContext(context.Background(), name, email)
}

func (s *AgencyService) CreateContext(ctx context.Context, name, email string) (*models.User, error) {
	db := s.dbWithContext(ctx)
	if utf8.RuneCountInString(name) > 50 {
		return nil, ErrAgencyNameTooLong
	}

	var count int64
	if err := db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAgencyEmailTaken
	}

	count = 0
	if err := db.Model(&models.User{}).Where("username = ?", name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAgencyNameTaken
	}

	user := &models.User{
		Username:      &name,
		Email:         &email,
		Role:          models.RoleAgency,
		PasswordHash:  nil,
		EmailVerified: true,
	}

	// Wrap user + email auth_provider in one transaction so we never end up
	// with a users row but no auth_providers row (or vice-versa).
	if err := db.Transaction(func(tx *gorm.DB) error {
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

func (s *AgencyService) UpdateSettings(agencyID uuid.UUID, name string) error {
	return s.UpdateSettingsContext(context.Background(), agencyID, name)
}

func (s *AgencyService) UpdateSettingsContext(ctx context.Context, agencyID uuid.UUID, name string) error {
	if utf8.RuneCountInString(name) > 50 {
		return ErrAgencyNameTooLong
	}

	res := s.dbWithContext(ctx).Model(&models.User{}).
		Where("id = ? AND role = ?", agencyID, models.RoleAgency).
		Update("username", name)
	if res.Error != nil {
		var pgErr *pgconn.PgError
		isUniq := errors.Is(res.Error, gorm.ErrDuplicatedKey) ||
			(errors.As(res.Error, &pgErr) && pgErr.Code == "23505") ||
			strings.Contains(res.Error.Error(), "UNIQUE constraint failed")
		if isUniq {
			return ErrAgencyNameTaken
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAgencyNotFound
	}
	return nil
}

// GetByID returns the agency user and whether onboarding is complete.
// onboardingComplete is true when the agency has set a password (PasswordHash IS NOT NULL).
// Returns ErrAgencyNotFound if no user with the given id and role=agency exists.
func (s *AgencyService) GetByID(id uuid.UUID) (*models.User, bool, error) {
	return s.GetByIDContext(context.Background(), id)
}

func (s *AgencyService) GetByIDContext(ctx context.Context, id uuid.UUID) (*models.User, bool, error) {
	var user models.User
	if err := s.dbWithContext(ctx).Where("id = ? AND role = ?", id, models.RoleAgency).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, ErrAgencyNotFound
		}
		return nil, false, err
	}
	return &user, user.PasswordHash != nil, nil
}

func (s *AgencyService) OwnsChannel(agencyUserID uuid.UUID, channelID string) (bool, error) {
	return s.OwnsChannelContext(context.Background(), agencyUserID, channelID)
}

func (s *AgencyService) OwnsChannelContext(ctx context.Context, agencyUserID uuid.UUID, channelID string) (bool, error) {
	var count int64
	err := s.dbWithContext(ctx).Model(&models.AgencyStreamer{}).
		Where("agency_id = ? AND channel_id = ?", agencyUserID, channelID).
		Count(&count).Error
	return count > 0, err
}

func (s *AgencyService) ListStreamers(agencyID uuid.UUID) ([]models.AgencyStreamer, error) {
	return s.ListStreamersContext(context.Background(), agencyID)
}

func (s *AgencyService) ListStreamersContext(ctx context.Context, agencyID uuid.UUID) ([]models.AgencyStreamer, error) {
	db := s.dbWithContext(ctx)
	var streamers []models.AgencyStreamer
	if err := db.
		Where("agency_id = ?", agencyID).
		Order("created_at ASC").
		Find(&streamers).Error; err != nil {
		return nil, err
	}
	if len(streamers) == 0 {
		var count int64
		if err := db.Model(&models.User{}).
			Where("id = ? AND role = ?", agencyID, models.RoleAgency).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrAgencyNotFound
		}
	}
	return streamers, nil
}

func (s *AgencyService) ListStreamerUserIDs(channelIDs []string) (map[string]uuid.UUID, error) {
	return s.ListStreamerUserIDsContext(context.Background(), channelIDs)
}

func (s *AgencyService) ListStreamerUserIDsContext(ctx context.Context, channelIDs []string) (map[string]uuid.UUID, error) {
	type streamerUserRow struct {
		ChannelID string
		UserID    uuid.UUID
	}

	if len(channelIDs) == 0 {
		return map[string]uuid.UUID{}, nil
	}

	var rows []streamerUserRow
	if err := s.dbWithContext(ctx).Model(&models.Streamer{}).
		Select("channel_id, user_id").
		Where("channel_id IN ?", channelIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	userIDs := make(map[string]uuid.UUID, len(rows))
	for _, row := range rows {
		if _, exists := userIDs[row.ChannelID]; exists {
			return nil, ErrDuplicateChannelID
		}
		userIDs[row.ChannelID] = row.UserID
	}

	return userIDs, nil
}
