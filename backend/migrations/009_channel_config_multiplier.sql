-- migrations/009_channel_config_multiplier.sql
-- Adds channel-level point multiplier support.
-- GORM AutoMigrate handles this automatically in dev.
-- Use this for manual DB setup or production migrations.

ALTER TABLE channel_configs ADD COLUMN IF NOT EXISTS multiplier INT NOT NULL DEFAULT 1;
