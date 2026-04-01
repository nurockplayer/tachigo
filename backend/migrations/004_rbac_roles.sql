-- migrations/004_rbac_roles.sql
-- Add agency / engineering roles; migrate legacy admin → engineering.
-- Idempotent: safe to run multiple times.

-- 1. Add new enum values (PostgreSQL requires ALTER TYPE; no-op if already exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum
        WHERE enumlabel = 'agency'
          AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'user_role')
    ) THEN
        ALTER TYPE user_role ADD VALUE 'agency';
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum
        WHERE enumlabel = 'engineering'
          AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'user_role')
    ) THEN
        ALTER TYPE user_role ADD VALUE 'engineering';
    END IF;
END$$;

-- 2. Migrate existing admin users to engineering
UPDATE users SET role = 'engineering' WHERE role = 'admin';
