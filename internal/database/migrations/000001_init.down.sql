-- Drop triggers
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_api_keys_environment;
DROP INDEX IF EXISTS idx_api_keys_tenant;
DROP INDEX IF EXISTS idx_api_keys_hash;
DROP INDEX IF EXISTS idx_tenants_active;
DROP INDEX IF EXISTS idx_tenants_slug;

-- Drop tables
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS tenants;

-- Drop extensions
DROP EXTENSION IF EXISTS "pgcrypto";
