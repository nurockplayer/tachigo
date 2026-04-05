-- migrations/007_points_transaction_sku.sql
-- Adds optional SKU metadata to points_transactions.
-- GORM AutoMigrate handles this automatically in dev.
-- Use this for manual DB setup or production migrations.

ALTER TABLE points_transactions
  ADD COLUMN IF NOT EXISTS sku VARCHAR(255);
