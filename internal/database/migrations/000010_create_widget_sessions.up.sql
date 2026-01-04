-- Create widget_sessions table for temporary session management
CREATE TABLE IF NOT EXISTS widget_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    origin VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_widget_sessions_tenant_id ON widget_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_widget_sessions_expires_at ON widget_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_widget_sessions_tenant_expires ON widget_sessions(tenant_id, expires_at);

-- Add comments for documentation
COMMENT ON TABLE widget_sessions IS 'Temporary sessions for widget authentication and CORS validation';
COMMENT ON COLUMN widget_sessions.origin IS 'Domain that created the session (for CORS validation)';
COMMENT ON COLUMN widget_sessions.expires_at IS 'Session expiration timestamp (typically 1 hour from creation)';

-- Create a function to automatically delete expired sessions
CREATE OR REPLACE FUNCTION delete_expired_widget_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM widget_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Note: Scheduled cleanup can be done via cron job or application-level scheduler
-- Example: SELECT delete_expired_widget_sessions();
