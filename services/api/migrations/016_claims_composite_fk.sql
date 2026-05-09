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

DO $$
DECLARE
    missing_claim_count INTEGER;
    missing_ledger_count INTEGER;
    missing_tx_count INTEGER;
    claim_ledger_user_mismatch_count INTEGER;
    tx_ledger_mismatch_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO missing_claim_count
    FROM claim_items ci
    LEFT JOIN claims c ON c.id = ci.claim_id
    WHERE c.id IS NULL;

    SELECT COUNT(*) INTO missing_ledger_count
    FROM claim_items ci
    LEFT JOIN points_ledgers pl ON pl.id = ci.ledger_id
    WHERE pl.id IS NULL;

    SELECT COUNT(*) INTO missing_tx_count
    FROM claim_items ci
    LEFT JOIN points_transactions pt ON pt.id = ci.points_transaction_id
    WHERE pt.id IS NULL;

    SELECT COUNT(*) INTO claim_ledger_user_mismatch_count
    FROM claim_items ci
    JOIN claims c ON c.id = ci.claim_id
    JOIN points_ledgers pl ON pl.id = ci.ledger_id
    WHERE c.user_id <> pl.user_id;

    SELECT COUNT(*) INTO tx_ledger_mismatch_count
    FROM claim_items ci
    JOIN points_transactions pt ON pt.id = ci.points_transaction_id
    WHERE pt.ledger_id <> ci.ledger_id;

    IF missing_claim_count > 0
       OR missing_ledger_count > 0
       OR missing_tx_count > 0
       OR claim_ledger_user_mismatch_count > 0
       OR tx_ledger_mismatch_count > 0 THEN
        RAISE EXCEPTION
            'migration 016 blocked: inconsistent claim_items rows detected (missing_claim=%, missing_ledger=%, missing_tx=%, claim_ledger_user_mismatch=%, tx_ledger_mismatch=%). resolve data before re-running migration',
            missing_claim_count,
            missing_ledger_count,
            missing_tx_count,
            claim_ledger_user_mismatch_count,
            tx_ledger_mismatch_count;
    END IF;
END $$;

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
