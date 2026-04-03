-- migrations/004_channel_config.sql
-- Per-channel watch-time earning configuration.

CREATE TABLE IF NOT EXISTS channel_configs (
    channel_id        VARCHAR(255) PRIMARY KEY,
    seconds_per_point BIGINT       NOT NULL DEFAULT 60,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
