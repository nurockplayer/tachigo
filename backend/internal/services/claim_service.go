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
	ErrClaimAmountInvalid    = errors.New("claim amount must be greater than zero")
	ErrClaimInsufficientBalance = errors.New("insufficient spendable balance to claim")
)

type ClaimService struct {
	db *gorm.DB
}

func NewClaimService(db *gorm.DB) *ClaimService {
	return &ClaimService{db: db}
}

// GetTachiBalance returns the user's current $TACHI balance.
// Returns 0 if no balance record exists yet.
func (s *ClaimService) GetTachiBalance(userID uuid.UUID) (int64, error) {
	var tb models.TachiBalance
	err := s.db.Where("user_id = ?", userID).First(&tb).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return tb.Balance, nil
}

// Claim converts T-Points from all channels into $TACHI balance.
// amount == 0 means claim all available spendable_balance.
// Returns the new tachi_balances.balance after the claim.
func (s *ClaimService) Claim(userID uuid.UUID, amount int64) (int64, error) {
	var newBalance int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock all ledgers for this user.
		// ORDER BY created_at ASC, id ASC ensures deterministic deduction order
		// across DB restarts and query plan changes.
		var ledgers []models.PointsLedger
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND spendable_balance > 0", userID).
			Order("created_at ASC, id ASC").
			Find(&ledgers).Error; err != nil {
			return err
		}

		// Sum total spendable balance across all channels.
		var totalSpendable int64
		for _, l := range ledgers {
			totalSpendable += l.SpendableBalance
		}

		// Resolve claim amount.
		claimAmount := amount
		if claimAmount == 0 {
			claimAmount = totalSpendable
		}
		if claimAmount <= 0 {
			return ErrClaimAmountInvalid
		}
		if totalSpendable < claimAmount {
			return ErrClaimInsufficientBalance
		}

		// Deduct from ledgers greedily in created_at ASC, id ASC order (oldest first).
		remaining := claimAmount
		now := time.Now()
		for _, ledger := range ledgers {
			if remaining == 0 {
				break
			}
			deduct := ledger.SpendableBalance
			if deduct > remaining {
				deduct = remaining
			}
			newBalance := ledger.SpendableBalance - deduct
			if err := tx.Model(&ledger).Updates(map[string]interface{}{
				"spendable_balance": newBalance,
				"updated_at":        now,
			}).Error; err != nil {
				return err
			}
			txRecord := &models.PointsTransaction{
				LedgerID:     ledger.ID,
				Source:       models.TxSourceClaim,
				Delta:        -deduct,
				BalanceAfter: newBalance,
			}
			if err := tx.Create(txRecord).Error; err != nil {
				return err
			}
			remaining -= deduct
		}

		// Upsert tachi_balances.
		if err := tx.Exec(`
			INSERT INTO tachi_balances (id, user_id, balance, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (user_id) DO UPDATE SET
				balance    = tachi_balances.balance + EXCLUDED.balance,
				updated_at = EXCLUDED.updated_at
		`, newUUID(), userID, claimAmount, now).Error; err != nil {
			return err
		}

		// Read back the new balance.
		var tb models.TachiBalance
		if err := tx.Where("user_id = ?", userID).First(&tb).Error; err != nil {
			return err
		}
		newBalance = tb.Balance
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newBalance, nil
}
