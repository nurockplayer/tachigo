package services

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrRaffleNotFound    = errors.New("raffle not found")
	ErrRaffleForbidden   = errors.New("raffle does not belong to this user")
	ErrRaffleExhausted   = errors.New("all entries have been drawn")
	ErrRaffleCompleted   = errors.New("raffle is already completed")
	ErrClaimTokenExpired = errors.New("claim token has expired")
	ErrClaimNotFound     = errors.New("claim token not found")
	ErrClaimAlreadyDone  = errors.New("claim already submitted")
)

const claimTokenTTL = 7 * 24 * time.Hour

type RaffleService struct {
	db *gorm.DB
}

func NewRaffleService(db *gorm.DB) *RaffleService {
	return &RaffleService{db: db}
}

// Create creates a new raffle owned by the given user.
func (s *RaffleService) Create(userID uuid.UUID, title string) (*models.Raffle, error) {
	raffle := &models.Raffle{
		UserID: userID,
		Title:  title,
		Status: models.RaffleStatusDraft,
		Source: models.RaffleSourceCSV,
	}
	if err := s.db.Create(raffle).Error; err != nil {
		return nil, err
	}
	return raffle, nil
}

// GetByID returns a raffle, verifying ownership.
func (s *RaffleService) GetByID(id, userID uuid.UUID) (*models.Raffle, error) {
	var raffle models.Raffle
	if err := s.db.Where("id = ?", id).First(&raffle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRaffleNotFound
		}
		return nil, err
	}
	if raffle.UserID != userID {
		return nil, ErrRaffleForbidden
	}
	return &raffle, nil
}

// ListByStreamer returns all raffles owned by the user.
func (s *RaffleService) ListByStreamer(userID uuid.UUID) ([]models.Raffle, error) {
	var raffles []models.Raffle
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&raffles).Error; err != nil {
		return nil, err
	}
	if raffles == nil {
		return []models.Raffle{}, nil
	}
	return raffles, nil
}

// ImportCSVResult summarises the result of a CSV import.
type ImportCSVResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

// ImportCSV parses a CSV reader and inserts RaffleEntry rows.
// First column must be twitch_login; an optional second column is display_name.
// Rows whose twitch_login is already in the raffle are skipped (idempotent).
func (s *RaffleService) ImportCSV(raffleID, userID uuid.UUID, r io.Reader) (*ImportCSVResult, error) {
	if _, err := s.GetByID(raffleID, userID); err != nil {
		return nil, err
	}

	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // allow variable columns

	result := &ImportCSVResult{}

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) == 0 {
			continue
		}

		twitchLogin := strings.TrimSpace(record[0])
		if twitchLogin == "" || strings.EqualFold(twitchLogin, "twitch_login") {
			// skip empty / header row
			continue
		}

		displayName := ""
		if len(record) > 1 {
			displayName = strings.TrimSpace(record[1])
		}

		// Check for duplicate within this raffle
		var count int64
		if err := s.db.Model(&models.RaffleEntry{}).
			Where("raffle_id = ? AND twitch_login = ?", raffleID, twitchLogin).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			result.Skipped++
			continue
		}

		// Try to match a tachigo user via auth_providers
		var userIDPtr *uuid.UUID
		var provider models.AuthProvider
		if err := s.db.
			Where("provider = ? AND provider_id = ?", models.ProviderTwitch, twitchLogin).
			First(&provider).Error; err == nil {
			uid := provider.UserID
			userIDPtr = &uid
		}

		entry := &models.RaffleEntry{
			RaffleID:    raffleID,
			UserID:      userIDPtr,
			TwitchLogin: twitchLogin,
			DisplayName: displayName,
		}
		if err := s.db.Create(entry).Error; err != nil {
			return nil, err
		}
		result.Imported++
	}

	return result, nil
}

