package services

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/config"
	contractpkg "github.com/tachigo/tachigo/internal/contract"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrClaimAmountInvalid       = errors.New("claim amount must be greater than zero")
	ErrClaimInsufficientBalance = errors.New("insufficient spendable balance to claim")
	ErrClaimWalletNotLinked     = errors.New("web3 wallet not linked")
	ErrClaimContractConfig      = errors.New("claim contract config is incomplete")
)

// ClaimBroadcastRecordError preserves the tx hash when the chain tx was
// broadcast but the DB failed before recording the broadcast state.
type ClaimBroadcastRecordError struct {
	ClaimID uuid.UUID
	UserID  uuid.UUID
	TxHash  string
	Err     error
}

func (e *ClaimBroadcastRecordError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"record claim broadcast failed: claim_id=%s user_id=%s tx_hash=%s: %v",
		e.ClaimID,
		e.UserID,
		e.TxHash,
		e.Err,
	)
}

func (e *ClaimBroadcastRecordError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type MintCaller interface {
	MintBroadcastOnChain(ctx context.Context, toAddr string, amount int64) (txHash string, err error)
	WaitMintReceiptOnChain(ctx context.Context, txHash string) error
}

type mintContract interface {
	MintBroadcast(ctx context.Context, toAddr common.Address, amount *big.Int, signerKey *ecdsa.PrivateKey) (txHash string, err error)
	WaitMintReceipt(ctx context.Context, txHash string) error
}

type ClaimService struct {
	db          *gorm.DB
	contractCfg config.ContractConfig
	tachiToken  mintContract
	mintCaller  MintCaller
}

type claimReservation struct {
	userID  uuid.UUID
	toAddr  string
	amount  int64
	claimID uuid.UUID
	items   []claimReservationItem
}

type claimReservationItem struct {
	ledgerID      uuid.UUID
	transactionID uuid.UUID
	amount        int64
}

func NewClaimService(db *gorm.DB, contractCfg config.ContractConfig, ethClient *ethclient.Client) *ClaimService {
	svc := &ClaimService{
		db:          db,
		contractCfg: contractCfg,
	}
	if ethClient != nil && contractCfg.TachiContractAddress != "" && contractCfg.SepoliaSignerKey != "" {
		if common.IsHexAddress(contractCfg.TachiContractAddress) {
			t, err := contractpkg.NewTachiToken(common.HexToAddress(contractCfg.TachiContractAddress), ethClient)
			if err == nil {
				svc.tachiToken = t
			}
		}
	}
	svc.mintCaller = svc
	return svc
}

// SetMintCallerForTest replaces the mint caller; use only in tests.
func (s *ClaimService) SetMintCallerForTest(mc MintCaller) { s.mintCaller = mc }

// GetTachiBalance returns the user's current $TACHI balance.
// Returns 0 if no balance record exists yet.
func (s *ClaimService) GetTachiBalance(userID uuid.UUID) (int64, error) {
	balance, err := loadTachiBalanceValue(s.db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return balance, nil
}

// Claim converts T-Points from all channels into $TACHI balance.
// amount == 0 means claim all available spendable_balance.
// Returns the new tachi_balances.balance after the claim.
func (s *ClaimService) Claim(ctx context.Context, userID uuid.UUID, amount int64) (int64, error) {
	mintCaller := s.mintCaller
	if mintCaller == nil {
		mintCaller = s
	}

	var reservation claimReservation
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		reservation, err = s.reserveClaim(tx, userID, amount)
		return err
	}); err != nil {
		return 0, err
	}

	mintCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	mintTxHash, err := mintCaller.MintBroadcastOnChain(mintCtx, reservation.toAddr, reservation.amount)
	if err != nil {
		rollbackErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.rollbackClaimReservation(tx, reservation)
		})
		if rollbackErr != nil {
			return 0, fmt.Errorf("%w; rollback claim reservation: %v", err, rollbackErr)
		}
		return 0, err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.markClaimBroadcast(tx, reservation, mintTxHash)
	}); err != nil {
		recordErr := &ClaimBroadcastRecordError{
			ClaimID: reservation.claimID,
			UserID:  reservation.userID,
			TxHash:  mintTxHash,
			Err:     err,
		}
		log.Printf(
			"claim broadcast record failed: claim_id=%s user_id=%s tx_hash=%s err=%v",
			reservation.claimID,
			reservation.userID,
			mintTxHash,
			err,
		)
		return 0, recordErr
	}

	if err := mintCaller.WaitMintReceiptOnChain(mintCtx, mintTxHash); err != nil {
		logMsg := "claim mint receipt unknown"
		if errors.Is(err, contractpkg.ErrMintReceiptStatusFailed) {
			logMsg = "claim mint receipt failed"
		}
		log.Printf(
			"%s: claim_id=%s user_id=%s tx_hash=%s err=%v",
			logMsg,
			reservation.claimID,
			reservation.userID,
			mintTxHash,
			err,
		)

		if errors.Is(err, contractpkg.ErrMintReceiptStatusFailed) {
			failedErr := s.db.Transaction(func(tx *gorm.DB) error {
				return s.markClaimFailedAndCompensate(tx, reservation, mintTxHash, err)
			})
			if failedErr != nil {
				return 0, fmt.Errorf("%w; mark claim failed: %v", err, failedErr)
			}
			return 0, err
		}

		recordErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.recordClaimReceiptUnknown(tx, reservation, mintTxHash, err)
		})
		if recordErr != nil {
			return 0, fmt.Errorf("%w; record claim receipt unknown: %v", err, recordErr)
		}
		return 0, err
	}

	var newBalance int64
	finalizeErr := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		newBalance, err = s.finalizeClaim(tx, reservation, mintTxHash)
		return err
	})
	if finalizeErr != nil {
		markErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.markFinalizeFailedClaim(tx, reservation, mintTxHash, finalizeErr)
		})
		if markErr != nil {
			return 0, fmt.Errorf("%w; mark finalize failed: %v", finalizeErr, markErr)
		}
		return 0, finalizeErr
	}

	return newBalance, nil
}

