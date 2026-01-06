-- Seed Demo Data for Rekko Dashboard
-- This script creates realistic demo data for presentations

-- Use existing tenant
DO $$
DECLARE
    tenant_uuid UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    face_id UUID;
    i INT;
    random_quality FLOAT;
    random_confidence FLOAT;
    random_latency FLOAT;
    random_verified BOOLEAN;
    random_date TIMESTAMP;
BEGIN
    -- Generate 49 additional faces (we already have 1)
    FOR i IN 1..49 LOOP
        face_id := gen_random_uuid();
        random_quality := 0.85 + (random() * 0.14); -- 0.85 to 0.99 (high quality!)
        random_date := NOW() - (random() * INTERVAL '30 days');

        INSERT INTO faces (
            id,
            tenant_id,
            external_id,
            embedding,
            quality_score,
            created_at,
            updated_at
        ) VALUES (
            face_id,
            tenant_uuid,
            'user_' || LPAD(i::text, 4, '0'),
            ARRAY(SELECT random() FROM generate_series(1, 512))::vector(512),
            random_quality,
            random_date,
            random_date
        );
    END LOOP;

    RAISE NOTICE 'Created 49 faces';
END $$;

-- Generate 500 verifications distributed over 30 days (IMPRESSIVE METRICS!)
DO $$
DECLARE
    tenant_uuid UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    face_ids UUID[];
    i INT;
    random_confidence FLOAT;
    random_latency FLOAT;
    random_verified BOOLEAN;
    random_date TIMESTAMP;
    selected_face_id UUID;
BEGIN
    -- Get all face IDs for this tenant
    SELECT ARRAY_AGG(id) INTO face_ids FROM faces WHERE tenant_id = tenant_uuid;

    -- Generate 500 verifications with IMPRESSIVE metrics for demo
    FOR i IN 1..500 LOOP
        -- Select random face
        selected_face_id := face_ids[1 + floor(random() * array_length(face_ids, 1))::int];

        -- HIGH confidence (0.88 to 0.99) - impressive for presentations!
        random_confidence := 0.88 + (random() * 0.11);

        -- 95% match rate (verified = true when confidence > 0.85)
        random_verified := random() < 0.95;

        random_date := NOW() - (random() * INTERVAL '30 days');

        -- ULTRA-LOW latency for impressive P99!
        -- 70% under 2ms, 25% between 2-4ms, only 5% between 4-4.8ms
        IF random() < 0.70 THEN
            random_latency := 0.5 + (random() * 1.5); -- 0.5ms to 2ms (70%)
        ELSIF random() < 0.95 THEN
            random_latency := 2 + (random() * 2); -- 2ms to 4ms (25%)
        ELSE
            random_latency := 4 + (random() * 0.8); -- 4ms to 4.8ms (5%) - still under 5ms target!
        END IF;

        INSERT INTO verifications (
            id,
            tenant_id,
            face_id,
            external_id,
            verified,
            confidence,
            latency_ms,
            created_at
        ) VALUES (
            gen_random_uuid(),
            tenant_uuid,
            selected_face_id,
            'user_' || LPAD(floor(random() * 50)::int::text, 4, '0'),
            random_verified,
            random_confidence,
            random_latency,
            random_date
        );
    END LOOP;

    RAISE NOTICE 'Created 500 verifications with impressive metrics';
END $$;

-- Create sample webhooks (all enabled for impressive demo!)
INSERT INTO webhooks (id, tenant_id, name, url, secret, events, enabled, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Slack Notifications', 'https://example.com/webhook/slack', 'whsec_demo123456789', '["face.registered", "face.verified"]'::jsonb, true, NOW() - INTERVAL '15 days', NOW()),
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Internal Analytics', 'https://analytics.example.com/webhook', 'whsec_analytics987654321', '["face.registered", "face.verified", "face.deleted"]'::jsonb, true, NOW() - INTERVAL '10 days', NOW()),
    (gen_random_uuid(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Security Audit', 'https://security.example.com/audit', 'whsec_security456789123', '["face.search.match"]'::jsonb, true, NOW() - INTERVAL '5 days', NOW())
ON CONFLICT DO NOTHING;

-- Summary report
SELECT
    (SELECT COUNT(*) FROM faces WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as total_faces,
    (SELECT COUNT(*) FROM verifications WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as total_verifications,
    (SELECT COUNT(*) FROM webhooks WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as total_webhooks,
    (SELECT AVG(quality_score)::numeric(4,2) FROM faces WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as avg_quality,
    (SELECT AVG(confidence)::numeric(4,2) FROM verifications WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as avg_confidence,
    (SELECT AVG(latency_ms)::numeric(4,2) FROM verifications WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as avg_latency_ms,
    (SELECT PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms)::numeric(4,2) FROM verifications WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11') as p99_latency_ms,
    (SELECT COUNT(*) * 100.0 / NULLIF((SELECT COUNT(*) FROM verifications WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'), 0) FROM verifications WHERE tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11' AND verified = true)::numeric(4,1) as match_rate_percent;
