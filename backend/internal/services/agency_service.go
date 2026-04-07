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

	if err := s.db.Create(user).Error; err != nil {
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

	// Create email auth_providers record so /users/me/providers is consistent
	// with accounts created via standard email registration.
	s.db.Create(&models.AuthProvider{
		UserID:     user.ID,
		Provider:   models.ProviderEmail,
		ProviderID: email,
	})

	return user, nil
}

func (s *AgencyService) OwnsChannel(agencyUserID uuid.UUID, channelID string) (bool, error) {
	var count int64
	err := s.db.Model(&models.AgencyStreamer{}).
		Where("agency_id = ? AND channel_id = ?", agencyUserID, channelID).
		Count(&count).Error
	return count > 0, err
}
