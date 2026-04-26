//go:build integration

package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

func TestClaim_FinalizeIdempotentConcurrent(t *testing.T) {
	db := newPGTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	claimID := newUUID()
	txHash := "0xconcurrentHash"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, broadcast_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW(), NOW())
	`, claimID, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusBroadcast, txHash).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	reservation := claimReservation{
		claimID: claimID,
		userID:  userID,
		amount:  60,
	}

	errs := make(chan error, 2)
	bals := make(chan int64, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := db.Transaction(func(tx *gorm.DB) error {
				bal, err := svc.finalizeClaim(tx, reservation, txHash)
				if err == nil {
					bals <- bal
				}
				return err
			})
			errs <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errs)
	close(bals)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent finalizeClaim unexpected error: %v", err)
		}
	}
	for bal := range bals {
		if bal != 60 {
			t.Fatalf("expected concurrent balance=60, got %d", bal)
		}
	}

	var claim models.Claim
	if err := db.Where("id = ?", claimID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&count).Error; err != nil {
		t.Fatalf("count balances: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 tachi balance row, got %d", count)
	}

	var balance int64
	if err := db.Raw("SELECT CAST(balance AS BIGINT) AS balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&balance).Error; err != nil {
		t.Fatalf("query tachi balance: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", balance)
	}
}

func TestClaim_FinalizeConcurrent_AFailsAtBalanceUpsert_BSucceeds(t *testing.T) {
	db := newPGTestDB(t)
	hookReady := make(chan struct{})
	hookProceed := make(chan struct{})
	var hookCalls atomic.Int32
	svc := &ClaimService{
		db: db,
		testAfterClaimUpdate: func() error {
			if hookCalls.Add(1) != 1 {
				return nil
			}
			hookReady <- struct{}{}
			<-hookProceed
			return errors.New("forced balance upsert failure")
		},
	}
	userID := userIDForClaim(t, db)
	claimID := newUUID()
	txHash := "0xconcurrentRollbackHash"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, broadcast_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW(), NOW())
	`, claimID, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusBroadcast, txHash).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	reservation := claimReservation{
		claimID: claimID,
		userID:  userID,
		amount:  60,
	}

	errs := make(chan error, 2)
	bals := make(chan int64, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := db.Transaction(func(tx *gorm.DB) error {
				bal, err := svc.finalizeClaim(tx, reservation, txHash)
				if err == nil {
					bals <- bal
				}
				return err
			})
			errs <- err
		}()
	}

	close(start)
	<-hookReady
	close(hookProceed)
	wg.Wait()
	close(errs)
	close(bals)

	var failedCount, successCount int
	for err := range errs {
		if err != nil {
			if !strings.Contains(err.Error(), "forced balance upsert failure") {
				t.Fatalf("unexpected finalizeClaim error: %v", err)
			}
			failedCount++
			continue
		}
		successCount++
	}
	if failedCount != 1 || successCount != 1 {
		t.Fatalf("expected 1 failed and 1 successful finalize, got failed=%d success=%d", failedCount, successCount)
	}
	for bal := range bals {
		if bal != 60 {
			t.Fatalf("expected successful balance=60, got %d", bal)
		}
	}

	var claim models.Claim
	if err := db.Where("id = ?", claimID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&count).Error; err != nil {
		t.Fatalf("count balances: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 tachi balance row, got %d", count)
	}

	var balance int64
	if err := db.Raw("SELECT CAST(balance AS BIGINT) AS balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&balance).Error; err != nil {
		t.Fatalf("query tachi balance: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", balance)
	}
}

