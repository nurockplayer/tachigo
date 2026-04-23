package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrRaffleNotFound          = errors.New("raffle not found")
	ErrRaffleForbidden         = errors.New("raffle does not belong to this user")
	ErrRaffleExhausted         = errors.New("all entries have been drawn")
	ErrRaffleCompleted         = errors.New("raffle is already completed")
	ErrClaimTokenExpired       = errors.New("claim token has expired")
	ErrClaimNotFound           = errors.New("claim token not found")
	ErrClaimAlreadyDone        = errors.New("claim already submitted")
	ErrTwitchTokenMissing       = errors.New("no twitch access token: streamer must log in via twitch")
	ErrTwitchInsufficientScope  = errors.New("twitch token lacks channel:read:subscriptions scope")
	ErrUnsupportedRaffleSource  = errors.New("raffle source does not support twitch sync")
)

const claimTokenTTL = 7 * 24 * time.Hour

type RaffleService struct {
	db             *gorm.DB
	twitchClientID string
	twitchBaseURL  string
	httpClient     *http.Client
	mailer         Mailer
	frontendURL    string
}

func NewRaffleService(db *gorm.DB, twitchClientID, frontendURL string, mailer Mailer) *RaffleService {
	return &RaffleService{
		db:             db,
		twitchClientID: twitchClientID,
		twitchBaseURL:  "https://api.twitch.tv",
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		mailer:         mailer,
		frontendURL:    frontendURL,
	}
}

