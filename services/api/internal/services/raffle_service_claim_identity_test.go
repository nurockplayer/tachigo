package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

// TestSubmitClaim_WinnerCanSubmit verifies that the draw winner can submit
// shipping info when their userID matches the entry's linked user.
func TestSubmitClaim_WinnerCanSubmit(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "claim_owner1@example.com")
	winnerID := seedUserWithEmail(t, db, "claim_winner1@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "claim_winner1_twitch")

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}

	input := ClaimInput{
		RecipientName: "得獎者本人",
		AddressLine1:  "台北市信義區信義路一段1號",
		City:          "台北市",
	}
	claim, err := svc.SubmitClaim(draw.ClaimTokenRaw, winnerID, input)
	if err != nil {
		t.Fatalf("SubmitClaim: expected success, got %v", err)
	}
	if claim == nil {
		t.Fatal("expected non-nil claim")
	}
}

// TestSubmitClaim_NonWinnerForbidden verifies that a user who is not the draw
// winner receives ErrClaimForbidden when attempting to submit shipping info.
func TestSubmitClaim_NonWinnerForbidden(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "claim_owner2@example.com")
	winnerID := seedUserWithEmail(t, db, "claim_winner2@example.com")
	otherID := seedUserWithEmail(t, db, "claim_other2@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, &winnerID, "claim_winner2_twitch")

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}

	input := ClaimInput{
		RecipientName: "非得獎者",
		AddressLine1:  "台北市中正區重慶南路一段122號",
		City:          "台北市",
	}
	_, err = svc.SubmitClaim(draw.ClaimTokenRaw, otherID, input)
	if !errors.Is(err, ErrClaimForbidden) {
		t.Errorf("expected ErrClaimForbidden, got %v", err)
	}
}

// TestSubmitClaim_NilUserIDForbidden verifies that an entry without a linked
// user account (userID == nil) always returns ErrClaimForbidden.
func TestSubmitClaim_NilUserIDForbidden(t *testing.T) {
	db := newTestDB(t)
	ownerID := seedUserWithEmail(t, db, "claim_owner3@example.com")
	raffleID := seedRaffle(t, db, ownerID)
	seedEntry(t, db, raffleID, nil, "anonymous_winner") // no linked account

	svc := &RaffleService{db: db}
	draw, err := svc.DrawNext(raffleID, ownerID)
	if err != nil {
		t.Fatalf("DrawNext: %v", err)
	}

	someUserID := uuid.New()
	_, err = svc.SubmitClaim(draw.ClaimTokenRaw, someUserID, ClaimInput{
		RecipientName: "任何人",
		AddressLine1:  "某地址",
		City:          "某市",
	})
	if !errors.Is(err, ErrClaimForbidden) {
		t.Errorf("expected ErrClaimForbidden for nil entry userID, got %v", err)
	}
}
