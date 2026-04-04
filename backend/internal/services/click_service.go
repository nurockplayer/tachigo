package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/models"
)

const (
	// clickCooldown is the minimum time between two clicks for the same viewer in the same channel.
	clickCooldown = 10 * time.Second
	// pointsPerClick is the fixed reward for each valid click.
	pointsPerClick int64 = 1
)

// ErrClickCooldown is returned when the viewer clicks too fast.
var ErrClickCooldown = errors.New("click cooldown active")

// ClickResult holds the outcome of a click.
type ClickResult struct {
	PointsEarned        int64
	NewBalance          int64
	CooldownRemainingMs int64 // 0 when the click was accepted
}

type ClickService struct {
	db *gorm.DB
}

func NewClickService(db *gorm.DB) *ClickService {
	return &ClickService{db: db}
}

// Click awards pointsPerClick to the viewer if the cooldown has expired.
//
// Concurrency safety:
//   - The ledger row is locked with SELECT FOR UPDATE before reading the last
//     click timestamp so that two concurrent requests cannot both pass the
//     cooldown check and double-award points.
//   - The ledger upsert uses ON CONFLICT DO UPDATE (atomic add) to avoid
//     read-modify-write races on the balance columns.
func (s *ClickService) Click(userID uuid.UUID, channelID string) (*ClickResult, error) {
	var result ClickResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Upsert ledger row so we always have a row to lock.
		ledgerID := newUUID()
		now := time.Now()
		if err := tx.Exec(`
			INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
			VALUES (?, ?, ?, 0, 0, ?, ?)
			ON CONFLICT (user_id, channel_id) DO NOTHING
		`, ledgerID, userID, channelID, now, now).Error; err != nil {
			return err
		}

		// Lock the ledger row.
		var ledger models.PointsLedger
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND channel_id = ?", userID, channelID).
			First(&ledger).Error; err != nil {
			return err
		}

		// Find the last click timestamp (if any) without a separate table.
		var lastClick models.PointsTransaction
		err := tx.
			Where("ledger_id = ? AND source = ?", ledger.ID, models.TxSourceClick).
			Order("created_at DESC").
			First(&lastClick).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err == nil {
			remaining := clickCooldown - time.Since(lastClick.CreatedAt)
			if remaining > 0 {
				result = ClickResult{
					PointsEarned:        0,
					NewBalance:          ledger.SpendableBalance,
					CooldownRemainingMs: remaining.Milliseconds(),
				}
				return ErrClickCooldown
			}
		}

		// Award points via atomic upsert.
		upsertTime := time.Now()
		if err := tx.Exec(`
			UPDATE points_ledgers SET
				spendable_balance = spendable_balance + ?,
				cumulative_total  = cumulative_total  + ?,
				updated_at        = ?
			WHERE user_id = ? AND channel_id = ?
		`, pointsPerClick, pointsPerClick, upsertTime, userID, channelID).Error; err != nil {
			return err
		}

		// Re-read balance after update.
		if err := tx.Where("user_id = ? AND channel_id = ?", userID, channelID).
			First(&ledger).Error; err != nil {
			return err
		}

		txRecord := &models.PointsTransaction{
			LedgerID:     ledger.ID,
			Source:       models.TxSourceClick,
			Delta:        pointsPerClick,
			BalanceAfter: ledger.SpendableBalance,
		}
		if err := tx.Create(txRecord).Error; err != nil {
			return err
		}

		result = ClickResult{
			PointsEarned:        pointsPerClick,
			NewBalance:          ledger.SpendableBalance,
			CooldownRemainingMs: 0,
		}
		return nil
	})

	if errors.Is(err, ErrClickCooldown) {
		// Not a hard error — caller inspects CooldownRemainingMs.
		return &result, ErrClickCooldown
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}
