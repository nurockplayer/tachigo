package services

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
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

type MintCaller interface {
	MintOnChain(ctx context.Context, toAddr string, amount int64) (txHash string, err error)
}

type ClaimService struct {
	db          *gorm.DB
	contractCfg config.ContractConfig
	tachiToken  *contractpkg.TachiToken
	mintCaller  MintCaller
}

type claimReservation struct {
	userID uuid.UUID
	toAddr string
	amount int64
	items  []claimReservationItem
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
	if _, err := mintCaller.MintOnChain(mintCtx, reservation.toAddr, reservation.amount); err != nil {
		rollbackErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.rollbackClaimReservation(tx, reservation)
		})
		if rollbackErr != nil {
			return 0, fmt.Errorf("%w; rollback claim reservation: %v", err, rollbackErr)
		}
		return 0, err
	}

	var newBalance int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		newBalance, err = s.finalizeClaim(tx, reservation.userID, reservation.amount)
		return err
	})
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (s *ClaimService) MintOnChain(ctx context.Context, toAddr string, amount int64) (string, error) {
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

	// TODO: wait for receipt status == 1 before committing DB (fire-and-forget for now)
	return s.tachiToken.Mint(ctx, common.HexToAddress(toAddr), big.NewInt(amount), signerKey)
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
		reservation.items = append(reservation.items, claimReservationItem{
			ledgerID:      ledger.ID,
			transactionID: txRecord.ID,
			amount:        deduct,
		})
		remaining -= deduct
	}

	return reservation, nil
}

func (s *ClaimService) rollbackClaimReservation(tx *gorm.DB, reservation claimReservation) error {
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

func (s *ClaimService) finalizeClaim(tx *gorm.DB, userID uuid.UUID, claimAmount int64) (int64, error) {
	now := time.Now()
	if err := tx.Exec(`
		INSERT INTO tachi_balances (id, user_id, balance, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			balance    = tachi_balances.balance + EXCLUDED.balance,
			updated_at = EXCLUDED.updated_at
	`, newUUID(), userID, claimAmount, now).Error; err != nil {
		return 0, err
	}

	var tb models.TachiBalance
	if err := tx.Where("user_id = ?", userID).First(&tb).Error; err != nil {
		return 0, err
	}

	return tb.Balance, nil
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
