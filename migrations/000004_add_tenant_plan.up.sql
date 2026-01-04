-- Add plan_id column to tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan_id VARCHAR(50) DEFAULT 'starter' REFERENCES plans(id);

CREATE INDEX IF NOT EXISTS idx_tenants_plan_id ON tenants(plan_id);

COMMENT ON COLUMN tenants.plan_id IS 'Reference to the plan this tenant is subscribed to';