// DrawNext picks a random un-drawn entry and records a RaffleDraw.
func (s *RaffleService) DrawNext(raffleID, userID uuid.UUID) (*models.RaffleDraw, error) {
	raffle, err := s.GetByID(raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status == models.RaffleStatusCompleted {
		return nil, ErrRaffleCompleted
	}

	// Pick a random entry that has NOT been drawn yet.
	var entry models.RaffleEntry
	if err := s.db.Raw(`
		SELECT * FROM raffle_entries
		WHERE raffle_id = ?
		  AND id NOT IN (
		        SELECT entry_id FROM raffle_draws WHERE raffle_id = ?
		      )
		ORDER BY RANDOM()
		LIMIT 1
	`, raffleID, raffleID).Scan(&entry).Error; err != nil {
		return nil, err
	}
	if entry.ID == uuid.Nil {
		return nil, ErrRaffleExhausted
	}

	token, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	draw := &models.RaffleDraw{
		RaffleID:       raffleID,
		EntryID:        entry.ID,
		ClaimToken:     token.String(),
		ClaimExpiresAt: time.Now().Add(claimTokenTTL),
		DrawnAt:        time.Now(),
	}
	if err := s.db.Create(draw).Error; err != nil {
		return nil, err
	}
	draw.Entry = entry
	return draw, nil
}

// ListDraws returns all draws for a raffle (with entry preloaded).
func (s *RaffleService) ListDraws(raffleID, userID uuid.UUID) ([]models.RaffleDraw, error) {
	if _, err := s.GetByID(raffleID, userID); err != nil {
		return nil, err
	}

	var draws []models.RaffleDraw
	if err := s.db.
		Preload("Entry").
		Where("raffle_id = ?", raffleID).
		Order("drawn_at DESC").
		Find(&draws).Error; err != nil {
		return nil, err
	}
	if draws == nil {
		return []models.RaffleDraw{}, nil
	}
	return draws, nil
}

// Complete marks a raffle as completed.
func (s *RaffleService) Complete(raffleID, userID uuid.UUID) (*models.Raffle, error) {
	raffle, err := s.GetByID(raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status == models.RaffleStatusCompleted {
		return raffle, nil
	}
	if err := s.db.Model(raffle).Update("status", models.RaffleStatusCompleted).Error; err != nil {
		return nil, err
	}
	return raffle, nil
}

// GetDrawByToken fetches a draw by its claim token. Returns ErrClaimTokenExpired if past expiry.
func (s *RaffleService) GetDrawByToken(token string) (*models.RaffleDraw, error) {
	var draw models.RaffleDraw
	if err := s.db.
		Preload("Entry").
		Where("claim_token = ?", token).
		First(&draw).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClaimNotFound
		}
		return nil, err
	}
	if time.Now().After(draw.ClaimExpiresAt) {
		return nil, ErrClaimTokenExpired
	}
	return &draw, nil
}

// ClaimInput holds the shipping info submitted by the winner.
type ClaimInput struct {
	RecipientName string `json:"recipient_name" binding:"required"`
	Phone         string `json:"phone"`
	AddressLine1  string `json:"address_line1" binding:"required"`
	AddressLine2  string `json:"address_line2"`
	City          string `json:"city" binding:"required"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
}

// SubmitClaim records the winner's shipping information.
func (s *RaffleService) SubmitClaim(token string, input ClaimInput) (*models.RaffleClaim, error) {
	draw, err := s.GetDrawByToken(token)
	if err != nil {
		return nil, err
	}

	// Check if already submitted
	var count int64
	if err := s.db.Model(&models.RaffleClaim{}).
		Where("draw_id = ?", draw.ID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrClaimAlreadyDone
	}

	country := input.Country
	if country == "" {
		country = "TW"
	}

	claim := &models.RaffleClaim{
		DrawID:        draw.ID,
		RecipientName: input.RecipientName,
		Phone:         input.Phone,
		AddressLine1:  input.AddressLine1,
		AddressLine2:  input.AddressLine2,
		City:          input.City,
		PostalCode:    input.PostalCode,
		Country:       country,
		SubmittedAt:   time.Now(),
	}
	if err := s.db.Create(claim).Error; err != nil {
		return nil, err
	}
	return claim, nil
}

// GetDrawsByRafflePublic returns drawn entries for Extension display (no auth check).
func (s *RaffleService) GetDrawsByRafflePublic(raffleID uuid.UUID) ([]models.RaffleDraw, error) {
	var draws []models.RaffleDraw
	if err := s.db.
		Preload("Entry").
		Where("raffle_id = ?", raffleID).
		Order("drawn_at DESC").
		Find(&draws).Error; err != nil {
		return nil, err
	}
	if draws == nil {
		return []models.RaffleDraw{}, nil
	}
	return draws, nil
}
