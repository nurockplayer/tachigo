-- migrations/011_streamers_agency.sql
-- Backfills and constrains streamers.agency_user_id.

ALTER TABLE streamers ADD COLUMN IF NOT EXISTS agency_user_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_streamers_agency_user_id'
    ) THEN
        ALTER TABLE streamers
        ADD CONSTRAINT fk_streamers_agency_user_id
        FOREIGN KEY (agency_user_id) REFERENCES users(id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_streamers_agency_user_id ON streamers (agency_user_id);
