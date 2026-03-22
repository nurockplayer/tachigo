package services

import (
	"testing"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
)

func TestGetByID_Found(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	user, err := svc.GetByID(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != userID {
		t.Errorf("ID: want %s, got %s", userID, user.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := NewUserService(newTestDB(t))

	_, err := svc.GetByID(uuid.New())
	if err != ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

func TestUpdateProfile_Username(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	newName := "newusername"
	user, err := svc.UpdateProfile(userID, UpdateProfileInput{Username: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *user.Username != newName {
		t.Errorf("username: want %s, got %s", newName, *user.Username)
	}
}

func TestUpdateProfile_AvatarURL(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	avatar := "https://example.com/avatar.png"
	user, err := svc.UpdateProfile(userID, UpdateProfileInput{AvatarURL: &avatar})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *user.AvatarURL != avatar {
		t.Errorf("avatar_url: want %s, got %s", avatar, *user.AvatarURL)
	}
}

func TestUpdateProfile_DuplicateUsername(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	taken := "taken"
	id1 := uuid.New()
	db.Create(&models.User{ID: id1, Username: &taken, Role: models.RoleViewer})

	id2 := uuid.New()
	db.Create(&models.User{ID: id2, Role: models.RoleViewer})

	_, err := svc.UpdateProfile(id2, UpdateProfileInput{Username: &taken})
	if err != ErrUsernameExists {
		t.Errorf("want ErrUsernameExists, got %v", err)
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	svc := NewUserService(newTestDB(t))

	name := "ghost"
	_, err := svc.UpdateProfile(uuid.New(), UpdateProfileInput{Username: &name})
	if err != ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

func TestListProviders_ReturnsLinkedProviders(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderEmail, ProviderID: "user@example.com"})
	db.Create(&models.AuthProvider{UserID: userID, Provider: models.ProviderTwitch, ProviderID: "twitch-123"})

	providers, err := svc.ListProviders(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) != 2 {
		t.Errorf("want 2 providers, got %d", len(providers))
	}
}

func TestListProviders_EmptyForNewUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	userID := uuid.New()
	db.Create(&models.User{ID: userID, Role: models.RoleViewer})

	providers, err := svc.ListProviders(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("want 0 providers, got %d", len(providers))
	}
}
