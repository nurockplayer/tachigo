-- migrations/013_airdrop.sql
-- Adds channel-level daily airdrop limit configuration.

ALTER TABLE channel_configs
    ADD COLUMN IF NOT EXISTS daily_airdrop_limit INT NOT NULL DEFAULT 5000;
