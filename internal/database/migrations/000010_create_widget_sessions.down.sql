-- Drop widget_sessions cleanup function
DROP FUNCTION IF EXISTS delete_expired_widget_sessions();

-- Drop indexes
DROP INDEX IF EXISTS idx_widget_sessions_tenant_expires;
DROP INDEX IF EXISTS idx_widget_sessions_expires_at;
DROP INDEX IF EXISTS idx_widget_sessions_tenant_id;

-- Drop table
DROP TABLE IF EXISTS widget_sessions;
