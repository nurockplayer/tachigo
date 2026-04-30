package models

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestClaimStatusValues(t *testing.T) {
	tests := []struct {
		name string
		got  ClaimStatus
		want string
	}{
		{name: "pending", got: ClaimStatusPending, want: "pending"},
		{name: "broadcast", got: ClaimStatusBroadcast, want: "broadcast"},
		{name: "confirmed", got: ClaimStatusConfirmed, want: "confirmed"},
		{name: "failed", got: ClaimStatusFailed, want: "failed"},
		{name: "finalize_failed", got: ClaimStatusFinalizeFailed, want: "finalize_failed"},
	}

	for _, tc := range tests {
		if string(tc.got) != tc.want {
			t.Fatalf("%s: want %q, got %q", tc.name, tc.want, tc.got)
		}
	}
}

func TestClaimBeforeCreate_UsesFallbackUUIDWhenV7Fails(t *testing.T) {
	orig := uuidV7Func
	uuidV7Func = func() (uuid.UUID, error) {
		return uuid.Nil, errors.New("boom")
	}
	t.Cleanup(func() { uuidV7Func = orig })

	c := &Claim{}
	if err := c.BeforeCreate(nil); err != nil {
		t.Fatalf("before create: %v", err)
	}
	if c.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected non-nil UUID fallback")
	}
}

func TestClaimItemBeforeCreate_UsesFallbackUUIDWhenV7Fails(t *testing.T) {
	orig := uuidV7Func
	uuidV7Func = func() (uuid.UUID, error) {
		return uuid.Nil, errors.New("boom")
	}
	t.Cleanup(func() { uuidV7Func = orig })

	c := &ClaimItem{}
	if err := c.BeforeCreate(nil); err != nil {
		t.Fatalf("before create: %v", err)
	}
	if c.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected non-nil UUID fallback")
	}
}
