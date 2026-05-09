package services

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/models"
)

// staleThreshold is how long without a heartbeat before a session is considered stale.
const staleThreshold = 2 * time.Minute

// clickCooldown is how long a viewer must wait between clicks.
const clickCooldown = 5 * time.Second

// clickPointsPerClick is the fixed reward per click (MVP; configurable in future).
const clickPointsPerClick = int64(1)

// newUUID generates a time-ordered UUID v7. Falls back to random v4 on failure.
func newUUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

var ErrNoActiveSession = errors.New("no active session")

// ErrClickOnCooldown is returned when a viewer clicks before their cooldown expires.
type ErrClickOnCooldown struct {
	RetryAfterMs int64
}

func (e ErrClickOnCooldown) Error() string { return "click on cooldown" }

type WatchService struct {
	db *gorm.DB
}

func NewWatchService(db *gorm.DB) *WatchService {
	return &WatchService{db: db}
}

// StartSession returns or creates an active watch session.
//   - active and not stale → return existing session
//   - active but stale     → close old session, create new
//   - no active session    → create new
//
// The entire lookup + conditional close + create runs inside a single transaction
// with a row-level lock so concurrent requests cannot race into the partial-unique
// index constraint on (user_id, channel_id) WHERE is_active = true.
func (s *WatchService) StartSession(userID uuid.UUID, channelID string) (*models.WatchSession, error) {
	var result models.WatchSession

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session models.WatchSession
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND channel_id = ? AND is_active = true", userID, channelID).
			First(&session).Error

		if err == nil {
			if time.Since(session.LastHeartbeatAt) <= staleThreshold {
				result = session
				return nil
			}
			// Stale: close it before opening a new one.
			now := time.Now()
			if err := tx.Model(&session).Updates(map[string]interface{}{
				"is_active": false,
				"ended_at":  now,
			}).Error; err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		newSession := models.WatchSession{
			ID:              newUUID(),
			UserID:          userID,
			ChannelID:       channelID,
			LastHeartbeatAt: time.Now(),
			IsActive:        true,
		}
		// Use a savepoint so that if Create fails (e.g. another concurrent request
		// just won the race and inserted first), we can roll back to a clean state
		// within the same transaction and re-query instead of aborting entirely.
		// Without this, PostgreSQL marks the transaction as aborted on any error,
		// making subsequent queries impossible.
		tx.SavePoint("sp_new_session")
		if err := tx.Create(&newSession).Error; err != nil {
			tx.RollbackTo("sp_new_session")
			var existing models.WatchSession
			if qErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND channel_id = ? AND is_active = true", userID, channelID).
				First(&existing).Error; qErr == nil {
				result = existing
				return nil
			}
			return err
		}
		result = newSession
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HeartbeatResult contains the outcome of a single heartbeat call.
type HeartbeatResult struct {
	Session      *models.WatchSession
	PointsEarned int64
	DeltaSeconds int64
}

// Heartbeat advances the session timer and awards points if a full minute has accumulated.
//
// Concurrency protections:
//   - SELECT FOR UPDATE on the session row prevents two concurrent heartbeats from
//     reading the same state and double-awarding points.
//   - PointsLedger is updated via an atomic INSERT … ON CONFLICT DO UPDATE, which
//     prevents balance overwrites under concurrent writes.
func (s *WatchService) Heartbeat(userID uuid.UUID, channelID string) (*HeartbeatResult, error) {
	var result HeartbeatResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row so concurrent heartbeats queue up rather than racing.
		var session models.WatchSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND channel_id = ? AND is_active = true", userID, channelID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoActiveSession
			}
			return err
		}

		now := time.Now()
		delta := now.Sub(session.LastHeartbeatAt)

		// Ignore heartbeats that arrive too quickly — likely a client retry.
		// Normal cadence is 30 s; anything under 20 s is anomalous.
		if delta < 20*time.Second {
			result = HeartbeatResult{Session: &session, PointsEarned: 0, DeltaSeconds: 0}
			return nil
		}

		if delta > 30*time.Second {
			delta = 30 * time.Second
		}
		// Use integer division on the duration to avoid float truncation issues
		// (e.g. 29.9s should count as 29s, not silently floor via float cast).
		deltaSeconds := int64(delta / time.Second)
		cfg, err := s.getChannelConfig(tx, channelID)
		if err != nil {
			return err
		}
		secondsPerPoint := cfg.SecondsPerPoint
		multiplier := cfg.Multiplier

		if session.AccumulatedSeconds > math.MaxInt64-deltaSeconds {
			return ErrPointsDeltaOverflow
		}
		newAccumulated := session.AccumulatedSeconds + deltaSeconds
		pendingSeconds := newAccumulated - session.RewardedSeconds
		basePoints := pendingSeconds / secondsPerPoint
		if basePoints > 0 && multiplier > math.MaxInt64/basePoints {
			return ErrPointsDeltaOverflow
		}
		pointsToAward := basePoints * multiplier
		newRewarded := session.RewardedSeconds + basePoints*secondsPerPoint

		if err := tx.Model(&session).Updates(map[string]interface{}{
			"accumulated_seconds": newAccumulated,
			"rewarded_seconds":    newRewarded,
			"last_heartbeat_at":   now,
		}).Error; err != nil {
			return err
		}
		session.AccumulatedSeconds = newAccumulated
		session.RewardedSeconds = newRewarded
		session.LastHeartbeatAt = now

		if pointsToAward > 0 {
			// Atomic upsert: avoid read-modify-write by letting the database do the arithmetic.
			// UUID and timestamps are passed as parameters for SQLite test compatibility
			// (gen_random_uuid() and NOW() are PostgreSQL-only).
			ledgerID := newUUID()
			upsertTime := time.Now()
			if err := tx.Exec(`
				INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (user_id, channel_id) DO UPDATE SET
					spendable_balance = points_ledgers.spendable_balance + EXCLUDED.spendable_balance,
					cumulative_total  = points_ledgers.cumulative_total  + EXCLUDED.cumulative_total,
					updated_at        = ?
			`, ledgerID, userID, channelID, pointsToAward, pointsToAward, upsertTime, upsertTime, upsertTime).Error; err != nil {
				return err
			}

			// Fetch the balance that was just written (within the same tx for consistency).
			var ledger models.PointsLedger
			if err := tx.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&ledger).Error; err != nil {
				return err
			}

			txRecord := &models.PointsTransaction{
				LedgerID:       ledger.ID,
				WatchSessionID: &session.ID,
				Source:         models.TxSourceWatchTime,
				Delta:          pointsToAward,
				BalanceAfter:   ledger.SpendableBalance,
			}
			if err := tx.Create(txRecord).Error; err != nil {
				return err
			}
		}

		result = HeartbeatResult{Session: &session, PointsEarned: pointsToAward, DeltaSeconds: deltaSeconds}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// EndSession marks the viewer's active session in the given channel as ended.
