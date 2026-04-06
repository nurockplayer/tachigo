CREATE TABLE IF NOT EXISTS agency_streamers (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agency_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id   VARCHAR     NOT NULL REFERENCES streamers(channel_id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (agency_id, channel_id)
);
