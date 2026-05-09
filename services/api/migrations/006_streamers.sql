CREATE TABLE IF NOT EXISTS streamers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id   VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_streamers_channel_id ON streamers(channel_id);
