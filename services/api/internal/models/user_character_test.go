package models

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCharacterKindValues(t *testing.T) {
	tests := []struct {
		name string
		got  CharacterKind
		want string
	}{
		{name: "crab", got: CharacterCrab, want: "crab"},
		{name: "dolphin", got: CharacterDolphin, want: "dolphin"},
		{name: "turtle", got: CharacterTurtle, want: "turtle"},
		{name: "whale", got: CharacterWhale, want: "whale"},
		{name: "capybara", got: CharacterCapybara, want: "capybara"},
	}

	for _, tc := range tests {
		if string(tc.got) != tc.want {
			t.Fatalf("%s: want %q, got %q", tc.name, tc.want, tc.got)
		}
	}
}

func TestUserCharacterBeforeCreate_UsesFallbackUUIDWhenV7Fails(t *testing.T) {
	orig := uuidV7Func
	uuidV7Func = func() (uuid.UUID, error) {
		return uuid.Nil, errors.New("boom")
	}
	t.Cleanup(func() { uuidV7Func = orig })

	c := &UserCharacter{}
	if err := c.BeforeCreate(nil); err != nil {
		t.Fatalf("before create: %v", err)
	}
	if c.ID == uuid.Nil {
		t.Fatalf("expected non-nil UUID fallback")
	}
}

func TestStreamerFamiliarityBeforeCreate_UsesFallbackUUIDWhenV7Fails(t *testing.T) {
	orig := uuidV7Func
	uuidV7Func = func() (uuid.UUID, error) {
		return uuid.Nil, errors.New("boom")
	}
	t.Cleanup(func() { uuidV7Func = orig })

	f := &StreamerFamiliarity{}
	if err := f.BeforeCreate(nil); err != nil {
		t.Fatalf("before create: %v", err)
	}
	if f.ID == uuid.Nil {
		t.Fatalf("expected non-nil UUID fallback")
	}
}
