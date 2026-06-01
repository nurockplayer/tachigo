package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
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

	"github.com/tachigo/tachigo/internal/metrics"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrRaffleNotFound           = errors.New("raffle not found")
	ErrRaffleForbidden          = errors.New("raffle does not belong to this user")
	ErrRaffleExhausted          = errors.New("all entries have been drawn")
	ErrRaffleCompleted          = errors.New("raffle is already completed")
	ErrRaffleNotDraft           = errors.New("raffle is not in draft status")
	ErrClaimTokenExpired        = errors.New("claim token has expired")
	ErrClaimNotFound            = errors.New("claim token not found")
	ErrClaimAlreadyDone         = errors.New("claim already submitted")
	ErrClaimForbidden           = errors.New("claim forbidden: not the winner")
	ErrTwitchTokenMissing       = errors.New("no twitch access token: streamer must log in via twitch")
	ErrTwitchInsufficientScope  = errors.New("twitch token lacks channel:read:subscriptions scope")
	ErrUnsupportedRaffleSource  = errors.New("raffle source does not support twitch sync")
	ErrInvalidDiscordWebhookURL = errors.New("invalid Discord webhook URL: must start with https://discord.com/api/webhooks/")

	ErrPrizeTierNotFound     = errors.New("prize tier not found")
	ErrPrizeTierHasDraws     = errors.New("prize tier already has draws, cannot delete")
	ErrPrizeTierExhausted    = errors.New("prize tier winner count reached")
	ErrPrizeTierInvalidCount = errors.New("winner_count must be at least 1")
	ErrRaffleNotActive       = errors.New("raffle is not active")

	ErrEntryNotOpen    = errors.New("raffle entry not open")
	ErrAlreadyJoined   = errors.New("already joined")
	ErrNotSubscriber   = errors.New("not a subscriber")
	ErrViewerNotLinked = errors.New("viewer has no linked tachigo account")
)

const claimTokenTTL = 7 * 24 * time.Hour

// hashClaimToken returns the SHA-256 hex digest of a raw claim token.
// Only the hash is stored in the DB; the raw token is returned to callers once.
func hashClaimToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

type RaffleService struct {
	db             *gorm.DB
	twitchClientID string
	twitchBaseURL  string
	httpClient     *http.Client
	mailer         Mailer
	frontendURL    string
	tokenCipher    *oauthTokenCipher
	metrics        *metrics.Collector
}

func NewRaffleService(db *gorm.DB, twitchClientID, frontendURL string, mailer Mailer, tokenEncryptionSecret ...string) *RaffleService {
	secret := ""
	if len(tokenEncryptionSecret) > 0 {
		secret = tokenEncryptionSecret[0]
	}
	return &RaffleService{
		db:             db,
		twitchClientID: twitchClientID,
		twitchBaseURL:  "https://api.twitch.tv",
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		mailer:         mailer,
		frontendURL:    frontendURL,
		tokenCipher:    newOAuthTokenCipher(secret),
	}
}

func (s *RaffleService) SetTwitchBaseURL(u string) { s.twitchBaseURL = u }
func (s *RaffleService) SetHTTPClient(c *http.Client) {
	if c == nil {
		return
	}
	s.httpClient = c
}

func (s *RaffleService) SetMetricsCollector(collector *metrics.Collector) {
	if s == nil {
		return
	}
	s.metrics = collector
}

// Create creates a new raffle owned by the given user.
func (s *RaffleService) Create(userID uuid.UUID, title string) (*models.Raffle, error) {
	return s.CreateContext(context.Background(), userID, title)
}

func (s *RaffleService) CreateContext(ctx context.Context, userID uuid.UUID, title string) (*models.Raffle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	raffle := &models.Raffle{
		UserID: userID,
		Title:  title,
		Status: models.RaffleStatusDraft,
		Source: models.RaffleSourceCSV,
	}
	if err := s.db.WithContext(ctx).Create(raffle).Error; err != nil {
		return nil, err
	}
	return raffle, nil
}

