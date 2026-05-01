package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/config"
	contractpkg "github.com/tachigo/tachigo/internal/contract"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrSpendAmountInvalid       = errors.New("spend amount must be greater than zero")
	ErrSpendInsufficientBalance = errors.New("insufficient tachi balance")
	ErrSpendWalletNotLinked     = errors.New("web3 wallet not linked")
	ErrSpendContractConfig      = errors.New("spend contract config is incomplete")
	ErrTachiyaRedeemFailed      = errors.New("tachiya coupon redeem failed after successful burn")
)

// BurnCaller abstracts the on-chain burn call; replaced with a mock in tests.
type BurnCaller interface {
	BurnOnChain(ctx context.Context, fromAddr string, amount int64) (txHash string, err error)
}

type SpendService struct {
	db            *gorm.DB
	contractCfg   config.ContractConfig
	tachiToken    *contractpkg.TachiToken
	burnCaller    BurnCaller
	tachiyaClient TachiyaClient
}

type spendReservation struct {
	fromAddr   string
	amount     int64
	newBalance int64
}

func NewSpendService(db *gorm.DB, contractCfg config.ContractConfig, ethClient *ethclient.Client, tachiyaClient TachiyaClient) *SpendService {
	svc := &SpendService{
		db:            db,
		contractCfg:   contractCfg,
		tachiyaClient: tachiyaClient,
	}
	if ethClient != nil && contractCfg.TachiContractAddress != "" && contractCfg.SepoliaSignerKey != "" {
		if common.IsHexAddress(contractCfg.TachiContractAddress) {
			t, err := contractpkg.NewTachiToken(common.HexToAddress(contractCfg.TachiContractAddress), ethClient)
			if err == nil {
				svc.tachiToken = t
			}
		}
	}
	svc.burnCaller = svc
	return svc
}

// SetBurnCallerForTest replaces the burn caller; use only in tests.
func (s *SpendService) SetBurnCallerForTest(bc BurnCaller) { s.burnCaller = bc }

// Redeem burns `amount` $TACHI from the user's on-chain wallet and deducts
// the same amount from tachi_balances. Returns the new balance and voucher code.
// If the Tachiya coupon call fails after a successful burn, returns
// ErrTachiyaRedeemFailed and records the attempt as "compensation-needed".
func (s *SpendService) Redeem(ctx context.Context, userID uuid.UUID, couponID string, amount int64) (int64, string, error) {
	if amount <= 0 {
		return 0, "", ErrSpendAmountInvalid
	}

	var reservation spendReservation
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		reservation, err = s.reserveSpend(tx, userID, amount)
		return err
	}); err != nil {
		return 0, "", err
	}

	burnCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	txHash, err := s.burnCaller.BurnOnChain(burnCtx, reservation.fromAddr, reservation.amount)
	if err != nil {
		if txHash != "" {
			// Tx was broadcast but receipt is unknown (e.g. context deadline, RPC error).
			// Do NOT roll back: the chain may have already burned the tokens.
			return 0, "", fmt.Errorf("burn tx broadcast (txHash=%s) but receipt unknown: %w", txHash, err)
		}
		rollbackErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.rollbackSpendReservation(tx, userID, reservation.amount)
		})
		if rollbackErr != nil {
			return 0, "", fmt.Errorf("%w; rollback spend reservation: %v", err, rollbackErr)
		}
		return 0, "", err
	}

	rec := &models.CouponRedemption{
		UserID:   userID,
		CouponID: couponID,
		Amount:   amount,
		TxHash:   txHash,
		Status:   models.CouponRedemptionPending,
	}
	if createErr := s.db.Create(rec).Error; createErr != nil {
		log.Printf("warning: failed to create coupon_redemption record coupon_id=%s user_id=%s: %v",
			couponID, userID, createErr)
		rec = nil
	}

	if s.tachiyaClient == nil {
		if rec != nil {
			if err := s.db.Model(rec).Updates(map[string]interface{}{
				"status":     models.CouponRedemptionRedeemed,
				"updated_at": time.Now(),
			}).Error; err != nil {
				log.Printf("warning: failed to mark coupon_redemption redeemed id=%s: %v", rec.ID, err)
			}
		}
		return reservation.newBalance, "", nil
	}

	tachiyaCtx, tachiyaCancel := context.WithTimeout(ctx, 10*time.Second)
	defer tachiyaCancel()
	voucherCode, tachiyaErr := s.tachiyaClient.RedeemCoupon(tachiyaCtx, couponID, reservation.amount)
	if tachiyaErr != nil {
		if rec != nil {
			errMsg := tachiyaErr.Error()
			if err := s.db.Model(rec).Updates(map[string]interface{}{
				"status":        models.CouponRedemptionCompensationNeeded,
				"error_message": errMsg,
				"updated_at":    time.Now(),
			}).Error; err != nil {
				log.Printf("warning: failed to mark coupon_redemption compensation-needed id=%s: %v", rec.ID, err)
			}
		}
		return 0, "", fmt.Errorf("%w (coupon_id=%s): %v", ErrTachiyaRedeemFailed, couponID, tachiyaErr)
	}

	if rec != nil {
		if err := s.db.Model(rec).Updates(map[string]interface{}{
			"status":       models.CouponRedemptionRedeemed,
			"voucher_code": voucherCode,
			"updated_at":   time.Now(),
		}).Error; err != nil {
			log.Printf("warning: failed to persist redeemed voucher id=%s: %v", rec.ID, err)
		}
	}

	return reservation.newBalance, voucherCode, nil
}

