package main

import (
	"context"
	"log"
	"time"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/database"
	"gorm.io/gorm"
)

func bootstrap(cfg *config.Config) *gorm.DB {
	db := database.Connect(cfg.Database.DSN)

	hashCtx, hashCancel := context.WithTimeout(context.Background(), 30*time.Second)
	hashErr := hashLegacyRaffleClaimTokens(hashCtx, db)
	hashCancel()
	if hashErr != nil {
		log.Fatalf("failed to hash existing claim tokens: %v", hashErr)
	}

	return db
}

func hashLegacyRaffleClaimTokens(ctx context.Context, db *gorm.DB) error {
	// Data repair only. Do not add schema DDL here.
	// Prefer Atlas migrations for schema changes and explicit data migrations
	// or ops runbooks for one-time data changes.
	// claim_token was previously a raw UUIDv7 (36 chars); it now stores the
	// SHA-256 hex digest (64 chars). This idempotent repair is data-only.
	return db.WithContext(ctx).Exec(`
		UPDATE raffle_draws
		SET claim_token = encode(sha256(claim_token::bytea), 'hex')
		WHERE length(claim_token) = 36
	`).Error
}
