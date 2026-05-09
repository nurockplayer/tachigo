package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

// seedUser creates a bare user row and returns its ID.
func seedUser(t *testing.T, svc *AddressService) uuid.UUID {
	t.Helper()
	id := uuid.New()
	svc.db.Create(&models.User{ID: id, Role: models.RoleViewer})
	return id
}

func minimalInput() AddressInput {
	return AddressInput{
		RecipientName: "John Doe",
		AddressLine1:  "123 Main St",
		City:          "Taipei",
	}
}

func failAddressBulkUpdate(t *testing.T, db *gorm.DB, name string, err error) {
	t.Helper()
	if callbackErr := db.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table != "shipping_addresses" {
			return
		}
		if _, ok := tx.Statement.Dest.(map[string]interface{}); !ok {
			return
		}
		tx.AddError(err)
	}); callbackErr != nil {
		t.Fatalf("register update callback: %v", callbackErr)
	}
}

func failAddressSave(t *testing.T, db *gorm.DB, name string, err error) {
	t.Helper()
	if callbackErr := db.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table != "shipping_addresses" {
			return
		}
		if _, ok := tx.Statement.Dest.(*models.ShippingAddress); !ok {
			return
		}
		tx.AddError(err)
	}); callbackErr != nil {
		t.Fatalf("register update callback: %v", callbackErr)
	}
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	addr, err := svc.Create(userID, minimalInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.RecipientName != "John Doe" {
		t.Errorf("recipient_name: want John Doe, got %s", addr.RecipientName)
	}
}