func (s *ClaimService) MintOnChain(ctx context.Context, toAddr string, amount int64) (string, error) {
	txHash, err := s.MintBroadcastOnChain(ctx, toAddr, amount)
	if err != nil {
		return "", err
	}
	if err := s.WaitMintReceiptOnChain(ctx, txHash); err != nil {
		return txHash, err
	}
	return txHash, nil
}

// MintBroadcastOnChain only broadcasts mint tx; receipt waiting is separate.
func (s *ClaimService) MintBroadcastOnChain(ctx context.Context, toAddr string, amount int64) (string, error) {
	if s.tachiToken == nil {
		return "", ErrClaimContractConfig
	}
	if !common.IsHexAddress(toAddr) {
		return "", fmt.Errorf("invalid wallet address: %s", toAddr)
	}
	if amount <= 0 {
		return "", ErrClaimAmountInvalid
	}

	signerKey, err := parseSignerKey(s.contractCfg.SepoliaSignerKey)
	if err != nil {
		return "", err
	}

	return s.tachiToken.MintBroadcast(ctx, common.HexToAddress(toAddr), tachiWholeTokensToRawUnits(amount), signerKey)
}

// WaitMintReceiptOnChain waits receipt for a previously broadcast mint tx.
func (s *ClaimService) WaitMintReceiptOnChain(ctx context.Context, txHash string) error {
	if s.tachiToken == nil {
		return ErrClaimContractConfig
	}
	return s.tachiToken.WaitMintReceipt(ctx, txHash)
}

func (s *ClaimService) reserveClaim(tx *gorm.DB, userID uuid.UUID, amount int64) (claimReservation, error) {
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND spendable_balance > 0", userID).
		Order("created_at ASC, id ASC")

	var ledgers []models.PointsLedger
	if err := query.Find(&ledgers).Error; err != nil {
		return claimReservation{}, err
	}

	var totalSpendable int64
	for _, l := range ledgers {
		totalSpendable += l.SpendableBalance
	}

	claimAmount := amount
	if claimAmount == 0 {
		claimAmount = totalSpendable
	}
	if claimAmount <= 0 {
		return claimReservation{}, ErrClaimAmountInvalid
	}
	if totalSpendable < claimAmount {
		return claimReservation{}, ErrClaimInsufficientBalance
	}

	toAddr, err := s.resolveWalletAddress(tx, userID)
	if err != nil {
		return claimReservation{}, err
	}

	reservation := claimReservation{
		userID: userID,
		toAddr: toAddr,
		amount: claimAmount,
		items:  make([]claimReservationItem, 0, len(ledgers)),
	}
	claim := &models.Claim{
		UserID:     userID,
		WalletAddr: toAddr,
		Amount:     claimAmount,
		Status:     models.ClaimStatusPending,
	}
	if err := tx.Create(claim).Error; err != nil {
		return claimReservation{}, err
	}
	reservation.claimID = claim.ID
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
		newLedgerBalance := ledger.SpendableBalance - deduct
		if err := tx.Model(&ledger).Updates(map[string]interface{}{
			"spendable_balance": newLedgerBalance,
			"updated_at":        now,
		}).Error; err != nil {
			return claimReservation{}, err
		}

		txRecord := &models.PointsTransaction{
			LedgerID:     ledger.ID,
			Source:       models.TxSourceClaim,
			Delta:        -deduct,
			BalanceAfter: newLedgerBalance,
		}
		if err := tx.Create(txRecord).Error; err != nil {
			return claimReservation{}, err
		}
		claimItem := &models.ClaimItem{
			ClaimID:             claim.ID,
			ClaimUserID:         userID,
			LedgerID:            ledger.ID,
			PointsTransactionID: txRecord.ID,
			Amount:              deduct,
		}
		if err := tx.Create(claimItem).Error; err != nil {
			return claimReservation{}, err
		}
		reservation.items = append(reservation.items, claimReservationItem{
			ledgerID:      ledger.ID,
			transactionID: txRecord.ID,
			amount:        deduct,
		})
		remaining -= deduct
	}

	return reservation, nil
}

