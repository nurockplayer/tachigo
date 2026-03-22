package services

import (
	"testing"

	"github.com/google/uuid"

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
