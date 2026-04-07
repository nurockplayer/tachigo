ALTER TABLE streamers ADD COLUMN IF NOT EXISTS agency_user_id UUID REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_streamers_agency_user_id ON streamers (agency_user_id);