func TestCreate_DefaultCountryIsTW(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	input := minimalInput() // Country is empty string
	addr, err := svc.Create(userID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Country != "TW" {
		t.Errorf("country: want TW, got %s", addr.Country)
	}
}

func TestCreate_ExplicitCountry(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	input := minimalInput()
	input.Country = "JP"
	addr, err := svc.Create(userID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Country != "JP" {
		t.Errorf("country: want JP, got %s", addr.Country)
	}
}

func TestCreate_SetDefaultUnsetsOthers(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)

	// First address as default
	input := minimalInput()
	input.IsDefault = true
	first, _ := svc.Create(userID, input)

	// Second address also as default
	second, err := svc.Create(userID, AddressInput{
		RecipientName: "Jane",
		AddressLine1:  "456 Other St",
		City:          "Kaohsiung",
		IsDefault:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reload first address from DB
	var firstReloaded models.ShippingAddress
	db.First(&firstReloaded, "id = ?", first.ID)

	if firstReloaded.IsDefault {
		t.Error("first address should no longer be default")
	}
	if !second.IsDefault {
		t.Error("second address should be default")
	}
}

func TestCreate_DefaultClearFailureReturnsErrorAndDoesNotCreate(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)

	input := minimalInput()
	input.IsDefault = true
	if _, err := svc.Create(userID, input); err != nil {
		t.Fatalf("seed default address: %v", err)
	}

	clearErr := errors.New("clear default failed")
	failAddressBulkUpdate(t, db, "fail_create_default_clear", clearErr)

	_, err := svc.Create(userID, input)
	if !errors.Is(err, clearErr) {
		t.Fatalf("expected clear default error, got %v", err)
	}

	var count int64
	if err := db.Model(&models.ShippingAddress{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count addresses: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected failed create to roll back, got %d addresses", count)
	}
}

// ─── List ────────────────────────────────────────────────────────────────────

func TestList_ReturnsUserAddresses(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	svc.Create(userID, minimalInput())
	svc.Create(userID, minimalInput())

	addrs, err := svc.List(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 2 {
		t.Errorf("want 2 addresses, got %d", len(addrs))
	}
}

func TestList_DefaultFirst(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	svc.Create(userID, minimalInput()) // not default

	input := minimalInput()
	input.IsDefault = true
	svc.Create(userID, input) // default

	addrs, err := svc.List(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) < 2 {
		t.Fatalf("expected 2 addresses, got %d", len(addrs))
	}
	if !addrs[0].IsDefault {
		t.Error("first address should be the default one")
	}
}

func TestList_IsolatedByUser(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	user1 := seedUser(t, svc)
	user2 := seedUser(t, svc)

	svc.Create(user1, minimalInput())

	addrs, _ := svc.List(user2)
	if len(addrs) != 0 {
		t.Errorf("user2 should see 0 addresses, got %d", len(addrs))
	}
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	addr, _ := svc.Create(userID, minimalInput())

	updated, err := svc.Update(userID, addr.ID, AddressInput{
		RecipientName: "Updated Name",
		AddressLine1:  "789 New St",
		City:          "Tainan",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.RecipientName != "Updated Name" {
		t.Errorf("recipient_name: want Updated Name, got %s", updated.RecipientName)
	}
	if updated.City != "Tainan" {
		t.Errorf("city: want Tainan, got %s", updated.City)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	_, err := svc.Update(userID, uuid.New(), minimalInput())
	if err != ErrAddressNotFound {
		t.Errorf("want ErrAddressNotFound, got %v", err)
	}
}

func TestUpdate_WrongUser(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	user1 := seedUser(t, svc)
	user2 := seedUser(t, svc)

	addr, _ := svc.Create(user1, minimalInput())

	_, err := svc.Update(user2, addr.ID, minimalInput())
	if err != ErrAddressNotFound {
		t.Errorf("want ErrAddressNotFound, got %v", err)
	}
}

func TestUpdate_DefaultClearFailureReturnsErrorAndRollsBack(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)

	defaultInput := minimalInput()
	defaultInput.IsDefault = true
	currentDefault, err := svc.Create(userID, defaultInput)
	if err != nil {
		t.Fatalf("create default address: %v", err)
	}
	target, err := svc.Create(userID, minimalInput())
	if err != nil {
		t.Fatalf("create target address: %v", err)
	}

	clearErr := errors.New("clear other defaults failed")
	failAddressBulkUpdate(t, db, "fail_update_default_clear", clearErr)

	_, err = svc.Update(userID, target.ID, AddressInput{
		RecipientName: "Target",
		AddressLine1:  "456 Other St",
		City:          "Taipei",
		IsDefault:     true,
	})
	if !errors.Is(err, clearErr) {
		t.Fatalf("expected clear default error, got %v", err)
	}

	var reloadedCurrent models.ShippingAddress
	if err := db.First(&reloadedCurrent, "id = ?", currentDefault.ID).Error; err != nil {
		t.Fatalf("load current default: %v", err)
	}
	if !reloadedCurrent.IsDefault {
		t.Fatal("existing default should remain default after rollback")
	}
	var reloadedTarget models.ShippingAddress
	if err := db.First(&reloadedTarget, "id = ?", target.ID).Error; err != nil {
		t.Fatalf("load target address: %v", err)
	}
	if reloadedTarget.IsDefault || reloadedTarget.RecipientName == "Target" {
		t.Fatal("target address should not be saved after clear failure")
	}
}

// ─── Delete ──────────────────────────────────────────────────────────────────

func TestDelete_Success(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	addr, _ := svc.Create(userID, minimalInput())

	if err := svc.Delete(userID, addr.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	addrs, _ := svc.List(userID)
	if len(addrs) != 0 {
		t.Errorf("expected 0 addresses after delete, got %d", len(addrs))
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	err := svc.Delete(userID, uuid.New())
	if err != ErrAddressNotFound {
		t.Errorf("want ErrAddressNotFound, got %v", err)
	}
}

func TestDelete_WrongUser(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	user1 := seedUser(t, svc)
	user2 := seedUser(t, svc)

	addr, _ := svc.Create(user1, minimalInput())

	err := svc.Delete(user2, addr.ID)
	if err != ErrAddressNotFound {
		t.Errorf("want ErrAddressNotFound, got %v", err)
	}
}

func TestDelete_ReturnsDBErrorBeforeNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)
	addr, err := svc.Create(userID, minimalInput())
	if err != nil {
		t.Fatalf("create address: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err = svc.Delete(userID, addr.ID)
	if err == nil {
		t.Fatal("expected DB error, got nil")
	}
	if errors.Is(err, ErrAddressNotFound) {
		t.Fatalf("expected DB error before ErrAddressNotFound, got %v", err)
	}
}

// ─── SetDefault ──────────────────────────────────────────────────────────────

func TestSetDefault_Success(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)

	addr1, _ := svc.Create(userID, minimalInput())
	addr2, _ := svc.Create(userID, minimalInput())

	result, err := svc.SetDefault(userID, addr2.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsDefault {
		t.Error("addr2 should be default")
	}

	// addr1 should no longer be default
	var reloaded models.ShippingAddress
	db.First(&reloaded, "id = ?", addr1.ID)
	if reloaded.IsDefault {
		t.Error("addr1 should no longer be default")
	}
}

func TestSetDefault_NotFound(t *testing.T) {
	svc := NewAddressService(newTestDB(t))
	userID := seedUser(t, svc)

	_, err := svc.SetDefault(userID, uuid.New())
	if err != ErrAddressNotFound {
		t.Errorf("want ErrAddressNotFound, got %v", err)
	}
}

func TestSetDefault_ClearFailureReturnsErrorAndRollsBack(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)

	defaultInput := minimalInput()
	defaultInput.IsDefault = true
	currentDefault, err := svc.Create(userID, defaultInput)
	if err != nil {
		t.Fatalf("create default address: %v", err)
	}
	target, err := svc.Create(userID, minimalInput())
	if err != nil {
		t.Fatalf("create target address: %v", err)
	}

	clearErr := errors.New("clear other defaults failed")
	failAddressBulkUpdate(t, db, "fail_set_default_clear", clearErr)

	_, err = svc.SetDefault(userID, target.ID)
	if !errors.Is(err, clearErr) {
		t.Fatalf("expected clear default error, got %v", err)
	}

	var reloadedCurrent models.ShippingAddress
	if err := db.First(&reloadedCurrent, "id = ?", currentDefault.ID).Error; err != nil {
		t.Fatalf("load current default: %v", err)
	}
	if !reloadedCurrent.IsDefault {
		t.Fatal("existing default should remain default after rollback")
	}
	var reloadedTarget models.ShippingAddress
	if err := db.First(&reloadedTarget, "id = ?", target.ID).Error; err != nil {
		t.Fatalf("load target address: %v", err)
	}
	if reloadedTarget.IsDefault {
		t.Fatal("target address should not become default after clear failure")
	}
}

func TestSetDefault_SaveFailureReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := NewAddressService(db)
	userID := seedUser(t, svc)

	addr, err := svc.Create(userID, minimalInput())
	if err != nil {
		t.Fatalf("create address: %v", err)
	}

	saveErr := errors.New("save default failed")
	failAddressSave(t, db, "fail_set_default_save", saveErr)

	_, err = svc.SetDefault(userID, addr.ID)
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected save error, got %v", err)
	}
}