func (s *SpendService) reserveSpend(tx *gorm.DB, userID uuid.UUID, amount int64) (spendReservation, error) {
	var tb models.TachiBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&tb).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return spendReservation{}, ErrSpendInsufficientBalance
		}
		return spendReservation{}, err
	}
	if tb.Balance < amount {
		return spendReservation{}, ErrSpendInsufficientBalance
	}

	fromAddr, err := s.resolveWalletAddress(tx, userID)
	if err != nil {
		return spendReservation{}, err
	}

	newBalance := tb.Balance - amount
	if err := tx.Model(&models.TachiBalance{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance":    newBalance,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return spendReservation{}, err
	}

	return spendReservation{
		fromAddr:   fromAddr,
		amount:     amount,
		newBalance: newBalance,
	}, nil
}

func (s *SpendService) rollbackSpendReservation(tx *gorm.DB, userID uuid.UUID, amount int64) error {
	result := tx.Model(&models.TachiBalance{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance":    gorm.Expr("balance + ?", amount),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("rollback found no balance row for user %s", userID)
	}
	return nil
}

func (s *SpendService) resolveWalletAddress(db *gorm.DB, userID uuid.UUID) (string, error) {
	var authProvider models.AuthProvider
	err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).
		Order("created_at ASC").
		First(&authProvider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrSpendWalletNotLinked
		}
		return "", err
	}
	if !common.IsHexAddress(authProvider.ProviderID) {
		return "", fmt.Errorf("invalid linked wallet address: %s", authProvider.ProviderID)
	}
	return common.HexToAddress(authProvider.ProviderID).Hex(), nil
}

// BurnOnChain implements BurnCaller using the real TachiToken contract.
func (s *SpendService) BurnOnChain(ctx context.Context, fromAddr string, amount int64) (string, error) {
	if s.tachiToken == nil {
		return "", ErrSpendContractConfig
	}
	if !common.IsHexAddress(fromAddr) {
		return "", fmt.Errorf("invalid wallet address: %s", fromAddr)
	}
	if amount <= 0 {
		return "", ErrSpendAmountInvalid
	}

	signerKey, err := parseSignerKey(s.contractCfg.SepoliaSignerKey)
	if err != nil {
		return "", err
	}

	return s.tachiToken.Burn(ctx, common.HexToAddress(fromAddr), tachiWholeTokensToRawUnits(amount), signerKey)
}
