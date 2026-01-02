-- ========================================
-- Seed Test Tenant for Development
-- ========================================
-- This script creates a test tenant for manual endpoint testing
--
-- API Key: test-api-key-rekko-dev
-- Hash: 4a526689e4037a92c11b6228a6aea1a26247875054585e177228365b9720e770
--
-- Usage:
--   psql $DATABASE_URL -f scripts/seed_test_tenant.sql
--
-- Or with Docker:
--   docker compose exec postgres psql -U postgres -d rekko -f /scripts/seed_test_tenant.sql
--
-- ========================================

BEGIN;

-- Clean up existing test tenant (idempotent)
DELETE FROM usage_records WHERE tenant_id IN (SELECT id FROM tenants WHERE name = 'Test Tenant');
DELETE FROM verifications WHERE tenant_id IN (SELECT id FROM tenants WHERE name = 'Test Tenant');
DELETE FROM faces WHERE tenant_id IN (SELECT id FROM tenants WHERE name = 'Test Tenant');
DELETE FROM tenants WHERE name = 'Test Tenant';

-- Insert test tenant
-- UUID fixo para facilitar debug: 00000000-0000-0000-0000-000000000001
INSERT INTO tenants (
    id,
    name,
    api_key_hash,
    settings,
    is_active,
    created_at,
    updated_at
)
VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Test Tenant',
    '4a526689e4037a92c11b6228a6aea1a26247875054585e177228365b9720e770', -- SHA-256 hash of "test-api-key-rekko-dev"
    '{
        "verification_threshold": 0.8,
        "max_faces_per_user": 5,
        "liveness_required": false,
        "retention_days": 90
    }'::jsonb,
    true,
    NOW(),
    NOW()
)
ON CONFLICT (api_key_hash) DO UPDATE SET
    name = EXCLUDED.name,
    settings = EXCLUDED.settings,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();

-- Verify insertion
SELECT
    id,
    name,
    is_active,
    settings,
    created_at
FROM tenants
WHERE name = 'Test Tenant';

COMMIT;

-- ========================================
-- Success Message
-- ========================================
\echo '✓ Test tenant created successfully!'
\echo ''
\echo 'Tenant Details:'
\echo '  ID: 00000000-0000-0000-0000-000000000001'
\echo '  Name: Test Tenant'
\echo '  API Key: test-api-key-rekko-dev'
\echo ''
\echo 'Example cURL commands (multipart form-data):'
\echo ''
\echo '# Register a face'
\echo 'curl -X POST http://localhost:3000/v1/faces \'
\echo '  -H "Authorization: Bearer test-api-key-rekko-dev" \'
\echo '  -F "external_id=user-123" \'
\echo '  -F "image=@/path/to/face.jpg"'
\echo ''
\echo '# Verify a face'
\echo 'curl -X POST http://localhost:3000/v1/faces/verify \'
\echo '  -H "Authorization: Bearer test-api-key-rekko-dev" \'
\echo '  -F "external_id=user-123" \'
\echo '  -F "image=@/path/to/face.jpg"'
\echo ''
\echo '# Delete a face'
\echo 'curl -X DELETE http://localhost:3000/v1/faces/user-123 \'
\echo '  -H "Authorization: Bearer test-api-key-rekko-dev"'
\echo ''
