-- Ocean character progression and per-streamer familiarity state.
-- Runtime behavior stays in services/handlers; this migration only adds schema.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS active_character VARCHAR(50) NOT NULL DEFAULT 'crab',
    ADD COLUMN IF NOT EXISTS switch_cooldown_until TIMESTAMPTZ;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_users_active_character'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT chk_users_active_character
            CHECK (active_character IN ('crab', 'dolphin', 'turtle', 'whale', 'capybara'));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS user_characters (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    character   VARCHAR(50)  NOT NULL,
    unlocked    BOOLEAN      NOT NULL DEFAULT false,
    stage       BIGINT       NOT NULL DEFAULT 1,
    xp          BIGINT       NOT NULL DEFAULT 0,
    unlocked_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_characters_character
        CHECK (character IN ('crab', 'dolphin', 'turtle', 'whale', 'capybara')),
    CONSTRAINT chk_user_characters_stage CHECK (stage >= 1 AND stage <= 3),
    CONSTRAINT chk_user_characters_xp CHECK (xp >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_characters_user_character
    ON user_characters (user_id, character);

CREATE TABLE IF NOT EXISTS streamer_familiarities (
    id                         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id                 VARCHAR(255) NOT NULL,
    cumulative_watch_seconds   BIGINT       NOT NULL DEFAULT 0,
    last_watched_at            TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_streamer_familiarities_watch_seconds
        CHECK (cumulative_watch_seconds >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_streamer_familiarities_user_channel
    ON streamer_familiarities (user_id, channel_id);
