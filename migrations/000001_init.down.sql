-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS faces;
DROP TABLE IF EXISTS tenants;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Note: We don't drop extensions as they might be used by other databases
-- DROP EXTENSION IF EXISTS "vector";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
