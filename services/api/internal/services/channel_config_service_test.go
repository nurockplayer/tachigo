package services

import (
	"testing"

	"github.com/tachigo/tachigo/internal/models"
)

func TestGet_NoConfig(t *testing.T) {
	svc := NewChannelConfigService(newTestDB(t))

	cfg, err := svc.Get("ch_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("want nil config, got %+v", cfg)
	}
}

func TestGet_OK(t *testing.T) {
	db := newTestDB(t)
	svc := NewChannelConfigService(db)
	if err := db.Create(&models.ChannelConfig{
		ChannelID:       "ch_abc",
		SecondsPerPoint: 45,
		Multiplier:      3,
	}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cfg, err := svc.Get("ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.SecondsPerPoint != 45 || cfg.Multiplier != 3 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestEffectiveMultiplier_Default(t *testing.T) {
	svc := NewChannelConfigService(newTestDB(t))

	multiplier, err := svc.EffectiveMultiplier("ch_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if multiplier != 1 {
		t.Fatalf("want 1, got %d", multiplier)
	}
}

func TestEffectiveMultiplier_OK(t *testing.T) {
	db := newTestDB(t)
	svc := NewChannelConfigService(db)
	if err := db.Create(&models.ChannelConfig{
		ChannelID:       "ch_abc",
		SecondsPerPoint: 60,
		Multiplier:      4,
	}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	multiplier, err := svc.EffectiveMultiplier("ch_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if multiplier != 4 {
		t.Fatalf("want 4, got %d", multiplier)
	}
}

func TestUpdateChannelConfig_MultiplierOnly(t *testing.T) {
	db := newTestDB(t)
	svc := NewChannelConfigService(db)
	if err := db.Create(&models.ChannelConfig{
		ChannelID:       "ch_abc",
		SecondsPerPoint: 90,
		Multiplier:      1,
	}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}

	cfg, err := svc.UpdateChannelConfig("ch_abc", 0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SecondsPerPoint != 90 || cfg.Multiplier != 5 {
		t.Fatalf("unexpected config after update: %+v", cfg)
	}
}

func TestUpdateChannelConfig_BothFields(t *testing.T) {
	svc := NewChannelConfigService(newTestDB(t))

	cfg, err := svc.UpdateChannelConfig("ch_abc", 30, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SecondsPerPoint != 30 || cfg.Multiplier != 2 {
		t.Fatalf("unexpected config after update: %+v", cfg)
	}
}
