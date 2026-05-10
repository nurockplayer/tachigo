//go:build integration

package services

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestRefresh_ConcurrentReuseOnlyAllowsSingleSuccess(t *testing.T) {
	db := newPGTestDB(t)
	svc := NewAuthService(db, testConfig())
	_, tokens, err := svc.Register(RegisterInput{
		Username: "pg_refresh_race",
		Email:    "pg-refresh-race@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	const callbackName = "test:block_refresh_token_lookup"
	var lookupCount atomic.Int32
	release := make(chan struct{})
	bothLookupsReady := make(chan struct{})

	if err := db.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "refresh_tokens" {
			return
		}
		if lookupCount.Add(1) > 2 {
			return
		}
		if lookupCount.Load() == 2 {
			close(bothLookupsReady)
		}
		<-release
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	defer func() {
		if err := db.Callback().Query().Remove(callbackName); err != nil {
			t.Fatalf("remove query callback: %v", err)
		}
	}()

	type refreshResult struct {
		tokens *TokenPair
		err    error
	}

	start := make(chan struct{})
	results := make(chan refreshResult, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			tokenPair, err := svc.Refresh(tokens.RefreshToken)
			results <- refreshResult{tokens: tokenPair, err: err}
		}()
	}

	close(start)

	select {
	case <-bothLookupsReady:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for concurrent refresh lookups")
	}
	close(release)

	wg.Wait()
	close(results)

	var successCount, invalidCount int
	for result := range results {
		if result.err == nil {
			if result.tokens == nil || result.tokens.RefreshToken == "" {
				t.Fatalf("successful refresh should return token pair, got %#v", result.tokens)
			}
			successCount++
			continue
		}
		if result.err == ErrInvalidToken {
			invalidCount++
			continue
		}
		t.Fatalf("unexpected refresh error: %v", result.err)
	}

	if successCount != 1 || invalidCount != 1 {
		t.Fatalf("want 1 success and 1 ErrInvalidToken, got success=%d invalid=%d", successCount, invalidCount)
	}
}