// If no active session exists the call is a no-op (idempotent).
func (s *WatchService) EndSession(userID uuid.UUID, channelID string) error {
	now := time.Now()
	return s.db.Model(&models.WatchSession{}).
		Where("user_id = ? AND channel_id = ? AND is_active = true", userID, channelID).
		Updates(map[string]interface{}{
			"is_active": false,
			"ended_at":  now,
		}).Error
}

// ClickResult contains the outcome of a single click event.
type ClickResult struct {
	BalanceAfter int64
	Delta        int64
}

// RecordClick awards points for a viewer clicking the mining character.
//
// Rules:
//   - The viewer must have an active watch session.
//   - A per-viewer cooldown (clickCooldown) prevents click spam.
//   - On success, clickPointsPerClick is added to the viewer's ledger.
//
// Concurrency: SELECT FOR UPDATE on the session row serialises concurrent clicks.
func (s *WatchService) RecordClick(userID uuid.UUID, channelID string) (*ClickResult, error) {
	var result ClickResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session models.WatchSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND channel_id = ? AND is_active = true", userID, channelID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoActiveSession
			}
			return err
		}

		if time.Now().Before(session.ClickCooldownUntil) {
			remaining := time.Until(session.ClickCooldownUntil)
			return ErrClickOnCooldown{RetryAfterMs: remaining.Milliseconds()}
		}

		if err := tx.Model(&session).Update("click_cooldown_until", time.Now().Add(clickCooldown)).Error; err != nil {
			return err
		}

		ledgerID := newUUID()
		upsertTime := time.Now()
		if err := tx.Exec(`
			INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (user_id, channel_id) DO UPDATE SET
				spendable_balance = points_ledgers.spendable_balance + EXCLUDED.spendable_balance,
				cumulative_total  = points_ledgers.cumulative_total  + EXCLUDED.cumulative_total,
				updated_at        = ?
		`, ledgerID, userID, channelID, clickPointsPerClick, clickPointsPerClick, upsertTime, upsertTime, upsertTime).Error; err != nil {
			return err
		}

		var ledger models.PointsLedger
		if err := tx.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&ledger).Error; err != nil {
			return err
		}

		txRecord := &models.PointsTransaction{
			LedgerID:       ledger.ID,
			WatchSessionID: &session.ID,
			Source:         models.TxSourceClick,
			Delta:          clickPointsPerClick,
			BalanceAfter:   ledger.SpendableBalance,
		}
		if err := tx.Create(txRecord).Error; err != nil {
			return err
		}

		result = ClickResult{BalanceAfter: ledger.SpendableBalance, Delta: clickPointsPerClick}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetBalance returns the viewer's current spendable balance and cumulative total
// for the given channel. Returns (0, 0, nil) if no ledger exists yet.
func (s *WatchService) GetBalance(userID uuid.UUID, channelID string) (spendable, cumulative int64, err error) {
	var ledger models.PointsLedger
	if err := s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&ledger).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return ledger.SpendableBalance, ledger.CumulativeTotal, nil
}

func (s *WatchService) getChannelConfig(db *gorm.DB, channelID string) (*models.ChannelConfig, error) {
	var cfg models.ChannelConfig
	if err := db.Where("channel_id = ?", channelID).First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.ChannelConfig{
				ChannelID:       channelID,
				SecondsPerPoint: DefaultSecondsPerPoint,
				Multiplier:      1,
			}, nil
		}
		return nil, err
	}
	if cfg.SecondsPerPoint <= 0 {
		cfg.SecondsPerPoint = DefaultSecondsPerPoint
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 1
	}
	return &cfg, nil
}
