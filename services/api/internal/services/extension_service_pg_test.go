//go:build integration

package services

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/models"
)

func TestLoginWithExtension_ConcurrentLogin_NoDuplicateUser(t *testing.T) {
	db := newPGTestDB(t)
	cfg := extTestConfig()
	authSvc := NewAuthService(db, cfg)
	watchSvc := NewWatchService(db)
	pointsSvc := NewPointsService(db, watchSvc)
	svc := NewExtensionService(db, cfg, authSvc, pointsSvc)

	twitchID := fmt.Sprintf("twitch-conc-%s", uuid.New().String()[:8])

	const goroutines = 10
	type result struct {
		user *models.User
		err  error
	}
	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			extJWT := makeExtJWT(t, twitchID, "channel-concurrent")
			u, _, err := svc.LoginWithExtension(extJWT)
			results[i] = result{u, err}
		}()
	}
	wg.Wait()

	var successUserID uuid.UUID
	for _, r := range results {
		if r.err != nil {
			t.Errorf("concurrent login error: %v", r.err)
			continue
		}
		if successUserID == uuid.Nil {
			successUserID = r.user.ID
		} else if r.user.ID != successUserID {
			t.Errorf("concurrent logins created different users: %s vs %s", successUserID, r.user.ID)
		}
	}

	var count int64
	db.Model(&models.User{}).
		Joins("JOIN auth_providers ON auth_providers.user_id = users.id").
		Where("auth_providers.provider = ? AND auth_providers.provider_id = ?", models.ProviderTwitch, twitchID).
		Count(&count)
	if count != 1 {
		t.Errorf("want exactly 1 user after concurrent login, got %d", count)
	}
}