// GetByID returns a raffle, verifying ownership.
func (s *RaffleService) GetByID(id, userID uuid.UUID) (*models.Raffle, error) {
	return s.GetByIDContext(context.Background(), id, userID)
}

func (s *RaffleService) GetByIDContext(ctx context.Context, id, userID uuid.UUID) (*models.Raffle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var raffle models.Raffle
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&raffle).Error; err != nil {
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
	return s.ListByStreamerContext(context.Background(), userID)
}

func (s *RaffleService) ListByStreamerContext(ctx context.Context, userID uuid.UUID) ([]models.Raffle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var raffles []models.Raffle
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&raffles).Error; err != nil {
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
	return s.ImportCSVContext(context.Background(), raffleID, userID, r)
}

func (s *RaffleService) ImportCSVContext(ctx context.Context, raffleID, userID uuid.UUID, r io.Reader) (*ImportCSVResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	db := s.db.WithContext(ctx)

	raffle, err := s.GetByIDContext(ctx, raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status != models.RaffleStatusDraft {
		return nil, ErrRaffleNotDraft
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
		if err := db.Model(&models.RaffleEntry{}).
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
		if err := db.
			Joins("JOIN users ON users.id = auth_providers.user_id AND users.deleted_at IS NULL").
			Where("auth_providers.provider = ? AND users.username = ?", models.ProviderTwitch, twitchLogin).
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
			TwitchLogin: twitchLogin,
			DisplayName: displayName,
		}
		if err := db.Create(entry).Error; err != nil {
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

	const drawMaxRetries = 5

	// Each attempt runs in its own transaction so that a duplicate-key error on
	// PostgreSQL (which aborts the transaction) does not poison subsequent SELECTs.
	var result *models.RaffleDraw
	err = func() error {
		for attempt := 0; attempt < drawMaxRetries; attempt++ {
			var draw *models.RaffleDraw
			txErr := s.db.Transaction(func(tx *gorm.DB) error {
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

				rawToken, err := uuid.NewV7()
				if err != nil {
					return err
				}

				d := &models.RaffleDraw{
					RaffleID:       raffleID,
					EntryID:        entry.ID,
					ClaimToken:     hashClaimToken(rawToken.String()),
					ClaimExpiresAt: time.Now().Add(claimTokenTTL),
					DrawnAt:        time.Now(),
				}
				if err := tx.Create(d).Error; err != nil {
					return err
				}
				d.ClaimTokenRaw = rawToken.String()
				d.Entry = entry
				draw = d
				return nil
			})
			if txErr == nil {
				result = draw
				return nil
			}
			if errors.Is(txErr, ErrRaffleExhausted) {
				return txErr
			}
			if errors.Is(txErr, gorm.ErrDuplicatedKey) {
				continue
			}
			return txErr
		}
		return ErrRaffleExhausted
	}()
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
	if err == nil && raffle.DiscordWebhookURL != nil {
		go func(d *models.RaffleDraw, webhookURL string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("raffle sendDiscordNotification panic (draw %s): %v", d.ID, r)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.sendDiscordNotification(ctx, d, webhookURL)
		}(result, *raffle.DiscordWebhookURL)
	}
	return result, err
}

// SetDiscordWebhook sets or clears the Discord webhook URL for a raffle.
// An empty webhookURL clears the setting.
func (s *RaffleService) SetDiscordWebhook(raffleID, userID uuid.UUID, webhookURL string) (*models.Raffle, error) {
	return s.SetDiscordWebhookContext(context.Background(), raffleID, userID, webhookURL)
}

func (s *RaffleService) SetDiscordWebhookContext(ctx context.Context, raffleID, userID uuid.UUID, webhookURL string) (*models.Raffle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	raffle, err := s.GetByIDContext(ctx, raffleID, userID)
	if err != nil {
		return nil, err
	}
	var val *string
	if webhookURL != "" {
		if !strings.HasPrefix(webhookURL, "https://discord.com/api/webhooks/") &&
			!strings.HasPrefix(webhookURL, "https://discordapp.com/api/webhooks/") {
			return nil, ErrInvalidDiscordWebhookURL
		}
		val = &webhookURL
	}
	if err := s.db.WithContext(ctx).Model(raffle).Update("discord_webhook_url", val).Error; err != nil {
		return nil, err
	}
	raffle.DiscordWebhookURL = val
	return raffle, nil
}

func (s *RaffleService) sendDiscordNotification(ctx context.Context, draw *models.RaffleDraw, webhookURL string) {
	// Intentionally omit the claim token from the public webhook payload.
	// /claim/:token is an unauthenticated endpoint; posting the token to a
	// Discord channel (which may be public) would allow any viewer to submit
	// the claim on behalf of the winner. The claim link is delivered privately
	// via email (issue #230) to the winner's registered address instead.
	twitchLogin := draw.Entry.TwitchLogin
	expiry := draw.ClaimExpiresAt.Format("2006-01-02 15:04 MST")

	var content string
	if draw.PrizeTier != nil {
		content = fmt.Sprintf(
			"🎉 [%s] 抽獎結果揭曉！中獎者：**%s**\n獎品：%s\n\n請中獎者留意 Tachigo 系統通知，或聯繫實況主確認領獎方式。\n領獎資格保留至：%s。",
			draw.PrizeTier.Name, twitchLogin, draw.PrizeTier.PrizeDescription, expiry,
		)
	} else {
		content = fmt.Sprintf(
			"🎉 抽獎結果揭曉！中獎者：**%s**\n\n請中獎者留意 Tachigo 系統通知，或聯繫實況主確認領獎方式。\n領獎資格保留至：%s。",
			twitchLogin, expiry,
		)
	}
	payload := map[string]interface{}{"content": content}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("raffle draw %s: discord marshal failed: %v", draw.ID, err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("raffle draw %s: discord request build failed: %v", draw.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("raffle draw %s: discord webhook failed: %v", draw.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("raffle draw %s: discord webhook status %d", draw.ID, resp.StatusCode)
	}
}

// ListDraws returns all draws for a raffle (with entry preloaded).
func (s *RaffleService) ListDraws(raffleID, userID uuid.UUID) ([]models.RaffleDraw, error) {
	return s.ListDrawsContext(context.Background(), raffleID, userID)
}

func (s *RaffleService) ListDrawsContext(ctx context.Context, raffleID, userID uuid.UUID) ([]models.RaffleDraw, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := s.GetByIDContext(ctx, raffleID, userID); err != nil {
		return nil, err
	}

	var draws []models.RaffleDraw
	if err := s.db.WithContext(ctx).
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

// Activate locks the participant list by moving the raffle from draft to active.
// After activation, ImportCSV will reject further uploads.
// The update is conditional on status = draft to avoid a read-then-write race.
func (s *RaffleService) Activate(raffleID, userID uuid.UUID) (*models.Raffle, error) {
	return s.ActivateContext(context.Background(), raffleID, userID)
}

func (s *RaffleService) ActivateContext(ctx context.Context, raffleID, userID uuid.UUID) (*models.Raffle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	raffle, err := s.GetByIDContext(ctx, raffleID, userID)
	if err != nil {
		return nil, err
	}
	res := s.db.WithContext(ctx).Model(&models.Raffle{}).
		Where("id = ? AND status = ?", raffle.ID, models.RaffleStatusDraft).
		Update("status", models.RaffleStatusActive)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrRaffleNotDraft
	}
	raffle.Status = models.RaffleStatusActive
	return raffle, nil
}

// Complete marks a raffle as completed.
func (s *RaffleService) Complete(raffleID, userID uuid.UUID) (*models.Raffle, error) {
	return s.CompleteContext(context.Background(), raffleID, userID)
}

func (s *RaffleService) CompleteContext(ctx context.Context, raffleID, userID uuid.UUID) (*models.Raffle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	raffle, err := s.GetByIDContext(ctx, raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status == models.RaffleStatusCompleted {
		return raffle, nil
	}
	if err := s.db.WithContext(ctx).Model(raffle).Update("status", models.RaffleStatusCompleted).Error; err != nil {
		return nil, err
	}
	return raffle, nil
}

// GetDrawByToken fetches a draw by its claim token. Returns ErrClaimTokenExpired if past expiry.
func (s *RaffleService) GetDrawByToken(token string) (*models.RaffleDraw, error) {
	return s.GetDrawByTokenContext(context.Background(), token)
}

func (s *RaffleService) GetDrawByTokenContext(ctx context.Context, token string) (*models.RaffleDraw, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var draw models.RaffleDraw
	if err := s.db.WithContext(ctx).
		Preload("Entry").
		Where("claim_token = ?", hashClaimToken(token)).
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
// userID must match the linked user on the winning entry; otherwise ErrClaimForbidden is returned.
func (s *RaffleService) SubmitClaim(token string, userID uuid.UUID, input ClaimInput) (*models.RaffleClaim, error) {
	return s.SubmitClaimContext(context.Background(), token, userID, input)
}

func (s *RaffleService) SubmitClaimContext(ctx context.Context, token string, userID uuid.UUID, input ClaimInput) (*models.RaffleClaim, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	draw, err := s.GetDrawByTokenContext(ctx, token)
	if err != nil {
		return nil, err
	}

	if draw.Entry.UserID == nil || *draw.Entry.UserID != userID {
		return nil, ErrClaimForbidden
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
	if err := s.db.WithContext(ctx).Create(claim).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrClaimAlreadyDone
		}
		return nil, err
	}
	return claim, nil
}

// GetDrawsByRafflePublic returns drawn entries for Extension display (no auth check).
func (s *RaffleService) GetDrawsByRafflePublic(raffleID uuid.UUID) ([]models.RaffleDraw, error) {
	return s.GetDrawsByRafflePublicContext(context.Background(), raffleID)
}

func (s *RaffleService) GetDrawsByRafflePublicContext(ctx context.Context, raffleID uuid.UUID) ([]models.RaffleDraw, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var draws []models.RaffleDraw
	if err := s.db.WithContext(ctx).
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

func (s *RaffleService) fetchTwitchSubscription(ctx context.Context, accessToken, broadcasterID, userID string) (bool, error) {
	endpoint, err := url.Parse(strings.TrimRight(s.twitchBaseURL, "/") + "/helix/subscriptions")
	if err != nil {
		return false, err
	}
	q := endpoint.Query()
	q.Set("broadcaster_id", broadcasterID)
	q.Set("user_id", userID)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", s.twitchClientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, ErrTwitchTokenMissing
	}
	if resp.StatusCode == http.StatusForbidden {
		return false, ErrTwitchInsufficientScope
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("twitch api: unexpected status %d", resp.StatusCode)
	}

	var page twitchSubsPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return false, err
	}
	return len(page.Data) > 0, nil
}

func (s *RaffleService) JoinRaffle(ctx context.Context, raffleID uuid.UUID, twitchUserID string) (*models.RaffleEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	db := s.db.WithContext(ctx)

	var raffle models.Raffle
	if err := db.First(&raffle, "id = ?", raffleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRaffleNotFound
		}
		return nil, err
	}
	if !raffle.EntryOpen {
		return nil, ErrEntryNotOpen
	}

	var viewerAP models.AuthProvider
	if err := db.
		Joins("JOIN users ON users.id = auth_providers.user_id AND users.deleted_at IS NULL").
		Preload("User").
		Where("auth_providers.provider = ? AND auth_providers.provider_id = ?", models.ProviderTwitch, twitchUserID).
		First(&viewerAP).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrViewerNotLinked
		}
		return nil, err
	}

	uid := viewerAP.UserID
	twitchLogin := twitchUserID
	if viewerAP.User.Username != nil {
		twitchLogin = *viewerAP.User.Username
	}
	entry := &models.RaffleEntry{
		RaffleID:    raffleID,
		UserID:      &uid,
		TwitchLogin: twitchLogin,
		DisplayName: twitchLogin,
		Source:      string(models.RaffleSourceExtensionButton),
		Eligible:    true,
	}

	if raffle.Mode == models.RaffleModeSubscribers {
		var broadcasterAP models.AuthProvider
		if err := db.Where("user_id = ? AND provider = ?", raffle.UserID, models.ProviderTwitch).First(&broadcasterAP).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTwitchTokenMissing
			}
			return nil, err
		}
		accessToken, err := s.decryptedProviderAccessToken(&broadcasterAP)
		if err != nil {
			return nil, err
		}
		if accessToken == "" {
			s.clearProviderTokensContext(ctx, broadcasterAP.ID)
			return nil, ErrTwitchTokenMissing
		}
		if broadcasterAP.TokenExpiresAt != nil && time.Now().After(*broadcasterAP.TokenExpiresAt) {
			s.clearProviderTokensContext(ctx, broadcasterAP.ID)
			return nil, ErrTwitchTokenMissing
		}
		ok, err := s.fetchTwitchSubscription(ctx, accessToken, broadcasterAP.ProviderID, twitchUserID)
		if err != nil {
			if errors.Is(err, ErrTwitchInsufficientScope) {
				s.clearProviderTokensContext(ctx, broadcasterAP.ID)
			}
			return nil, err
		}
		if !ok {
			// 非訂閱者不寫入 DB，直接拒絕
			return nil, ErrNotSubscriber
		}
	}

	res := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "raffle_id"}, {Name: "twitch_login"}},
		DoNothing: true,
	}).Create(entry)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrAlreadyJoined
	}
	return entry, nil
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
	if ctx == nil {
		ctx = context.Background()
	}

	db := s.db.WithContext(ctx)
	raffle, err := s.GetByIDContext(ctx, raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Source != models.RaffleSourceTwitchAPI {
		return nil, ErrUnsupportedRaffleSource
	}

	var ap models.AuthProvider
	if err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderTwitch).First(&ap).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTwitchTokenMissing
		}
		return nil, err
	}
	accessToken, err := s.decryptedProviderAccessToken(&ap)
	if err != nil {
		return nil, err
	}
	if accessToken == "" {
		return nil, ErrTwitchTokenMissing
	}
	if ap.TokenExpiresAt != nil && time.Now().After(*ap.TokenExpiresAt) {
		s.clearProviderTokensContext(ctx, ap.ID)
		return nil, ErrTwitchTokenMissing
	}

	result := &SyncFromTwitchResult{}
	cursor := ""
	for {
		subs, nextCursor, err := s.fetchTwitchSubsPage(ctx, accessToken, ap.ProviderID, cursor)
		if err != nil {
			if errors.Is(err, ErrTwitchInsufficientScope) {
				s.clearProviderTokensContext(ctx, ap.ID)
			}
			return nil, err
		}

		for _, sub := range subs {
			var provider models.AuthProvider
			if err := db.
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
			res := db.Clauses(clause.OnConflict{
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

func (s *RaffleService) decryptedProviderAccessToken(ap *models.AuthProvider) (string, error) {
	if ap == nil || ap.AccessToken == nil {
		return "", nil
	}
	cipher := s.tokenCipher
	if cipher == nil {
		cipher = newOAuthTokenCipher("")
	}
	return cipher.decrypt(*ap.AccessToken)
}

func (s *RaffleService) clearProviderTokensContext(ctx context.Context, providerID uuid.UUID) {
	if s == nil || s.db == nil || providerID == uuid.Nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cleanupCtx := context.WithoutCancel(ctx)
	_ = s.db.WithContext(cleanupCtx).Model(&models.AuthProvider{}).
		Where("id = ?", providerID).
		Updates(map[string]interface{}{
			"access_token":     nil,
			"refresh_token":    nil,
			"token_expires_at": nil,
		}).Error
}

// ── Scheduled snapshot (cron) ─────────────────────────────────────────────────

const scheduledSnapshotBatchLimit = 500

// RunScheduledSnapshots finds draft raffles whose scheduled_at is due — already
// elapsed or within the next 10 minutes — and triggers their snapshot. There is
// intentionally no lower bound, so a raffle whose scheduled_at passed while the
// server was down is caught up on the next scan rather than skipped forever.
// Per-raffle errors are logged and do not abort the batch. CSV raffles are
// excluded: they are uploaded manually and have no remote source to sync from.
func (s *RaffleService) RunScheduledSnapshots(ctx context.Context, now time.Time) error {
	start := time.Now()
	result := "success"
	defer func() {
		if s != nil && s.metrics != nil {
			s.metrics.ObserveRaffleSchedulerRun(result, time.Since(start))
		}
	}()

	window := now.Add(10 * time.Minute)
	var raffles []models.Raffle
	if err := s.db.WithContext(ctx).Where(
		"status = ? AND source != ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?",
		models.RaffleStatusDraft, models.RaffleSourceCSV, window,
	).Order("scheduled_at ASC, id ASC").Limit(scheduledSnapshotBatchLimit).Find(&raffles).Error; err != nil {
		log.Printf(
			"event=raffle_scheduled_snapshots_query_error job=raffle_scheduled_snapshots run_at=%s window_end=%s err=%q",
			now.Format(time.RFC3339),
			window.Format(time.RFC3339),
			err,
		)
		result = "failure"
		return err
	}
	log.Printf(
		"event=raffle_scheduled_snapshots_start job=raffle_scheduled_snapshots run_at=%s window_end=%s candidate_count=%d",
		now.Format(time.RFC3339),
		window.Format(time.RFC3339),
		len(raffles),
	)
	failed := 0
	for _, r := range raffles {
		if err := s.snapshotOne(ctx, r); err != nil {
			failed++
			log.Printf(
				"event=raffle_scheduled_snapshot_error job=raffle_scheduled_snapshots raffle_id=%s user_id=%s source=%s err=%s",
				r.ID,
				r.UserID,
				r.Source,
				safeScheduledJobError(err),
			)
		}
	}
	switch {
	case failed == 0:
		result = "success"
	case failed == len(raffles):
		result = "failure"
	default:
		result = "partial_failure"
	}
	log.Printf(
		"event=raffle_scheduled_snapshots_complete job=raffle_scheduled_snapshots run_at=%s window_end=%s candidate_count=%d processed=%d failed=%d",
		now.Format(time.RFC3339),
		window.Format(time.RFC3339),
		len(raffles),
		len(raffles),
		failed,
	)
	return nil
}

func safeScheduledJobError(err error) string {
	if err == nil {
		return "-"
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	for _, marker := range []string{
		"access_token",
		"refresh_token",
		"authorization",
		"bearer ",
		"client_secret",
		"private_key",
		"signer_key",
		"voucher",
		"receipt",
	} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	msg = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return ' '
		default:
			return r
		}
	}, msg)
	if len(msg) > 512 {
		return msg[:512]
	}
	return fmt.Sprintf("%q", msg)
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
	return s.db.WithContext(ctx).Model(&models.Raffle{}).
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
	link := fmt.Sprintf("%s/claim/%s", strings.TrimRight(s.frontendURL, "/"), draw.ClaimTokenRaw)
	body := raffleWinnerEmailBody(draw.ClaimExpiresAt, link)
	if err := s.mailer.Send(ctx, *user.Email, "恭喜中獎！領取你的 Tachigo 抽獎獎品", body); err != nil {
		log.Printf("raffle draw %s: failed to send winner email to %s: %v", draw.ID, *user.Email, err)
	}
}

// ── Prize Tier CRUD ───────────────────────────────────────────────────────────

type CreatePrizeTierInput struct {
	Name             string `json:"name"`
	PrizeDescription string `json:"prize_description"`
	WinnerCount      int    `json:"winner_count"`
}

type UpdatePrizeTierInput struct {
	Name             *string `json:"name,omitempty"`
	PrizeDescription *string `json:"prize_description,omitempty"`
	WinnerCount      *int    `json:"winner_count,omitempty"`
}

func (s *RaffleService) CreatePrizeTier(raffleID, userID uuid.UUID, input CreatePrizeTierInput) (*models.RafflePrizeTier, error) {
	return s.CreatePrizeTierContext(context.Background(), raffleID, userID, input)
}

func (s *RaffleService) CreatePrizeTierContext(ctx context.Context, raffleID, userID uuid.UUID, input CreatePrizeTierInput) (*models.RafflePrizeTier, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if input.WinnerCount < 1 {
		return nil, ErrPrizeTierInvalidCount
	}
	if _, err := s.GetByIDContext(ctx, raffleID, userID); err != nil {
		return nil, err
	}

	db := s.db.WithContext(ctx)
	var maxPos int
	if err := db.Model(&models.RafflePrizeTier{}).
		Where("raffle_id = ?", raffleID).
		Select("COALESCE(MAX(position), 0)").
		Scan(&maxPos).Error; err != nil {
		return nil, err
	}

	tier := &models.RafflePrizeTier{
		RaffleID:         raffleID,
		Name:             input.Name,
		PrizeDescription: input.PrizeDescription,
		WinnerCount:      input.WinnerCount,
		Position:         maxPos + 1,
	}
	if err := db.Create(tier).Error; err != nil {
		return nil, err
	}
	return tier, nil
}

func (s *RaffleService) ListPrizeTiers(raffleID, userID uuid.UUID) ([]models.RafflePrizeTier, error) {
	return s.ListPrizeTiersContext(context.Background(), raffleID, userID)
}

func (s *RaffleService) ListPrizeTiersContext(ctx context.Context, raffleID, userID uuid.UUID) ([]models.RafflePrizeTier, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := s.GetByIDContext(ctx, raffleID, userID); err != nil {
		return nil, err
	}
	var tiers []models.RafflePrizeTier
	if err := s.db.WithContext(ctx).
		Where("raffle_id = ?", raffleID).
		Order("position ASC").
		Find(&tiers).Error; err != nil {
		return nil, err
	}
	return tiers, nil
}

func (s *RaffleService) UpdatePrizeTier(raffleID, tierID, userID uuid.UUID, input UpdatePrizeTierInput) (*models.RafflePrizeTier, error) {
	return s.UpdatePrizeTierContext(context.Background(), raffleID, tierID, userID, input)
}

func (s *RaffleService) UpdatePrizeTierContext(ctx context.Context, raffleID, tierID, userID uuid.UUID, input UpdatePrizeTierInput) (*models.RafflePrizeTier, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := s.GetByIDContext(ctx, raffleID, userID); err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	var tier models.RafflePrizeTier
	if err := db.Where("id = ? AND raffle_id = ?", tierID, raffleID).First(&tier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPrizeTierNotFound
		}
		return nil, err
	}
	if input.WinnerCount != nil {
		if *input.WinnerCount < 1 || *input.WinnerCount < tier.DrawnCount {
			return nil, ErrPrizeTierInvalidCount
		}
		tier.WinnerCount = *input.WinnerCount
	}
	if input.Name != nil {
		tier.Name = *input.Name
	}
	if input.PrizeDescription != nil {
		tier.PrizeDescription = *input.PrizeDescription
	}
	if err := db.Save(&tier).Error; err != nil {
		return nil, err
	}
	return &tier, nil
}

func (s *RaffleService) DeletePrizeTier(raffleID, tierID, userID uuid.UUID) error {
	return s.DeletePrizeTierContext(context.Background(), raffleID, tierID, userID)
}

func (s *RaffleService) DeletePrizeTierContext(ctx context.Context, raffleID, tierID, userID uuid.UUID) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := s.GetByIDContext(ctx, raffleID, userID); err != nil {
		return err
	}
	db := s.db.WithContext(ctx)
	// Verify tier exists first (to distinguish not-found from has-draws).
	var tier models.RafflePrizeTier
	if err := db.Where("id = ? AND raffle_id = ?", tierID, raffleID).First(&tier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPrizeTierNotFound
		}
		return err
	}
	// Conditional atomic delete: only removes if drawn_count = 0, closing the TOCTOU
	// between a drawn_count check and the delete.
	res := db.Where("id = ? AND raffle_id = ? AND drawn_count = 0", tierID, raffleID).
		Delete(&models.RafflePrizeTier{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPrizeTierHasDraws
	}
	return nil
}

