-- Extends claim status with finalize_failed to track cases where the chain tx
-- succeeded but the DB finalization step failed, enabling recovery without
-- re-crediting the on-chain mint.

DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    FOR constraint_name IN
        SELECT c.conname
        FROM pg_constraint c
        WHERE c.conrelid = 'claims'::regclass
          AND c.contype = 'c'
          AND pg_get_constraintdef(c.oid) LIKE '%status%'
          AND pg_get_constraintdef(c.oid) LIKE '%pending%'
          AND pg_get_constraintdef(c.oid) LIKE '%broadcast%'
          AND pg_get_constraintdef(c.oid) LIKE '%confirmed%'
          AND pg_get_constraintdef(c.oid) LIKE '%failed%'
    LOOP
        -- atlas:nolint
        EXECUTE format('ALTER TABLE claims DROP CONSTRAINT %I', constraint_name);
    END LOOP;
END $$;

-- atlas:nolint
ALTER TABLE claims DROP CONSTRAINT IF EXISTS claims_status_check;
-- atlas:nolint
ALTER TABLE claims DROP CONSTRAINT IF EXISTS chk_claim_status;

ALTER TABLE claims
    ADD CONSTRAINT chk_claim_status
    CHECK (status IN ('pending', 'broadcast', 'confirmed', 'failed', 'finalize_failed'));

ALTER TABLE claims
    ADD COLUMN IF NOT EXISTS finalize_failed_at TIMESTAMPTZ;
