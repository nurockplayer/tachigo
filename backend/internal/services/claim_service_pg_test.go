//go:build integration

package services

import (
	"sync"
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
