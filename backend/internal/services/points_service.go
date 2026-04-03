package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrInsufficientBalance = errors.New("insufficient spendable balance")
	ErrLedgerNotFound      = errors.New("points ledger not found")
)

// PointsBalance holds both balance views for a viewer in a channel.
type PointsBalance struct {
	CumulativeTotal  int64 `json:"cumulative_total"`
	SpendableBalance int64 `json:"spendable_balance"`
}

// WatchStats holds a viewer's accumulated watch time in a channel.
type WatchStats struct {
	TotalWatchSeconds int64 `json:"total_watch_seconds"`
}

// BroadcastStats holds a streamer's broadcast time across four time windows.
type BroadcastStats struct {
	CurrentSessionSeconds int64 `json:"current_session_seconds"`
	DailySeconds          int64 `json:"daily_seconds"`
	MonthlySeconds        int64 `json:"monthly_seconds"`
	YearlySeconds         int64 `json:"yearly_seconds"`
}

type PointsService struct {
	db       *gorm.DB
	watchSvc *WatchService
}

func NewPointsService(db *gorm.DB, watchSvc *WatchService) *PointsService {
	return &PointsService{db: db, watchSvc: watchSvc}
}

// GetBalance wraps WatchService.GetBalance and returns a PointsBalance struct.
// Returns zeroed balance if no ledger exists yet.
func (s *PointsService) GetBalance(userID uuid.UUID, channelID string) (*PointsBalance, error) {
	spendable, cumulative, err := s.watchSvc.GetBalance(userID, channelID)
	if err != nil {
		return nil, err
	}
	return &PointsBalance{
		CumulativeTotal:  cumulative,
		SpendableBalance: spendable,
	}, nil
}

// ListTransactions returns the most recent 50 transactions for a viewer in a channel.
func (s *PointsService) ListTransactions(userID uuid.UUID, channelID string) ([]models.PointsTransaction, error) {
	var ledger models.PointsLedger
	if err := s.db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&ledger).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.PointsTransaction{}, nil
		}
		return nil, err
	}

	var txs []models.PointsTransaction
	err := s.db.Where("ledger_id = ?", ledger.ID).
		Order("created_at DESC").
		Limit(50).
		Find(&txs).Error
	return txs, err
}

// DeductPoints subtracts amount from spendable_balance only.
// cumulative_total is never modified by a deduction.
// Returns ErrInsufficientBalance if the current spendable balance is too low.
func (s *PointsService) DeductPoints(userID uuid.UUID, channelID string, amount int64, note string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var ledger models.PointsLedger
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND channel_id = ?", userID, channelID).
			First(&ledger).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInsufficientBalance
			}
			return err
		}

		if ledger.SpendableBalance < amount {
			return ErrInsufficientBalance
		}

		newBalance := ledger.SpendableBalance - amount
		if err := tx.Model(&ledger).Updates(map[string]interface{}{
			"spendable_balance": newBalance,
			"updated_at":        time.Now(),
		}).Error; err != nil {
			return err
		}

		notePtr := &note
		txRecord := &models.PointsTransaction{
			LedgerID:     ledger.ID,
			Source:       models.TxSourceSpend,
			Delta:        -amount,
			BalanceAfter: newBalance,
			Note:         notePtr,
		}
		return tx.Create(txRecord).Error
	})
}

// AddPoints adds amount to both spendable_balance and cumulative_total.
// Intended for use by AirdropService — Heartbeat points are handled by WatchService.Heartbeat.
func (s *PointsService) AddPoints(userID uuid.UUID, channelID string, source models.TxSource, amount int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		ledgerID := newUUID()
		now := time.Now()

		if err := tx.Exec(`
			INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (user_id, channel_id) DO UPDATE SET
				spendable_balance = points_ledgers.spendable_balance + EXCLUDED.spendable_balance,
				cumulative_total  = points_ledgers.cumulative_total  + EXCLUDED.cumulative_total,
				updated_at        = ?
		`, ledgerID, userID, channelID, amount, amount, now, now, now).Error; err != nil {
			return err
		}

		var ledger models.PointsLedger
		if err := tx.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&ledger).Error; err != nil {
			return err
		}

		txRecord := &models.PointsTransaction{
			LedgerID:     ledger.ID,
			Source:       source,
			Delta:        amount,
			BalanceAfter: ledger.SpendableBalance,
		}
		return tx.Create(txRecord).Error
	})
}

// AddWatchTime accumulates observed seconds for a viewer in a channel.
// Called after each successful Heartbeat.
func (s *PointsService) AddWatchTime(userID uuid.UUID, channelID string, seconds int64) error {
	now := time.Now()
	return s.db.Exec(`
		INSERT INTO watch_time_stats (id, user_id, channel_id, total_watch_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, channel_id) DO UPDATE SET
			total_watch_seconds = watch_time_stats.total_watch_seconds + EXCLUDED.total_watch_seconds,
			updated_at          = ?
	`, newUUID(), userID, channelID, seconds, now, now, now).Error
}

// GetWatchStats returns the total accumulated watch seconds for a viewer in a channel.
func (s *PointsService) GetWatchStats(userID uuid.UUID, channelID string) (*WatchStats, error) {
	var result struct {
		TotalWatchSeconds int64
	}
	err := s.db.Raw(`
		SELECT COALESCE(total_watch_seconds, 0) AS total_watch_seconds
		FROM watch_time_stats
		WHERE user_id = ? AND channel_id = ?
	`, userID, channelID).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &WatchStats{TotalWatchSeconds: result.TotalWatchSeconds}, nil
}

// AddBroadcastTime accumulates broadcast seconds for a streamer in a channel.
// Called on each Heartbeat received, in sync with viewer AddWatchTime.
func (s *PointsService) AddBroadcastTime(streamerID uuid.UUID, channelID string, seconds int64) error {
	now := time.Now()
	return s.db.Exec(`
		INSERT INTO broadcast_time_stats (id, streamer_id, channel_id, total_broadcast_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (streamer_id, channel_id) DO UPDATE SET
			total_broadcast_seconds = broadcast_time_stats.total_broadcast_seconds + EXCLUDED.total_broadcast_seconds,
			updated_at              = ?
	`, newUUID(), streamerID, channelID, seconds, now, now, now).Error
}

// GetBroadcastStats returns broadcast time across four time windows for a streamer.
func (s *PointsService) GetBroadcastStats(streamerID uuid.UUID, channelID string) (*BroadcastStats, error) {
	now := time.Now()

	// current_session_seconds: accumulated seconds in the current active watch session
	// used as a proxy for the ongoing broadcast session duration.
	var currentSession struct {
		AccumulatedSeconds int64
	}
	s.db.Raw(`
		SELECT COALESCE(SUM(accumulated_seconds), 0) AS accumulated_seconds
		FROM watch_sessions
		WHERE channel_id = ? AND is_active = true
	`, channelID).Scan(&currentSession)

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	query := `
		SELECT COALESCE(SUM(total_broadcast_seconds), 0) AS total
		FROM broadcast_time_stats
		WHERE streamer_id = ? AND channel_id = ? AND created_at >= ?
	`

	fetch := func(since time.Time) int64 {
		var r struct{ Total int64 }
		s.db.Raw(query, streamerID, channelID, since).Scan(&r)
		return r.Total
	}

	return &BroadcastStats{
		CurrentSessionSeconds: currentSession.AccumulatedSeconds,
		DailySeconds:          fetch(startOfDay),
		MonthlySeconds:        fetch(startOfMonth),
		YearlySeconds:         fetch(startOfYear),
	}, nil
}
