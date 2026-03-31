-- migrations/003_watch_points.sql
-- Reference schema for watch-time points system — Phase 1 (data layer).
-- GORM AutoMigrate handles this automatically in dev.
-- Use this for manual DB setup or production migrations.
-- All statements are idempotent (IF NOT EXISTS).

-- ---------------------------------------------------------------------------
-- watch_sessions
-- Tracks a viewer's active or completed session per channel.
--
-- Session lifecycle:
--   active  : is_active = TRUE,  ended_at = NULL
--   finished: is_active = FALSE, ended_at = <timestamp>
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS watch_sessions (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    twitch_user_id      VARCHAR(255) NOT NULL,
    channel_id          VARCHAR(255) NOT NULL,
    accumulated_seconds BIGINT       NOT NULL DEFAULT 0,
    rewarded_seconds    BIGINT       NOT NULL DEFAULT 0,
    last_heartbeat_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    is_active           BOOLEAN      NOT NULL DEFAULT TRUE,
    ended_at            TIMESTAMPTZ  NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_watch_sessions_twitch_user_id ON watch_sessions (twitch_user_id);
CREATE INDEX IF NOT EXISTS idx_watch_sessions_channel_id     ON watch_sessions (channel_id);
CREATE INDEX IF NOT EXISTS idx_watch_sessions_is_active      ON watch_sessions (is_active);

-- Partial unique index: only one active session per (twitch_user_id, channel_id).
CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_sessions_active_user_channel
    ON watch_sessions (twitch_user_id, channel_id)
    WHERE is_active = TRUE;

-- ---------------------------------------------------------------------------
-- points_ledgers
-- Single, platform-wide balance per viewer.
-- Table name is "points_ledgers" (GORM plural) — not "points_ledger".
-- Points are NOT scoped to a channel; they are a platform currency.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS points_ledgers (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    twitch_user_id    VARCHAR(255) NOT NULL UNIQUE,
    cumulative_total  BIGINT       NOT NULL DEFAULT 0,
    spendable_balance BIGINT       NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- points_transactions
-- Immutable log of every balance change.
--
-- watch_session_id rules by source:
--   "watch_time" → always non-null (links to the rewarding session)
--   "bits"       → always null
--   "spend"      → always null
--
-- No FK constraint on watch_session_id — sessions may be archived
-- independently without orphaning transaction history.
--
-- MVP: no CHECK constraint on source; can be added later:
--   CHECK (source IN ('watch_time', 'bits', 'spend'))
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS points_transactions (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_id        UUID         NOT NULL REFERENCES points_ledgers(id),
    watch_session_id UUID         NULL,
    source           VARCHAR(50)  NOT NULL,
    delta            BIGINT       NOT NULL,
    balance_after    BIGINT       NOT NULL,
    note             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_points_transactions_ledger_id        ON points_transactions (ledger_id);
CREATE INDEX IF NOT EXISTS idx_points_transactions_watch_session_id ON points_transactions (watch_session_id);
