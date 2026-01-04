-- Rate limiting table for sliding window algorithm
CREATE TABLE IF NOT EXISTS rate_limit_counters (
    key VARCHAR(255) PRIMARY KEY,
    count INTEGER NOT NULL DEFAULT 0,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for cleanup of expired windows
CREATE INDEX idx_rate_limit_window_end ON rate_limit_counters(window_end);

-- Index for tenant-specific queries
CREATE INDEX idx_rate_limit_tenant ON rate_limit_counters(tenant_id);

-- Trigger to update updated_at
CREATE TRIGGER update_rate_limit_counters_updated_at
    BEFORE UPDATE ON rate_limit_counters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE rate_limit_counters IS 'Rate limiting counters with sliding window for search endpoint';
COMMENT ON COLUMN rate_limit_counters.key IS 'Rate limit key format: search_rate:{tenant_id}';
COMMENT ON COLUMN rate_limit_counters.count IS 'Number of requests in current window';
COMMENT ON COLUMN rate_limit_counters.window_start IS 'Window start timestamp (now - window_duration)';
COMMENT ON COLUMN rate_limit_counters.window_end IS 'Window end timestamp (now)';