func (s *ClaimService) markClaimBroadcast(tx *gorm.DB, reservation claimReservation, mintTxHash string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":        models.ClaimStatusBroadcast,
		"tx_hash":       mintTxHash,
		"broadcast_at":  now,
		"error_message": nil,
		"updated_at":    now,
	}
	return tx.Model(&models.Claim{}).
		Where("id = ? AND user_id = ?", reservation.claimID, reservation.userID).
		Updates(updates).Error
}

func (s *ClaimService) rollbackClaimReservation(tx *gorm.DB, reservation claimReservation) error {
	if reservation.claimID != uuid.Nil {
		if err := tx.Delete(&models.Claim{}, "id = ?", reservation.claimID).Error; err != nil {
			return err
		}
	}

	now := time.Now()
	for _, item := range reservation.items {
		if err := tx.Model(&models.PointsLedger{}).
			Where("id = ?", item.ledgerID).
			Updates(map[string]interface{}{
				"spendable_balance": gorm.Expr("spendable_balance + ?", item.amount),
				"updated_at":        now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.PointsTransaction{}, "id = ?", item.transactionID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ClaimService) recordClaimReceiptUnknown(tx *gorm.DB, reservation claimReservation, mintTxHash string, waitErr error) error {
	return tx.Model(&models.Claim{}).
		Where("id = ? AND user_id = ?", reservation.claimID, reservation.userID).
		Updates(map[string]interface{}{
			"status":        models.ClaimStatusBroadcast,
			"tx_hash":       mintTxHash,
			"error_message": waitErr.Error(),
			"updated_at":    time.Now(),
		}).Error
}

func (s *ClaimService) markClaimFailedAndCompensate(tx *gorm.DB, reservation claimReservation, mintTxHash string, mintErr error) error {
	now := time.Now()
	if err := tx.Model(&models.Claim{}).
		Where("id = ? AND user_id = ?", reservation.claimID, reservation.userID).
		Updates(map[string]interface{}{
			"status":        models.ClaimStatusFailed,
			"tx_hash":       mintTxHash,
			"error_message": mintErr.Error(),
			"failed_at":     now,
			"updated_at":    now,
		}).Error; err != nil {
		return err
	}

	for _, item := range reservation.items {
		var ledger models.PointsLedger
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", item.ledgerID).
			First(&ledger).Error; err != nil {
			return err
		}

		balanceAfter := ledger.SpendableBalance + item.amount
		if err := tx.Model(&ledger).Updates(map[string]interface{}{
			"spendable_balance": balanceAfter,
			"updated_at":        now,
		}).Error; err != nil {
			return err
		}

		note := fmt.Sprintf("claim failed compensation for claim %s", reservation.claimID)
		txRecord := &models.PointsTransaction{
			LedgerID:     item.ledgerID,
			Source:       models.TxSourceClaim,
			Delta:        item.amount,
			BalanceAfter: balanceAfter,
			Note:         &note,
		}
		if err := tx.Create(txRecord).Error; err != nil {
			return err
		}
	}

	return nil
}

// markFinalizeFailedClaim records that the chain tx succeeded but the DB
// finalization step failed. The claim is left in finalize_failed state for a
// recovery job to retry. Points are not compensated here because the on-chain
// mint already confirmed.
func (s *ClaimService) markFinalizeFailedClaim(tx *gorm.DB, reservation claimReservation, mintTxHash string, finalizeErr error) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":             models.ClaimStatusFinalizeFailed,
		"error_message":      finalizeErr.Error(),
		"finalize_failed_at": now,
		"updated_at":         now,
	}
	if mintTxHash != "" {
		updates["tx_hash"] = mintTxHash
	}

	query := tx.Model(&models.Claim{}).
		Where(
			"id = ? AND user_id = ? AND status IN ?",
			reservation.claimID,
			reservation.userID,
			[]string{
				string(models.ClaimStatusBroadcast),
				string(models.ClaimStatusFinalizeFailed),
			},
		)
	if mintTxHash != "" {
		query = query.Where("(tx_hash IS NULL OR tx_hash = ?)", mintTxHash)
	}

	result := query.Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var existing models.Claim
		if err := tx.Select("status", "tx_hash").
			Where("id = ? AND user_id = ?", reservation.claimID, reservation.userID).
			First(&existing).Error; err != nil {
			return fmt.Errorf("markFinalizeFailedClaim: load claim after skipped mark: %w", err)
		}
		if mintTxHash != "" && existing.TxHash != nil && *existing.TxHash != mintTxHash {
			return fmt.Errorf("markFinalizeFailedClaim: tx_hash mismatch existing=%s retry=%s", *existing.TxHash, mintTxHash)
		}
		if existing.Status == models.ClaimStatusConfirmed {
			return nil
		}
		return fmt.Errorf("markFinalizeFailedClaim: invalid claim status %s", existing.Status)
	}
	return nil
}

