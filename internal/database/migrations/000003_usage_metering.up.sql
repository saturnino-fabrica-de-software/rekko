-- Create plans table
CREATE TABLE IF NOT EXISTS plans (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    monthly_price DECIMAL(10, 2) NOT NULL,
    quota_registrations INTEGER NOT NULL,
    quota_verifications INTEGER NOT NULL,
    overage_price DECIMAL(10, 4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE plans IS 'Subscription plans with quotas and pricing';
COMMENT ON COLUMN plans.quota_verifications IS '-1 = unlimited';
COMMENT ON COLUMN plans.overage_price IS 'Price per unit when quota is exceeded';

-- Seed initial plans
INSERT INTO plans (id, name, monthly_price, quota_registrations, quota_verifications, overage_price) VALUES
('starter', 'Starter', 99.00, 1000, 500, 0.0500),
('pro', 'Pro', 499.00, 10000, 5000, 0.0200),
('enterprise', 'Enterprise', 1999.00, 50000, -1, 0.0100)
ON CONFLICT (id) DO NOTHING;

-- Create usage_daily table
CREATE TABLE IF NOT EXISTS usage_daily (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    registrations INTEGER NOT NULL DEFAULT 0,
    verifications INTEGER NOT NULL DEFAULT 0,
    liveness_checks INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, date)
);

CREATE INDEX idx_usage_daily_tenant_date ON usage_daily(tenant_id, date DESC);
CREATE INDEX idx_usage_daily_date ON usage_daily(date DESC);

COMMENT ON TABLE usage_daily IS 'Daily usage aggregation per tenant for metering and billing';
COMMENT ON COLUMN usage_daily.registrations IS 'Number of face registrations on this date';
COMMENT ON COLUMN usage_daily.verifications IS 'Number of face verifications on this date';
COMMENT ON COLUMN usage_daily.liveness_checks IS 'Number of liveness checks on this date';

-- Triggers for updated_at
CREATE TRIGGER update_usage_daily_updated_at BEFORE UPDATE ON usage_daily
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_plans_updated_at BEFORE UPDATE ON plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
