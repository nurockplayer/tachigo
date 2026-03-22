package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type UpdateProfileInput struct {
	Username  *string `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

func (s *UserService) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (s *UserService) UpdateProfile(id uuid.UUID, input UpdateProfileInput) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if input.Username != nil {
		// Uniqueness check
		var count int64
		s.db.Model(&models.User{}).
			Where("username = ? AND id != ?", *input.Username, id).
			Count(&count)
		if count > 0 {
			return nil, ErrUsernameExists
		}
		user.Username = input.Username
	}

	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) ListProviders(userID uuid.UUID) ([]models.AuthProvider, error) {
	var providers []models.AuthProvider
	if err := s.db.Where("user_id = ?", userID).Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}
