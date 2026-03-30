package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/models"
)

// staleThreshold is how long without a heartbeat before a session is considered stale.
const staleThreshold = 2 * time.Minute

var ErrNoActiveSession = errors.New("no active session")

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
// index constraint on (opaque_user_id, channel_id) WHERE is_active = true.
func (s *WatchService) StartSession(opaqueUserID, channelID string) (*models.WatchSession, error) {
	var result models.WatchSession

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session models.WatchSession
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("opaque_user_id = ? AND channel_id = ? AND is_active = true", opaqueUserID, channelID).
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
			ID:              uuid.New(),
			OpaqueUserID:    opaqueUserID,
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
				Where("opaque_user_id = ? AND channel_id = ? AND is_active = true", opaqueUserID, channelID).
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
}

// Heartbeat advances the session timer and awards points if a full minute has accumulated.
//
// Concurrency protections:
//   - SELECT FOR UPDATE on the session row prevents two concurrent heartbeats from
//     reading the same state and double-awarding points.
//   - PointsLedger is updated via an atomic INSERT … ON CONFLICT DO UPDATE, which
//     prevents balance overwrites under concurrent writes.
func (s *WatchService) Heartbeat(opaqueUserID, channelID string) (*HeartbeatResult, error) {
	var result HeartbeatResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row so concurrent heartbeats queue up rather than racing.
		var session models.WatchSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("opaque_user_id = ? AND channel_id = ? AND is_active = true", opaqueUserID, channelID).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoActiveSession
			}
			return err
		}

		now := time.Now()
		delta := now.Sub(session.LastHeartbeatAt)
		if delta > 30*time.Second {
			delta = 30 * time.Second
		}
		// Use integer division on the duration to avoid float truncation issues
		// (e.g. 29.9s should count as 29s, not silently floor via float cast).
		deltaSeconds := int64(delta / time.Second)

		newAccumulated := session.AccumulatedSeconds + deltaSeconds
		pendingSeconds := newAccumulated - session.RewardedSeconds
		pointsToAward := pendingSeconds / 60
		newRewarded := session.RewardedSeconds + pointsToAward*60

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
			if err := tx.Exec(`
				INSERT INTO points_ledgers (id, opaque_user_id, spendable_balance, cumulative_total, created_at, updated_at)
				VALUES (gen_random_uuid(), ?, ?, ?, NOW(), NOW())
				ON CONFLICT (opaque_user_id) DO UPDATE SET
					spendable_balance = points_ledgers.spendable_balance + EXCLUDED.spendable_balance,
					cumulative_total  = points_ledgers.cumulative_total  + EXCLUDED.cumulative_total,
					updated_at        = NOW()
			`, opaqueUserID, pointsToAward, pointsToAward).Error; err != nil {
				return err
			}

			// Fetch the balance that was just written (within the same tx for consistency).
			var ledger models.PointsLedger
			if err := tx.Where("opaque_user_id = ?", opaqueUserID).First(&ledger).Error; err != nil {
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

		result = HeartbeatResult{Session: &session, PointsEarned: pointsToAward}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// EndSession marks the viewer's active session in the given channel as ended.
// If no active session exists the call is a no-op (idempotent).
func (s *WatchService) EndSession(opaqueUserID, channelID string) error {
	now := time.Now()
	return s.db.Model(&models.WatchSession{}).
		Where("opaque_user_id = ? AND channel_id = ? AND is_active = true", opaqueUserID, channelID).
		Updates(map[string]interface{}{
			"is_active": false,
			"ended_at":  now,
		}).Error
}

// GetBalance returns the viewer's current spendable balance and cumulative total.
// Returns (0, 0, nil) if no ledger exists yet.
func (s *WatchService) GetBalance(opaqueUserID string) (spendable, cumulative int64, err error) {
	var ledger models.PointsLedger
	if err := s.db.Where("opaque_user_id = ?", opaqueUserID).First(&ledger).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return ledger.SpendableBalance, ledger.CumulativeTotal, nil
}