func TestClaim_FinalizeConcurrent_TxHashMismatch_NeverCredits(t *testing.T) {
	db := newPGTestDB(t)
	hookReady := make(chan struct{})
	hookProceed := make(chan struct{})
	var hookCalls atomic.Int32
	svc := &ClaimService{
		db: db,
		testAfterClaimUpdate: func() error {
			if hookCalls.Add(1) != 1 {
				return nil
			}
			hookReady <- struct{}{}
			<-hookProceed
			return nil
		},
	}
	userID := userIDForClaim(t, db)
	claimID := newUUID()

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, broadcast_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NULL, NOW(), NOW(), NOW())
	`, claimID, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusBroadcast).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	reservation := claimReservation{
		claimID: claimID,
		userID:  userID,
		amount:  60,
	}

	aDone := make(chan error, 1)
	bDone := make(chan error, 1)

	go func() {
		aDone <- db.Transaction(func(tx *gorm.DB) error {
			bal, err := svc.finalizeClaim(tx, reservation, "0xAAAA")
			if err != nil {
				return err
			}
			if bal != 60 {
				return fmt.Errorf("expected first finalize balance=60, got %d", bal)
			}
			return nil
		})
	}()

	<-hookReady
	go func() {
		bDone <- db.Transaction(func(tx *gorm.DB) error {
			_, err := svc.finalizeClaim(tx, reservation, "0xBBBB")
			return err
		})
	}()
	close(hookProceed)

	if err := <-aDone; err != nil {
		t.Fatalf("first finalizeClaim: %v", err)
	}
	err := <-bDone
	if err == nil || !strings.Contains(err.Error(), "tx_hash mismatch") {
		t.Fatalf("expected tx_hash mismatch error, got %v", err)
	}

	var claim models.Claim
	if err := db.Where("id = ?", claimID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}
	if claim.TxHash == nil || *claim.TxHash != "0xAAAA" {
		t.Fatalf("expected claim tx_hash=0xAAAA, got %v", claim.TxHash)
	}

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&count).Error; err != nil {
		t.Fatalf("count balances: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 tachi balance row, got %d", count)
	}

	var balance int64
	if err := db.Raw("SELECT CAST(balance AS BIGINT) AS balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&balance).Error; err != nil {
		t.Fatalf("query tachi balance: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", balance)
	}
}

func TestClaim_LateMarkFinalizeFailedIsNoOp_AfterConfirmed(t *testing.T) {
	db := newPGTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	claimID := newUUID()
	txHash := "0xlateFinalizeFailedNoOp"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, broadcast_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW(), NOW())
	`, claimID, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusBroadcast, txHash).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	reservation := claimReservation{
		claimID: claimID,
		userID:  userID,
		amount:  60,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		_, err := svc.finalizeClaim(tx, reservation, txHash)
		return err
	}); err != nil {
		t.Fatalf("finalizeClaim: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return svc.markFinalizeFailedClaim(tx, reservation, txHash, errors.New("late finalize failure"))
	}); err != nil {
		t.Fatalf("markFinalizeFailedClaim unexpected error: %v", err)
	}

	var claim models.Claim
	if err := db.Where("id = ?", claimID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}

	var balance int64
	if err := db.Raw("SELECT CAST(balance AS BIGINT) AS balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&balance).Error; err != nil {
		t.Fatalf("query tachi balance: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", balance)
	}
}

func TestClaim_Finalize_FromFinalizeFailedState_Succeeds(t *testing.T) {
	db := newPGTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	claimID := newUUID()
	txHash := "0xretryFinalizeFailed"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, broadcast_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW(), NOW())
	`, claimID, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusBroadcast, txHash).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}
	if err := db.Exec("UPDATE claims SET status = ? WHERE id = ?", models.ClaimStatusFinalizeFailed, claimID).Error; err != nil {
		t.Fatalf("set finalize_failed: %v", err)
	}

	reservation := claimReservation{
		claimID: claimID,
		userID:  userID,
		amount:  60,
	}

	var finalBal int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		bal, err := svc.finalizeClaim(tx, reservation, txHash)
		if err != nil {
			return err
		}
		finalBal = bal
		return nil
	}); err != nil {
		t.Fatalf("finalizeClaim: %v", err)
	}
	if finalBal != 60 {
		t.Fatalf("expected retry finalize balance=60, got %d", finalBal)
	}

	var claim models.Claim
	if err := db.Where("id = ?", claimID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&count).Error; err != nil {
		t.Fatalf("count balances: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 tachi balance row, got %d", count)
	}

	var balance int64
	if err := db.Raw("SELECT CAST(balance AS BIGINT) AS balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&balance).Error; err != nil {
		t.Fatalf("query tachi balance: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", balance)
	}
}
