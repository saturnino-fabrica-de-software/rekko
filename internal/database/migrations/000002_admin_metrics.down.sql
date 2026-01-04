-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS cache_entries CASCADE;
DROP TABLE IF EXISTS webhook_queue CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS alert_history CASCADE;
DROP TABLE IF EXISTS alerts CASCADE;
DROP TABLE IF EXISTS metrics_aggregated CASCADE;
DROP TABLE IF EXISTS admin_users CASCADE;

-- Drop function
DROP FUNCTION IF EXISTS cleanup_expired_cache();
