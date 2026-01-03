-- ========================================
-- Seed de Desenvolvimento para Rekko
-- ========================================
-- ATENÇÃO: NÃO use em produção!
--
-- Este script cria um tenant de desenvolvimento com uma API Key fixa
-- para facilitar testes locais sem precisar gerar keys a cada vez.
--
-- API Key: rekko_test_devdevdevdevdevdevdevdevdevdev00
-- Hash: adf716ab3ebb2a1138973de4a44fe454c05c0d070e897fc55220af74807b25ae
--
-- Usage:
--   ./scripts/db.sh seed
--
-- Or manually:
--   psql $DATABASE_URL -f scripts/seed_dev.sql
--
-- ========================================

BEGIN;

-- Limpar dados existentes do tenant de desenvolvimento (idempotente)
DELETE FROM api_keys WHERE tenant_id IN (
    SELECT id FROM tenants WHERE slug = 'rekko-dev'
);
DELETE FROM tenants WHERE slug = 'rekko-dev';

-- Inserir tenant de desenvolvimento
-- UUID fixo para facilitar debug: 00000000-0000-0000-0000-000000000001
INSERT INTO tenants (
    id,
    name,
    slug,
    is_active,
    plan,
    settings,
    created_at,
    updated_at
)
VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Rekko Development',
    'rekko-dev',
    true,
    'enterprise',
    '{
        "rate_limit": 1000,
        "features": ["liveness", "search", "bulk_import"],
        "verification_threshold": 0.8,
        "max_faces_per_user": 10,
        "liveness_required": false,
        "retention_days": 90,
        "webhook_url": null
    }'::jsonb,
    NOW(),
    NOW()
);

-- Inserir API Key de desenvolvimento (test environment)
INSERT INTO api_keys (
    id,
    tenant_id,
    name,
    key_hash,
    key_prefix,
    environment,
    is_active,
    created_at
)
VALUES (
    '00000000-0000-0000-0000-000000000002'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Development Key',
    'adf716ab3ebb2a1138973de4a44fe454c05c0d070e897fc55220af74807b25ae', -- SHA-256 of "rekko_test_devdevdevdevdevdevdevdevdevdev00"
    'rekko_test_devd',
    'test',
    true,
    NOW()
);

-- Verificar inserção
SELECT
    t.id as tenant_id,
    t.name,
    t.slug,
    t.plan,
    t.is_active,
    ak.key_prefix,
    ak.environment,
    ak.name as key_name
FROM tenants t
JOIN api_keys ak ON ak.tenant_id = t.id
WHERE t.slug = 'rekko-dev';

COMMIT;

-- ========================================
-- Mensagens de Sucesso
-- ========================================
\echo ''
\echo '✅ Tenant de desenvolvimento criado com sucesso!'
\echo ''
\echo '╔══════════════════════════════════════════════════════════════════╗'
\echo '║                   REKKO DEVELOPMENT TENANT                       ║'
\echo '╚══════════════════════════════════════════════════════════════════╝'
\echo ''
\echo 'Tenant Details:'
\echo '  ID: 00000000-0000-0000-0000-000000000001'
\echo '  Name: Rekko Development'
\echo '  Slug: rekko-dev'
\echo '  Plan: Enterprise (todas as features habilitadas)'
\echo '  Rate Limit: 1000 req/s'
\echo ''
\echo 'API Key (use no header Authorization):'
\echo '  rekko_test_devdevdevdevdevdevdevdevdevdev00'
\echo ''
\echo '─────────────────────────────────────────────────────────────────'
\echo 'Exemplos de uso (cURL):'
\echo '─────────────────────────────────────────────────────────────────'
\echo ''
\echo '# 1. Registrar uma face'
\echo 'curl -X POST http://localhost:3000/v1/faces \'
\echo '  -H "Authorization: Bearer rekko_test_devdevdevdevdevdevdevdevdevdev00" \'
\echo '  -F "external_id=user-123" \'
\echo '  -F "image=@/path/to/face.jpg"'
\echo ''
\echo '# 2. Verificar uma face (1:1)'
\echo 'curl -X POST http://localhost:3000/v1/faces/verify \'
\echo '  -H "Authorization: Bearer rekko_test_devdevdevdevdevdevdevdevdevdev00" \'
\echo '  -F "external_id=user-123" \'
\echo '  -F "image=@/path/to/face.jpg"'
\echo ''
\echo '# 3. Buscar faces similares (1:N)'
\echo 'curl -X POST http://localhost:3000/v1/faces/search \'
\echo '  -H "Authorization: Bearer rekko_test_devdevdevdevdevdevdevdevdevdev00" \'
\echo '  -F "image=@/path/to/face.jpg" \'
\echo '  -F "threshold=0.8" \'
\echo '  -F "limit=10"'
\echo ''
\echo '# 4. Listar faces do usuário'
\echo 'curl -X GET http://localhost:3000/v1/faces/user-123 \'
\echo '  -H "Authorization: Bearer rekko_test_devdevdevdevdevdevdevdevdevdev00"'
\echo ''
\echo '# 5. Deletar face'
\echo 'curl -X DELETE http://localhost:3000/v1/faces/user-123 \'
\echo '  -H "Authorization: Bearer rekko_test_devdevdevdevdevdevdevdevdevdev00"'
\echo ''
\echo '─────────────────────────────────────────────────────────────────'
\echo ''
