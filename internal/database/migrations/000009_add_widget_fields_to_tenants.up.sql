-- Add widget support fields to tenants table
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS public_key VARCHAR(64) UNIQUE,
    ADD COLUMN IF NOT EXISTS allowed_domains TEXT[];

-- Create index on public_key for fast widget authentication
CREATE INDEX IF NOT EXISTS idx_tenants_public_key ON tenants(public_key) WHERE public_key IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN tenants.public_key IS 'Public key for widget authentication (format: pk_live_xxx or pk_test_xxx)';
COMMENT ON COLUMN tenants.allowed_domains IS 'List of allowed domains for CORS and widget embedding';
