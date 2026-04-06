-- migrations/006_click_boost.sql
-- Adds click-boost cooldown tracking to watch_sessions.
-- GORM AutoMigrate handles this automatically in dev.
-- Use this for manual DB setup or production migrations.

ALTER TABLE watch_sessions
  ADD COLUMN IF NOT EXISTS click_cooldown_until TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00';
