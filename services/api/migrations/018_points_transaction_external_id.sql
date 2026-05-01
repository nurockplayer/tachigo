-- migrations/018_points_transaction_external_id.sql
-- Adds external_transaction_id to points_transactions for Bits receipt idempotency.

ALTER TABLE points_transactions
  ADD COLUMN IF NOT EXISTS external_transaction_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_transactions_external_transaction_id
  ON points_transactions (external_transaction_id)
  WHERE external_transaction_id IS NOT NULL;
