-- Verification queries for Rekko database schema
-- Run this after migrations to verify everything is set up correctly

-- 1. Verify extensions
\echo '=== Checking Extensions ==='
SELECT extname, extversion FROM pg_extension WHERE extname IN ('uuid-ossp', 'vector');

-- 2. Verify tables
\echo '\n=== Checking Tables ==='
SELECT table_name, table_type
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_name;

-- 3. Verify indexes
\echo '\n=== Checking Indexes ==='
SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
ORDER BY tablename, indexname;

-- 4. Verify foreign keys
\echo '\n=== Checking Foreign Keys ==='
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    tc.constraint_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
    AND ccu.table_schema = tc.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
    AND tc.table_schema = 'public'
ORDER BY tc.table_name, kcu.column_name;

-- 5. Verify triggers
\echo '\n=== Checking Triggers ==='
SELECT
    trigger_name,
    event_manipulation,
    event_object_table,
    action_statement
FROM information_schema.triggers
WHERE trigger_schema = 'public'
ORDER BY event_object_table, trigger_name;

-- 6. Verify column types for vector
\echo '\n=== Checking Vector Columns ==='
SELECT
    table_name,
    column_name,
    data_type,
    udt_name
FROM information_schema.columns
WHERE table_schema = 'public'
    AND column_name = 'embedding';

-- 7. Test UUID generation
\echo '\n=== Testing UUID Generation ==='
SELECT uuid_generate_v4() AS test_uuid;

-- 8. Table row counts
\echo '\n=== Table Row Counts ==='
SELECT
    'tenants' AS table_name,
    COUNT(*) AS row_count
FROM tenants
UNION ALL
SELECT 'faces', COUNT(*) FROM faces
UNION ALL
SELECT 'verifications', COUNT(*) FROM verifications
UNION ALL
SELECT 'usage_records', COUNT(*) FROM usage_records;

-- 9. Verify unique constraints
\echo '\n=== Checking Unique Constraints ==='
SELECT
    tc.table_name,
    tc.constraint_name,
    kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
WHERE tc.constraint_type = 'UNIQUE'
    AND tc.table_schema = 'public'
ORDER BY tc.table_name, tc.constraint_name;

-- 10. Database size
\echo '\n=== Database Size ==='
SELECT
    pg_database.datname,
    pg_size_pretty(pg_database_size(pg_database.datname)) AS size
FROM pg_database
WHERE datname = current_database();

\echo '\n=== Verification Complete ==='
