-- Enable pgcrypto if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Admin users for super admin (JWT auth)
CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'super_admin',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_users_email ON admin_users(email) WHERE is_active = true;

-- Aggregated metrics (pre-computed)
CREATE TABLE IF NOT EXISTS metrics_aggregated (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    aggregation_type VARCHAR(20) NOT NULL, -- 'sum', 'avg', 'count', 'p99', 'min', 'max'
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, metric_name, aggregation_type, period_start)
);

CREATE INDEX idx_metrics_tenant_period ON metrics_aggregated(tenant_id, period_start, period_end);
CREATE INDEX idx_metrics_name ON metrics_aggregated(metric_name);
CREATE INDEX idx_metrics_type ON metrics_aggregated(aggregation_type);
CREATE INDEX idx_metrics_period ON metrics_aggregated(period_start, period_end);

-- Alerts configuration
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    conditions JSONB NOT NULL,
    window_seconds INTEGER NOT NULL DEFAULT 300,
    cooldown_seconds INTEGER NOT NULL DEFAULT 3600,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    channels JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_tenant ON alerts(tenant_id);
CREATE INDEX idx_alerts_enabled ON alerts(tenant_id, enabled) WHERE enabled = true;

-- Alert history
CREATE TABLE IF NOT EXISTS alert_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'triggered' CHECK (status IN ('triggered', 'acknowledged', 'resolved')),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_history_alert ON alert_history(alert_id);
CREATE INDEX idx_alert_history_tenant ON alert_history(tenant_id, triggered_at DESC);
CREATE INDEX idx_alert_history_status ON alert_history(status) WHERE status = 'triggered';

-- Webhooks configuration
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    events JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_tenant ON webhooks(tenant_id);
CREATE INDEX idx_webhooks_enabled ON webhooks(tenant_id, enabled) WHERE enabled = true;

-- Webhook delivery queue
CREATE TABLE IF NOT EXISTS webhook_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'delivered', 'failed')),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_queue_status ON webhook_queue(status, next_retry_at) WHERE status IN ('pending', 'processing');
CREATE INDEX idx_webhook_queue_webhook ON webhook_queue(webhook_id);

-- PostgreSQL-native cache table
CREATE TABLE IF NOT EXISTS cache_entries (
    key VARCHAR(255) PRIMARY KEY,
    value BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cache_expires ON cache_entries(expires_at);

-- Trigger to update updated_at on admin_users
CREATE TRIGGER update_admin_users_updated_at BEFORE UPDATE ON admin_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update updated_at on alerts
CREATE TRIGGER update_alerts_updated_at BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update updated_at on webhooks
CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update updated_at on webhook_queue
CREATE TRIGGER update_webhook_queue_updated_at BEFORE UPDATE ON webhook_queue
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM cache_entries WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE admin_users IS 'Super admin users with JWT authentication';
COMMENT ON TABLE metrics_aggregated IS 'Pre-aggregated metrics for dashboard performance';
COMMENT ON TABLE alerts IS 'Alert configuration for monitoring thresholds';
COMMENT ON TABLE alert_history IS 'History of triggered alerts';
COMMENT ON TABLE webhooks IS 'Webhook endpoints for event notifications';
COMMENT ON TABLE webhook_queue IS 'Queue for webhook delivery with retry logic';
COMMENT ON TABLE cache_entries IS 'PostgreSQL-native cache with TTL support';

COMMENT ON COLUMN metrics_aggregated.aggregation_type IS 'Type of aggregation: sum, avg, count, p99, min, max';
COMMENT ON COLUMN metrics_aggregated.metadata IS 'Additional metadata like labels, tags, etc.';
COMMENT ON COLUMN alerts.conditions IS 'JSON with threshold conditions, e.g., {"metric": "api_errors", "operator": ">", "value": 100}';
COMMENT ON COLUMN alerts.channels IS 'JSON array of notification channels, e.g., ["webhook", "email"]';
COMMENT ON COLUMN webhooks.events IS 'JSON array of event types to subscribe to';
COMMENT ON COLUMN webhook_queue.status IS 'Delivery status: pending, processing, delivered, failed';