func (s *ClaimService) finalizeClaim(tx *gorm.DB, reservation claimReservation, mintTxHash string) (int64, error) {
	now := time.Now()
	claimUpdates := map[string]interface{}{
		"status":        models.ClaimStatusConfirmed,
		"confirmed_at":  now,
		"error_message": nil,
		"updated_at":    now,
	}
	if mintTxHash != "" {
		claimUpdates["tx_hash"] = mintTxHash
	}

	query := tx.Model(&models.Claim{}).
		Where(
			"id = ? AND user_id = ? AND status IN ?",
			reservation.claimID,
			reservation.userID,
			[]string{
				string(models.ClaimStatusBroadcast),
				string(models.ClaimStatusFinalizeFailed),
			},
		)
	if mintTxHash != "" {
		query = query.Where("(tx_hash IS NULL OR tx_hash = ?)", mintTxHash)
	}

	result := query.Updates(claimUpdates)
	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected == 0 {
		var existing models.Claim
		if err := tx.Select("status", "tx_hash").
			Where("id = ? AND user_id = ?", reservation.claimID, reservation.userID).
			First(&existing).Error; err != nil {
			return 0, fmt.Errorf("finalizeClaim: load claim after skipped finalize: %w", err)
		}

		if mintTxHash != "" && existing.TxHash != nil && *existing.TxHash != mintTxHash {
			return 0, fmt.Errorf("finalizeClaim: tx_hash mismatch existing=%s retry=%s", *existing.TxHash, mintTxHash)
		}

		if existing.Status == models.ClaimStatusConfirmed {
			balance, err := loadTachiBalanceValue(tx, reservation.userID)
			if err != nil {
				return 0, fmt.Errorf("finalizeClaim: confirmed claim missing tachi balance: %w", err)
			}
			return balance, nil
		}

		return 0, fmt.Errorf("finalizeClaim: invalid claim status %s", existing.Status)
	}

	if err := tx.Exec(`
		INSERT INTO tachi_balances (id, user_id, balance, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			balance    = tachi_balances.balance + EXCLUDED.balance,
			updated_at = EXCLUDED.updated_at
	`, newUUID(), reservation.userID, reservation.amount, now).Error; err != nil {
		return 0, err
	}

	balance, err := loadTachiBalanceValue(tx, reservation.userID)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func loadTachiBalanceValue(db *gorm.DB, userID uuid.UUID) (int64, error) {
	var balance int64
	err := db.Raw(
		"SELECT CAST(balance AS BIGINT) FROM tachi_balances WHERE user_id = ?",
		userID,
	).Row().Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, gorm.ErrRecordNotFound
		}
		return 0, err
	}
	return balance, nil
}

func (s *ClaimService) resolveWalletAddress(db *gorm.DB, userID uuid.UUID) (string, error) {
	var authProvider models.AuthProvider
	err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).
		Order("created_at ASC").
		First(&authProvider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrClaimWalletNotLinked
		}
		return "", err
	}
	if !common.IsHexAddress(authProvider.ProviderID) {
		return "", fmt.Errorf("invalid linked wallet address: %s", authProvider.ProviderID)
	}
	return common.HexToAddress(authProvider.ProviderID).Hex(), nil
}

func parseSignerKey(rawKey string) (*ecdsa.PrivateKey, error) {
	key := strings.TrimPrefix(rawKey, "0x")
	signerKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return nil, fmt.Errorf("parse signer key: %w", err)
	}
	return signerKey, nil
}
