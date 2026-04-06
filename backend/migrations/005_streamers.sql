CREATE TABLE IF NOT EXISTS streamers (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agency_user_id  UUID        REFERENCES users(id),
    twitch_login    VARCHAR(50) NOT NULL UNIQUE,
    display_name    VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_streamers_agency_user_id ON streamers (agency_user_id);
