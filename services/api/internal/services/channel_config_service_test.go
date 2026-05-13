package services

import (
	"context"
	"errors"
	"testing"

	"github.com/tachigo/tachigo/internal/models"
	"gorm.io/gorm"
)

type channelConfigContextKey struct{}

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

func TestGetContext_UsesRequestContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewChannelConfigService(db)
	ctx := context.WithValue(context.Background(), channelConfigContextKey{}, "request")

	const callbackName = "test:channel_config_get_context"
	seenContext := false
	if err := db.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "channel_configs" {
			return
		}
		seenContext = tx.Statement.Context.Value(channelConfigContextKey{}) == "request"
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	defer func() {
		if err := db.Callback().Query().Remove(callbackName); err != nil {
			t.Fatalf("remove query callback: %v", err)
		}
	}()

	cfg, err := svc.GetContext(ctx, "ch_missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("want nil config, got %+v", cfg)
	}
	if !seenContext {
		t.Fatal("expected GetContext to pass request context to GORM")
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

func TestUpdateChannelConfigContext_CanceledContext(t *testing.T) {
	db := newTestDB(t)
	svc := NewChannelConfigService(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.UpdateChannelConfigContext(ctx, "ch_canceled", 30, 2)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}

	var count int64
	if err := db.Model(&models.ChannelConfig{}).Where("channel_id = ?", "ch_canceled").Count(&count).Error; err != nil {
		t.Fatalf("count channel configs: %v", err)
	}
	if count != 0 {
		t.Fatalf("canceled update should not persist config, got count %d", count)
	}
}
