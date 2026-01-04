-- Remove widget support fields from tenants table
DROP INDEX IF EXISTS idx_tenants_public_key;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS public_key,
    DROP COLUMN IF EXISTS allowed_domains;