func (s *RaffleService) SetTwitchBaseURL(u string) { s.twitchBaseURL = u }

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

		// Only import users who have a tachigo account linked to this Twitch login.
		var provider models.AuthProvider
		if err := s.db.
			Joins("JOIN users ON users.id = auth_providers.user_id AND users.deleted_at IS NULL").
			Where("auth_providers.provider = ? AND users.username = ?", models.ProviderTwitch, twitchLogin).
			First(&provider).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) { return nil, err }
			result.Skipped++
			continue
		}
		uid := provider.UserID

		entry := &models.RaffleEntry{
			RaffleID:    raffleID,
			UserID:      &uid,
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
// The SELECT+INSERT runs inside a transaction so concurrent draws cannot pick
// the same entry; the unique constraint on (raffle_id, entry_id) provides an
// additional DB-level guard.
func (s *RaffleService) DrawNext(raffleID, userID uuid.UUID) (*models.RaffleDraw, error) {
	raffle, err := s.GetByID(raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status == models.RaffleStatusCompleted {
		return nil, ErrRaffleCompleted
	}

	var result *models.RaffleDraw
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var entry models.RaffleEntry
		if err := tx.Raw(`
			SELECT * FROM raffle_entries
			WHERE raffle_id = ?
			  AND id NOT IN (
			        SELECT entry_id FROM raffle_draws WHERE raffle_id = ?
			      )
			ORDER BY RANDOM()
			LIMIT 1
		`, raffleID, raffleID).Scan(&entry).Error; err != nil {
			return err
		}
		if entry.ID == uuid.Nil {
			return ErrRaffleExhausted
		}

		token, err := uuid.NewV7()
		if err != nil {
			return err
		}

		draw := &models.RaffleDraw{
			RaffleID:       raffleID,
			EntryID:        entry.ID,
			ClaimToken:     token.String(),
			ClaimExpiresAt: time.Now().Add(claimTokenTTL),
			DrawnAt:        time.Now(),
		}
		if err := tx.Create(draw).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrRaffleExhausted
			}
			return err
		}
		draw.Entry = entry
		result = draw
		return nil
	})
	if err == nil && s.mailer != nil {
		go func(d *models.RaffleDraw) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("raffle sendWinnerEmail panic (draw %s): %v", d.ID, r)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.sendWinnerEmail(ctx, d)
		}(result)
	}
	return result, err
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
// Duplicate submissions are caught by the unique constraint on draw_id.
func (s *RaffleService) SubmitClaim(token string, input ClaimInput) (*models.RaffleClaim, error) {
	draw, err := s.GetDrawByToken(token)
	if err != nil {
		return nil, err
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
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrClaimAlreadyDone
		}
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

// ── Twitch API sync ───────────────────────────────────────────────────────────

type SyncFromTwitchResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type twitchSubscription struct {
	UserID    string `json:"user_id"`
	UserLogin string `json:"user_login"`
	UserName  string `json:"user_name"`
}

type twitchSubsPage struct {
	Data       []twitchSubscription `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

func (s *RaffleService) fetchTwitchSubsPage(ctx context.Context, accessToken, broadcasterID, cursor string) ([]twitchSubscription, string, error) {
	endpoint, err := url.Parse(strings.TrimRight(s.twitchBaseURL, "/") + "/helix/subscriptions")
	if err != nil {
		return nil, "", err
	}
	q := endpoint.Query()
	q.Set("broadcaster_id", broadcasterID)
	q.Set("first", "100")
	if cursor != "" {
		q.Set("after", cursor)
	}
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", s.twitchClientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, "", ErrTwitchInsufficientScope
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("twitch api: unexpected status %d", resp.StatusCode)
	}

	var page twitchSubsPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, "", err
	}
	return page.Data, page.Pagination.Cursor, nil
}

// SyncFromTwitchAPI pulls the broadcaster's subscriber list from Twitch Helix
// and inserts them as RaffleEntry rows (idempotent; duplicates are skipped).
// Only subscribers who already have a tachigo account are imported.
func (s *RaffleService) SyncFromTwitchAPI(ctx context.Context, raffleID, userID uuid.UUID) (*SyncFromTwitchResult, error) {
	raffle, err := s.GetByID(raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Source != models.RaffleSourceTwitchAPI {
		return nil, ErrUnsupportedRaffleSource
	}

	var ap models.AuthProvider
	if err := s.db.Where("user_id = ? AND provider = ?", userID, models.ProviderTwitch).First(&ap).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTwitchTokenMissing
		}
		return nil, err
	}
	if ap.AccessToken == nil {
		return nil, ErrTwitchTokenMissing
	}

	result := &SyncFromTwitchResult{}
	cursor := ""
	for {
		subs, nextCursor, err := s.fetchTwitchSubsPage(ctx, *ap.AccessToken, ap.ProviderID, cursor)
		if err != nil {
			return nil, err
		}

		for _, sub := range subs {
			var provider models.AuthProvider
			if err := s.db.
				Joins("JOIN users ON users.id = auth_providers.user_id AND users.deleted_at IS NULL").
				Where("auth_providers.provider = ? AND auth_providers.provider_id = ?", models.ProviderTwitch, sub.UserID).
				First(&provider).Error; err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, err
				}
				result.Skipped++
				continue
			}

			uid := provider.UserID
			entry := &models.RaffleEntry{
				RaffleID:    raffleID,
				UserID:      &uid,
				TwitchLogin: sub.UserLogin,
				DisplayName: sub.UserName,
			}
			res := s.db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "raffle_id"}, {Name: "twitch_login"}},
				DoNothing: true,
			}).Create(entry)
			if res.Error != nil {
				return nil, res.Error
			}
			if res.RowsAffected == 0 {
				result.Skipped++
				continue
			}
			result.Imported++
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return result, nil
}

// ── Scheduled snapshot (cron) ─────────────────────────────────────────────────

// RunScheduledSnapshots finds draft raffles with scheduled_at in [now, now+10min]
// and triggers their snapshot. Per-raffle errors are logged and do not abort the batch.
// CSV raffles are excluded: they are uploaded manually and have no remote source to sync from.
func (s *RaffleService) RunScheduledSnapshots(ctx context.Context, now time.Time) error {
	window := now.Add(10 * time.Minute)
	var raffles []models.Raffle
	if err := s.db.Where(
		"status = ? AND source != ? AND scheduled_at IS NOT NULL AND scheduled_at >= ? AND scheduled_at <= ?",
		models.RaffleStatusDraft, models.RaffleSourceCSV, now, window,
	).Find(&raffles).Error; err != nil {
		return err
	}
	for _, r := range raffles {
		if err := s.snapshotOne(ctx, r); err != nil {
			log.Printf("raffle %s snapshot error: %v", r.ID, err)
		}
	}
	return nil
}

func (s *RaffleService) snapshotOne(ctx context.Context, r models.Raffle) error {
	switch r.Source {
	case models.RaffleSourceTwitchAPI:
		if _, err := s.SyncFromTwitchAPI(ctx, r.ID, r.UserID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported snapshot source: %s", r.Source)
	}
	// Guard against concurrent status changes: only promote if still draft.
	return s.db.Model(&models.Raffle{}).
		Where("id = ? AND status = ?", r.ID, models.RaffleStatusDraft).
		Update("status", models.RaffleStatusActive).Error
}

// ── Winner email notification ─────────────────────────────────────────────────

// sendWinnerEmail is called async after DrawNext succeeds.
// If the winner has no linked user account or no email, it logs and skips silently.
func (s *RaffleService) sendWinnerEmail(ctx context.Context, draw *models.RaffleDraw) {
	if draw.Entry.UserID == nil {
		log.Printf("raffle draw %s: winner has no user_id, skipping email", draw.ID)
		return
	}
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", *draw.Entry.UserID).Error; err != nil {
		log.Printf("raffle draw %s: failed to look up winner user: %v", draw.ID, err)
		return
	}
	if user.Email == nil {
		log.Printf("raffle draw %s: winner has no email, skipping", draw.ID)
		return
	}
	link := fmt.Sprintf("%s/claim/%s", s.frontendURL, draw.ClaimToken)
	body := raffleWinnerEmailBody(draw.ClaimExpiresAt, link)
	if err := s.mailer.Send(*user.Email, "恭喜中獎！領取你的 Tachigo 抽獎獎品", body); err != nil {
		log.Printf("raffle draw %s: failed to send winner email to %s: %v", draw.ID, *user.Email, err)
	}
}

func raffleWinnerEmailBody(expiresAt time.Time, claimLink string) string {
	expiry := expiresAt.UTC().Format("2006-01-02 15:04 UTC")
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;color:#333;max-width:480px;margin:auto;padding:24px">
  <h2>恭喜中獎！</h2>
  <p>你已在 Tachigo 抽獎中中獎！請點擊下方按鈕填寫收件資訊以領取獎品。</p>
  <p>領獎期限：<strong>%s</strong></p>
  <p style="margin:32px 0">
    <a href="%s" style="background:#6441a5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:bold">
      領取獎品
    </a>
  </p>
  <p style="font-size:12px;color:#999">若你認為這封郵件有誤，請忽略。</p>
</body>
</html>`, expiry, claimLink)
}