func (s *RaffleService) DrawFromTier(raffleID, tierID, userID uuid.UUID) (*models.RaffleDraw, error) {
	return s.DrawFromTierContext(context.Background(), raffleID, tierID, userID)
}

// DrawFromTierContext picks a random un-drawn entry for the specified prize tier.
// Winners drawn for any tier of this raffle are excluded from the pool.
// drawn_count is incremented via a conditional atomic UPDATE (WHERE drawn_count < winner_count)
// so that concurrent calls cannot exceed the quota.
func (s *RaffleService) DrawFromTierContext(ctx context.Context, raffleID, tierID, userID uuid.UUID) (*models.RaffleDraw, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	raffle, err := s.GetByIDContext(ctx, raffleID, userID)
	if err != nil {
		return nil, err
	}
	if raffle.Status != models.RaffleStatusActive {
		return nil, ErrRaffleNotActive
	}

	db := s.db.WithContext(ctx)
	var tier models.RafflePrizeTier
	if err := db.Where("id = ? AND raffle_id = ?", tierID, raffleID).First(&tier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPrizeTierNotFound
		}
		return nil, err
	}
	if tier.DrawnCount >= tier.WinnerCount {
		return nil, ErrPrizeTierExhausted
	}

	const drawMaxRetries = 5
	var result *models.RaffleDraw

	err = func() error {
		for attempt := 0; attempt < drawMaxRetries; attempt++ {
			var draw *models.RaffleDraw
			txErr := db.Transaction(func(tx *gorm.DB) error {
				// Exclude ALL winners across ALL tiers of this raffle.
				var wonIDs []uuid.UUID
				if err := tx.Model(&models.RaffleDraw{}).
					Where("raffle_id = ?", raffleID).
					Pluck("entry_id", &wonIDs).Error; err != nil {
					return err
				}

				var entry models.RaffleEntry
				q := tx.Where("raffle_id = ?", raffleID)
				if len(wonIDs) > 0 {
					q = q.Where("id NOT IN ?", wonIDs)
				}
				if err := q.Order("RANDOM()").First(&entry).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return ErrRaffleExhausted
					}
					return err
				}

				rawToken, err := uuid.NewV7()
				if err != nil {
					return err
				}
				tierIDCopy := tierID
				d := &models.RaffleDraw{
					RaffleID:       raffleID,
					EntryID:        entry.ID,
					ClaimToken:     hashClaimToken(rawToken.String()),
					ClaimExpiresAt: time.Now().Add(claimTokenTTL),
					DrawnAt:        time.Now(),
					PrizeTierID:    &tierIDCopy,
				}
				if err := tx.Create(d).Error; err != nil {
					return err
				}
				// Atomic conditional increment: only succeeds if drawn_count < winner_count,
				// preventing concurrent draws from exceeding the quota without a row lock.
				res := tx.Model(&models.RafflePrizeTier{}).
					Where("id = ? AND drawn_count < winner_count", tierID).
					UpdateColumn("drawn_count", gorm.Expr("drawn_count + 1"))
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					return ErrPrizeTierExhausted
				}
				d.ClaimTokenRaw = rawToken.String()
				d.Entry = entry
				d.PrizeTier = &tier
				d.PrizeTier.DrawnCount = tier.DrawnCount + 1
				draw = d
				return nil
			})
			if txErr == nil {
				result = draw
				return nil
			}
			if errors.Is(txErr, ErrRaffleExhausted) || errors.Is(txErr, ErrPrizeTierExhausted) {
				return txErr
			}
			if errors.Is(txErr, gorm.ErrDuplicatedKey) {
				continue
			}
			return txErr
		}
		return ErrRaffleExhausted
	}()

	if err != nil {
		return nil, err
	}

	if s.mailer != nil {
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
	if raffle.DiscordWebhookURL != nil {
		go func(d *models.RaffleDraw, webhookURL string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("raffle sendDiscordNotification panic (draw %s): %v", d.ID, r)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.sendDiscordNotification(ctx, d, webhookURL)
		}(result, *raffle.DiscordWebhookURL)
	}
	return result, nil
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
