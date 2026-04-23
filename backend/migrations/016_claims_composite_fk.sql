-- Strengthens cross-table consistency for claim_items by enforcing
-- claim-user-ledger-transaction alignment with composite foreign keys.

CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_id_user_id
    ON claims (id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_id_user_id
    ON points_ledgers (id, user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_transactions_id_ledger_id
    ON points_transactions (id, ledger_id);

ALTER TABLE claim_items
    ADD COLUMN IF NOT EXISTS claim_user_id UUID;

UPDATE claim_items ci
SET claim_user_id = c.user_id
FROM claims c
WHERE ci.claim_id = c.id
  AND ci.claim_user_id IS NULL;

ALTER TABLE claim_items
    ALTER COLUMN claim_user_id SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_claim_items_claim_user'
    ) THEN
        ALTER TABLE claim_items
            ADD CONSTRAINT fk_claim_items_claim_user
            FOREIGN KEY (claim_id, claim_user_id)
            REFERENCES claims (id, user_id)
            ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_claim_items_ledger_user'
    ) THEN
        ALTER TABLE claim_items
            ADD CONSTRAINT fk_claim_items_ledger_user
            FOREIGN KEY (ledger_id, claim_user_id)
            REFERENCES points_ledgers (id, user_id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_claim_items_tx_ledger'
    ) THEN
        ALTER TABLE claim_items
            ADD CONSTRAINT fk_claim_items_tx_ledger
            FOREIGN KEY (points_transaction_id, ledger_id)
            REFERENCES points_transactions (id, ledger_id);
    END IF;
END $$;
