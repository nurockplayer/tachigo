-- migrations/004_rbac_roles.sql
-- Add agency role to user_role enum.
-- Idempotent: safe to run multiple times.

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
