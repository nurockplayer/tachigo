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
	ethClient   *ethclient.Client
	mintCaller  MintCaller
}

func NewClaimService(db *gorm.DB, contractCfg config.ContractConfig, ethClient *ethclient.Client) *ClaimService {
	svc := &ClaimService{
		db:          db,
		contractCfg: contractCfg,
		ethClient:   ethClient,
	}
	svc.mintCaller = svc
	return svc
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
	mintCaller := s.mintCaller
	if mintCaller == nil {
		mintCaller = s
	}

	var newBalance int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		claimAmount, err := s.calculateClaimAmount(tx, userID, amount, true)
		if err != nil {
			return err
		}

		toAddr, err := s.resolveWalletAddress(tx, userID)
		if err != nil {
			return err
		}

		if _, err := mintCaller.MintOnChain(context.Background(), toAddr, claimAmount); err != nil {
			return err
		}

		var applyErr error
		newBalance, applyErr = s.applyClaim(tx, userID, claimAmount)
		return applyErr
	})
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (s *ClaimService) MintOnChain(ctx context.Context, toAddr string, amount int64) (string, error) {
	if s.contractCfg.TachiContractAddress == "" || s.contractCfg.SepoliaSignerKey == "" {
		return "", ErrClaimContractConfig
	}
	if s.ethClient == nil {
		return "", ErrClaimContractConfig
	}
	if !common.IsHexAddress(s.contractCfg.TachiContractAddress) {
		return "", fmt.Errorf("invalid contract address: %s", s.contractCfg.TachiContractAddress)
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

	token, err := contractpkg.NewTachiToken(common.HexToAddress(s.contractCfg.TachiContractAddress), s.ethClient)
	if err != nil {
		return "", err
	}

	return token.Mint(ctx, common.HexToAddress(toAddr), big.NewInt(amount), signerKey)
}

func (s *ClaimService) calculateClaimAmount(db *gorm.DB, userID uuid.UUID, amount int64, lock bool) (int64, error) {
	query := db.Where("user_id = ? AND spendable_balance > 0", userID).Order("created_at ASC, id ASC")
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var ledgers []models.PointsLedger
	if err := query.Find(&ledgers).Error; err != nil {
		return 0, err
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
		return 0, ErrClaimAmountInvalid
	}
	if totalSpendable < claimAmount {
		return 0, ErrClaimInsufficientBalance
	}

	return claimAmount, nil
}

func (s *ClaimService) applyClaim(tx *gorm.DB, userID uuid.UUID, claimAmount int64) (int64, error) {
	var ledgers []models.PointsLedger
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND spendable_balance > 0", userID).
		Order("created_at ASC, id ASC").
		Find(&ledgers).Error; err != nil {
		return 0, err
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
			return 0, err
		}
		txRecord := &models.PointsTransaction{
			LedgerID:     ledger.ID,
			Source:       models.TxSourceClaim,
			Delta:        -deduct,
			BalanceAfter: newLedgerBalance,
		}
		if err := tx.Create(txRecord).Error; err != nil {
			return 0, err
		}
		remaining -= deduct
	}

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
