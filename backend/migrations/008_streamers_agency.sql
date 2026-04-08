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

-- Fail fast if any streamer/channel maps to multiple agencies in legacy data.
-- One streamer can belong to at most one agency (business rule).
DO $$
DECLARE conflict_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO conflict_count
    FROM (
        SELECT channel_id
        FROM agency_streamers
        GROUP BY channel_id
        HAVING COUNT(DISTINCT agency_id) > 1
    ) conflicts;
    IF conflict_count > 0 THEN
        RAISE EXCEPTION 'agency backfill conflict: % channel(s) map to multiple agencies in agency_streamers; resolve before deploying', conflict_count;
    END IF;
END $$;

-- Backfill agency_user_id from legacy agency_streamers table.
UPDATE streamers s
SET agency_user_id = ag.agency_id
FROM agency_streamers ag
WHERE ag.channel_id = s.channel_id
  AND s.agency_user_id IS NULL;
